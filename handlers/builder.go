package handlers

import (
  "net/http"
)

type handlerInterface interface {
  Handle()
}

func NewHandler(request *http.Request, requestBody string, writer http.ResponseWriter) handlerInterface {
  switch request.Method {
  case "GET": return getHandler{request, writer, false}
  case "HEAD": return getHandler{request, writer, true}
  case "PUT": return putHandler{request, requestBody, writer}
  case "DELETE": return deleteHandler{request, writer}
  case "PROPFIND": return propfindHandler{request, requestBody, writer}
  case "OPTIONS": return optionsHandler{writer}
  case "REPORT": return reportHandler{request, requestBody, writer}
  default: return notImplementedHandler{writer}
  }
}
