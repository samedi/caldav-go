package caldav

import (
  "net/http"
  "git.samedi.cc/ferraz/caldav/data"
)

type GetHandler struct {
  request *http.Request
  requestBody string
  writer http.ResponseWriter
  onlyHeaders bool
}

func (gh GetHandler) Handle() {
  resource, found, err := Storage.GetResource(gh.request.URL.Path)
  if err != nil && err != data.ErrResourceNotFound {
    respondWithError(err, gh.writer)
    return
  }

  if !found {
    respond(http.StatusNotFound, "", gh.writer)
    return
  }

  etag, _ := resource.GetEtag()
  gh.writer.Header().Set("ETag", etag)
  lastm, _ := resource.GetLastModified(http.TimeFormat)
  gh.writer.Header().Set("Last-Modified", lastm)
  ctype, _ := resource.GetContentType()
  gh.writer.Header().Set("Content-Type", ctype)

  var response string
  if gh.onlyHeaders {
    response = ""
  } else {
    response, _ = resource.GetContentData()
  }

  respond(http.StatusOK, response, gh.writer)
}
