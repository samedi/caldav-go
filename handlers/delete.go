package handlers

import (
  "net/http"
  "git.samedi.cc/ferraz/caldav/data"
  "git.samedi.cc/ferraz/caldav/global"
)

type deleteHandler struct {
  request *http.Request
  writer http.ResponseWriter
}

func (dh deleteHandler) Handle() {
  precond := requestPreconditions{dh.request}

  // get the event from the storage
  resource, found, err := global.Storage.GetResource(dh.request.URL.Path)
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
  err = global.Storage.DeleteResource(resource.Path)
  if err != nil {
    respondWithError(err, dh.writer)
    return
  }

  respond(http.StatusNoContent, "", dh.writer)
}
