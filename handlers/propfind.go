package handlers

import (
  "net/http"
  "encoding/xml"
  "git.samedi.cc/ferraz/caldav/data"
  "git.samedi.cc/ferraz/caldav/global"
)

type propfindHandler struct {
  request *http.Request
  requestBody string
  writer http.ResponseWriter
}

func (ph propfindHandler) Handle()  {
  // get the target resources based on the request URL
  resources, err := global.Storage.GetResources(ph.request.URL.Path, getDepth(ph.request))
  if err != nil {
    if err == data.ErrResourceNotFound {
      respond(http.StatusNotFound, "", ph.writer)
      return
    }
    respondWithError(err, ph.writer)
    return
  }

  // read body string to xml struct
  type XMLProp2 struct {
    Tags []xml.Name `xml:",any"`
  }
  type XMLRoot2 struct {
    XMLName xml.Name
    Prop    XMLProp2  `xml:"DAV: prop"`
  }
  var requestXML XMLRoot2
  xml.Unmarshal([]byte(ph.requestBody), &requestXML)

  multistatus := newMultistatusResp()
  // for each href, build the multistatus responses
  for _, resource := range resources {
    propstats := multistatus.Propstats(&resource, requestXML.Prop.Tags)
    multistatus.AddResponse(resource.Path, true, propstats)
  }

  respond(207, multistatus.ToXML(), ph.writer)
}
