package handlers

import (
	"net/http"
)

type notImplementedHandler struct {
	handlerData
}

func (h notImplementedHandler) Handle() *Response {
	return h.response.Set(http.StatusNotImplemented, "")
}
