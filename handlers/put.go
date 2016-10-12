package handlers

import (
  "net/http"
  "git.samedi.cc/ferraz/caldav/errs"
  "git.samedi.cc/ferraz/caldav/global"
)

type putHandler struct {
  request *http.Request
  requestBody string
  response *Response
}

func (ph putHandler) Handle() *Response {
  precond := requestPreconditions{ph.request}
  success := false

  // check if resource exists
  resourcePath := ph.request.URL.Path
  resource, found, err := global.Storage.GetResource(resourcePath)
  if err != nil && err != errs.ResourceNotFoundError {
    return ph.response.SetError(err)
  }

  // PUT is allowed in 2 cases:
  //
  // 1. Item NOT FOUND and there is NO ETAG match header: CREATE a new item
  if !found && !precond.IfMatchPresent() {
    // create new event resource
    resource, err = global.Storage.CreateResource(resourcePath, ph.requestBody)
    if err != nil {
      return ph.response.SetError(err)
    }

    success = true
  }

  if found {
    // TODO: Handle PUT on collections
    if resource.IsCollection() {
      return ph.response.Set(http.StatusPreconditionFailed, "")
    }

    // 2. Item exists, the resource etag is verified and there's no IF-NONE-MATCH=* header: UPDATE the item
    resourceEtag, _ := resource.GetEtag()
    if found && precond.IfMatch(resourceEtag) && !precond.IfNoneMatch("*") {
      // update resource
      resource, err = global.Storage.UpdateResource(resourcePath, ph.requestBody)
      if err != nil {
        return ph.response.SetError(err)
      }

      success = true
    }
  }

  if success {
    resourceEtag, _ := resource.GetEtag()
    ph.response.SetHeader("ETag", resourceEtag)
    return ph.response.Set(http.StatusCreated, "")
  }

  return ph.response.Set(http.StatusPreconditionFailed, "")
}
