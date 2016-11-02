package caldav

import (
  "net/http"
  "git.samedi.cc/ferraz/caldav/handlers"
)

func RequestHandler(writer http.ResponseWriter, request *http.Request) {
  response := HandleRequest(writer, request)
  response.Write(writer)
}

func HandleRequest(writer http.ResponseWriter, request *http.Request) *handlers.Response {
  handler := handlers.NewHandler(request)
  return handler.Handle()
}
