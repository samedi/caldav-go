package handlers

import (
  "net/http"
  "git.samedi.cc/ferraz/caldav/data"
  "git.samedi.cc/ferraz/caldav/global"
)

type putHandler struct {
  request *http.Request
  requestBody string
  writer http.ResponseWriter
}

func (ph putHandler) Handle() {
  precond := requestPreconditions{ph.request}
  success := false

  // check if resource exists
  resourcePath := ph.request.URL.Path
  resource, found, err := global.Storage.GetResource(resourcePath)
  if err != nil && err != data.ErrResourceNotFound {
    respondWithError(err, ph.writer)
    return
  }

  // PUT is allowed in 2 cases:
  //
  // 1. Item NOT FOUND and there is NO ETAG match header: CREATE a new item
  if !found && !precond.IfMatchPresent() {
    // create new event resource
    resource, err = global.Storage.CreateResource(resourcePath, ph.requestBody)
    if err != nil {
      respondWithError(err, ph.writer)
      return
    }

    success = true
  }

  if found {
    // TODO: Handle PUT on collections
    if resource.IsCollection() {
      respond(http.StatusPreconditionFailed, "", ph.writer)
      return
    }

    // 2. Item exists, the resource etag is verified and there's no IF-NONE-MATCH=* header: UPDATE the item
    resourceEtag, _ := resource.GetEtag()
    if found && precond.IfMatch(resourceEtag) && !precond.IfNoneMatch("*") {
      // update resource
      resource, err = global.Storage.UpdateResource(resourcePath, ph.requestBody)
      if err != nil {
        respondWithError(err, ph.writer)
        return
      }

      success = true
    }
  }

  if success {
    resourceEtag, _ := resource.GetEtag()
    ph.writer.Header().Set("ETag", resourceEtag)
    respond(http.StatusCreated, "", ph.writer)
    return
  }

  respond(http.StatusPreconditionFailed, "", ph.writer)
}
