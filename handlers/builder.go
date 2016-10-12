package handlers

import (
  "net/http"
)

type handlerInterface interface {
  Handle() *Response
}

func NewHandler(request *http.Request, requestBody string) handlerInterface {
  response := NewResponse()

  switch request.Method {
  case "GET": return getHandler{request, response, false}
  case "HEAD": return getHandler{request, response, true}
  case "PUT": return putHandler{request, requestBody, response}
  case "DELETE": return deleteHandler{request, response}
  case "PROPFIND": return propfindHandler{request, requestBody, response}
  case "OPTIONS": return optionsHandler{response}
  case "REPORT": return reportHandler{request, requestBody, response}
  default: return notImplementedHandler{response}
  }
}
