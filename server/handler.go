package server

import (
  "net/http"
)

func StartServer() {
  http.HandleFunc("/", RequestHandler)
	http.ListenAndServe(":8000", nil)
}

func RequestHandler(writer http.ResponseWriter, request *http.Request) {
  requestBody := readRequestBody(request)

  logRequest(request, requestBody)

  var method MethodHandler
  switch request.Method {
  case "GET": method = GetHandler{request, requestBody, writer, false}
  case "HEAD": method = GetHandler{request, requestBody, writer, true}
  case "PUT": method = PutHandler{request, requestBody, writer}
  case "DELETE": method = DeleteHandler{request, requestBody, writer}
  case "PROPFIND": method = PropfindHandler{request, requestBody, writer}
  case "OPTIONS": method = OptionsHandler{request, requestBody, writer}
  case "REPORT": method = ReportHandler{request, requestBody, writer}
  default:
    respond(http.StatusNotImplemented, "", writer)
    return
  }

  method.Handle()
}

type MethodHandler interface {
  Handle()
}
