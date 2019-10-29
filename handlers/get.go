package handlers

import (
	"net/http"
)

type getHandler struct {
	handlerData
	onlyHeaders bool
}

func (gh getHandler) Handle() *Response {
	resource, _, err := gh.storage.GetResource(gh.requestPath)
	if err != nil {
		return gh.response.SetError(err)
	}

	var response string
	if gh.onlyHeaders {
		response = ""
	} else {
		response, _ = resource.GetContentData()
	}

	etag, _ := resource.GetEtag()
	lastm, _ := resource.GetLastModified(http.TimeFormat)
	ctype, _ := resource.GetContentType()

	gh.response.SetHeader("ETag", etag).
		SetHeader("Last-Modified", lastm).
		SetHeader("Content-Type", ctype).
		Set(http.StatusOK, response)

	return gh.response
}
