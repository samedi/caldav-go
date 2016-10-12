package handlers

import (
  "net/http"
  "git.samedi.cc/ferraz/caldav/global"
)

type getHandler struct {
  request *http.Request
  response *Response
  onlyHeaders bool
}

func (gh getHandler) Handle() *Response {
  resource, _, err := global.Storage.GetResource(gh.request.URL.Path)
  if err != nil {
    return gh.response.SetError(err)
  }

  etag, _ := resource.GetEtag()
  gh.response.SetHeader("ETag", etag)
  lastm, _ := resource.GetLastModified(http.TimeFormat)
  gh.response.SetHeader("Last-Modified", lastm)
  ctype, _ := resource.GetContentType()
  gh.response.SetHeader("Content-Type", ctype)

  var response string
  if gh.onlyHeaders {
    response = ""
  } else {
    response, _ = resource.GetContentData()
  }

  return gh.response.Set(http.StatusOK, response)
}
