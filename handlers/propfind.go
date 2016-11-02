package handlers

import (
  "net/http"
  "encoding/xml"
  "git.samedi.cc/ferraz/caldav/global"
)

type propfindHandler struct {
  request *http.Request
  response *Response
}

func (ph propfindHandler) Handle() *Response {
  requestBody := readRequestBody(ph.request)

  // get the target resources based on the request URL
  resources, err := global.Storage.GetResources(ph.request.URL.Path, parseResourceDepth(ph.request))
  if err != nil {
    return ph.response.SetError(err)
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
  xml.Unmarshal([]byte(requestBody), &requestXML)

  multistatus := newMultistatusResp()
  // for each href, build the multistatus responses
  for _, resource := range resources {
    propstats := multistatus.Propstats(&resource, requestXML.Prop.Tags)
    multistatus.AddResponse(resource.Path, true, propstats)
  }

  return ph.response.Set(207, multistatus.ToXML())
}
