package server

import (
  "strings"
  "net/http"
  "encoding/xml"

  "caldav/data"
)

func HandleREPORT(writer http.ResponseWriter, request *http.Request, requestBody string) {
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
  xml.Unmarshal([]byte(requestBody), &requestXML)

  urlResource, found, err := storage.GetResource(request.URL.Path)
  if !found {
    respond(http.StatusNotFound, "", writer)
    return
  } else if err != nil {
    respondWithError(err, writer)
    return
  }

  resourcesToReport, err := fetchResourcesByList(urlResource, requestXML.Hrefs)
  if err != nil {
    respondWithError(err, writer)
    return
  }

  multistatus := NewMultistatusResp()
  // for each href, build the multistatus responses
  for _, r := range resourcesToReport {
    propstats := multistatus.Propstats(r.resource, requestXML.Prop.Tags)
    multistatus.AddResponse(r.href, r.found, propstats)
  }

  respond(207, multistatus.ToXML(), writer)
}


// Wraps a resource that has to be reported. Basically it contains
// the original requested `href`, the actual `resource` (can be nil)
// and if the `resource` was `found` or not
type reportRes struct {
  href string
  resource *data.Resource
  found bool
}

// The hrefs can come from the request URL (in this case will be only one) or from the request body itself.
// The one in the URL will have priority (see RFC4791#section-7.9).
func fetchResourcesByList(origin *data.Resource, requestedPaths []string) ([]reportRes, error) {
  storage := new(data.FileStorage)

  reps := []reportRes{}

  // if origin resource is a collection, we have to check if each requested path belongs to it
  if origin.IsCollection() {
    for _, path := range requestedPaths {
      // if the requested path does not belong to the origin collection, skip
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
