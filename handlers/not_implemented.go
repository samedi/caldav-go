package handlers

import (
  "net/http"
)

type notImplementedHandler struct {
  writer http.ResponseWriter
}

func (h notImplementedHandler) Handle() {
  respond(http.StatusNotImplemented, "", h.writer)
}
