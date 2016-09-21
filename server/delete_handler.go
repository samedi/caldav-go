package server

import (
  "net/http"
  "caldav/data"
)

type DeleteHandler struct {
  request *http.Request
  requestBody string
  writer http.ResponseWriter
}

func (dh DeleteHandler) Handle() {
  precond := RequestPreconditions{dh.request}

  // get the event from the storage
  resource, found, err := storage.GetResource(dh.request.URL.Path)
  if err != nil && err != data.ErrResourceNotFound {
    respondWithError(err, dh.writer)
    return
  }

  if !found {
    respond(http.StatusNotFound, "", dh.writer)
    return
  }

  // TODO: Handle delete on collections
  if resource.IsCollection() {
    respond(http.StatusMethodNotAllowed, "", dh.writer)
    return
  }

  // check ETag pre-condition
  resourceEtag, _ := resource.GetEtag()
  if !precond.IfMatch(resourceEtag) {
    respond(http.StatusPreconditionFailed, "", dh.writer)
    return
  }

  // delete event after pre-condition passed
  err = storage.DeleteResource(resource.Path)
  if err != nil {
    respondWithError(err, dh.writer)
    return
  }

  respond(http.StatusNoContent, "", dh.writer)
}
