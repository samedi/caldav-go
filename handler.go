package caldav

import (
  "net/http"
  "io/ioutil"
  "git.samedi.cc/ferraz/caldav/handlers"
)

func RequestHandler(writer http.ResponseWriter, request *http.Request) {
  response := HandleRequest(writer, request)
  response.Write(writer)
}

func HandleRequest(writer http.ResponseWriter, request *http.Request) *handlers.Response {
  requestBody := readRequestBody(request)
  handler := handlers.NewHandler(request, requestBody)
  return handler.Handle()
}

func readRequestBody(request *http.Request) string {
  body, _ := ioutil.ReadAll(request.Body)
  return string(body)
}
