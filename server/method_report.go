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

func (rh ReportHandler) Handle() {
  // TODO: HANDLE FILTERS, DEPTH
  storage := new(data.FileStorage)

  // read body string to xml struct
  type XMLProp struct {
    Tags []xml.Name `xml:",any"`
  }
  type XMLRoot struct {
    XMLName xml.Name
    Prop    XMLProp  `xml:"DAV: prop"`
    Hrefs   []string `xml:"DAV: href"`
  }
  var requestXML XMLRoot
  xml.Unmarshal([]byte(rh.requestBody), &requestXML)

  urlResource, found, err := storage.GetResource(rh.request.URL.Path)
  if !found {
    respond(http.StatusNotFound, "", rh.writer)
    return
  } else if err != nil {
    respondWithError(err, rh.writer)
    return
  }

  resourcesToReport, err := rh.fetchResourcesByList(urlResource, requestXML.Hrefs)
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

// Wraps a resource that has to be reported. Basically it contains
// the original requested `href`, the actual `resource` (can be nil)
// and if the `resource` was `found` or not
type reportRes struct {
  href string
  resource *data.Resource
  found bool
}

// The hrefs can come from (1) the request URL or (2) from the request body itself.
// If the resource from the URL points to a collection (2), we will check the request body
// to get the requested `hrefs` (resource paths). Each requested href has to be related to the collection.
// The ones that are not, we simply ignore them.
// If the resource from the URL is NOT a collection (1) we process the the report only for this resource
// and ignore any othre requested hrefs that might be present in the request body.
// [See RFC4791#section-7.9]
func (rh ReportHandler) fetchResourcesByList(origin *data.Resource, requestedPaths []string) ([]reportRes, error) {
  storage := new(data.FileStorage)

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

// func fetchResourcesByFilters(origin *data.Resource, depth string, filters string) []reportRes {
//   return nil
// }
