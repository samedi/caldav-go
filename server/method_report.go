package server

import (
  "strings"
  "net/http"
  "encoding/xml"

  "caldav/data"
)

type ReportHandler struct{
  request *http.Request
  requestBody string
  writer http.ResponseWriter
}

// See more at RFC4791#section-7.1
func (rh ReportHandler) Handle() {
  urlResource, found, err := storage.GetResource(rh.request.URL.Path)
  if !found {
    respond(http.StatusNotFound, "", rh.writer)
    return
  } else if err != nil {
    respondWithError(err, rh.writer)
    return
  }

  // read body string to xml struct
  var requestXML reportRootXML
  xml.Unmarshal([]byte(rh.requestBody), &requestXML)

  // The resources to be reported are fetched by the type of the request. If it is
  // a `calendar-multiget`, the resources come based on a set of `hrefs` in the request body.
  // If it is a `calendar-query`, the resources are calculated based on set of filters in the request.
  var resourcesToReport []reportRes
  switch requestXML.XMLName {
  case xml.Name{Space:"urn:ietf:params:xml:ns:caldav", Local:"calendar-multiget"}:
    resourcesToReport, err = rh.fetchResourcesByList(urlResource, requestXML.Hrefs)
  case xml.Name{Space:"urn:ietf:params:xml:ns:caldav", Local:"calendar-query"}:
    resourcesToReport, err = rh.fetchResourcesByFilters(urlResource, requestXML.Filters)
  default:
    respond(http.StatusPreconditionFailed, "", rh.writer)
    return
  }

  if err != nil {
    respondWithError(err, rh.writer)
    return
  }

  multistatus := NewMultistatusResp()
  // for each href, build the multistatus responses
  for _, r := range resourcesToReport {
    propstats := multistatus.Propstats(r.resource, requestXML.Prop.Tags)
    multistatus.AddResponse(r.href, r.found, propstats)
  }

  respond(207, multistatus.ToXML(), rh.writer)
}

type reportPropXML struct {
  Tags []xml.Name `xml:",any"`
}

type reportRootXML struct {
  XMLName xml.Name
  Prop    reportPropXML  `xml:"DAV: prop"`
  Hrefs   []string       `xml:"DAV: href"`
  Filters []string       `xml:"urn:ietf:params:xml:ns:caldav: filter"`
}

// Wraps a resource that has to be reported, either fetched by filters or by a list.
// Basically it contains the original requested `href`, the actual `resource` (can be nil)
// and if the `resource` was `found` or not
type reportRes struct {
  href string
  resource *data.Resource
  found bool
}

// The resources are fetched based on the origin resource and a set of filters.
// If the origin resource is a collection, the filters are checked against each of the collection's resources
// to see if they match. The collection's resources that match the filters are returned. The ones that will be returned
// are the resources that were not found (does not exist) and the ones that matched the filters. The ones that did not
// match the filter will not appear in the response result.
// If the origin resource is not a collection, the function just returns it and ignore any filter processing.
// [See RFC4791#section-7.8]
func (rh ReportHandler) fetchResourcesByFilters(origin *data.Resource, filters []string) ([]reportRes, error) {
  // The list of resources that has to be reported back in the response.
  reps := []reportRes{}

  // TODO: update after filters implementation is finished
  dummyFilter := ResourceFilter{
    name: "C:comp-filter",
    attrs: map[string]string{"name": "VEVENT"},
  }

  if origin.IsCollection() {
    collectionContent, _ := origin.GetCollectionContentPaths()
    for _, path := range collectionContent {
      resource, found, err := storage.GetResource(path)
      if err != nil && err != data.ErrResourceNotFound {
        return nil, err
      }

      // onlye add it if the resource was not found or if the resource match the filters
      if !found || dummyFilter.Match(resource) {
        reps = append(reps, reportRes{path, resource, found})
      }
    }
  } else {
    // the origin resource is not a collection, so returns just that as the result
    reps = append(reps, reportRes{origin.Path, origin, true})
  }

  return reps, nil
}

// The hrefs can come from (1) the request URL or (2) from the request body itself.
// If the origin resource from the URL points to a collection (2), we will check the request body
// to get the requested `hrefs` (resource paths). Each requested href has to be related to the collection.
// The ones that are not, we simply ignore them.
// If the resource from the URL is NOT a collection (1) we process the the report only for this resource
// and ignore any othre requested hrefs that might be present in the request body.
// [See RFC4791#section-7.9]
func (rh ReportHandler) fetchResourcesByList(origin *data.Resource, requestedPaths []string) ([]reportRes, error) {
  reps := []reportRes{}

  // if origin resource is a collection, we have to check if each requested path belongs to it
  if origin.IsCollection() {
    for _, path := range requestedPaths {
      // if the requested path does not belong to the origin collection, skip
      // ('belonging' means that the path's prefix is the same as the collection path)
      if !strings.HasPrefix(path, origin.Path) {
        continue
      }

      resource, found, err := storage.GetResource(path)
      if err != nil && err != data.ErrResourceNotFound {
        return nil, err
      }

      reps = append(reps, reportRes{path, resource, found})
    }
  } else {
    reps = append(reps, reportRes{origin.Path, origin, true})
  }

  return reps, nil
}
