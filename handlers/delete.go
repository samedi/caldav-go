package handlers

import (
	"net/http"
)

type deleteHandler struct {
	handlerData
}

func (dh deleteHandler) Handle() *Response {
	precond := requestPreconditions{dh.request}

	// get the event from the storage
	resource, _, err := dh.storage.GetShallowResource(dh.requestPath)
	if err != nil {
		return dh.response.SetError(err)
	}

	// TODO: Handle delete on collections
	if resource.IsCollection() {
		return dh.response.Set(http.StatusMethodNotAllowed, "")
	}

	// check ETag pre-condition
	resourceEtag, _ := resource.GetEtag()
	if !precond.IfMatch(resourceEtag) {
		return dh.response.Set(http.StatusPreconditionFailed, "")
	}

	// delete event after pre-condition passed
	err = dh.storage.DeleteResource(resource.Path)
	if err != nil {
		return dh.response.SetError(err)
	}

	return dh.response.Set(http.StatusNoContent, "")
}
