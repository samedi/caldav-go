package caldav

import (
  "net/http"
  "io/ioutil"
  "git.samedi.cc/ferraz/caldav/handlers"
)

func RequestHandler(writer http.ResponseWriter, request *http.Request) {
  requestBody := readRequestBody(request)
  handler := handlers.NewHandler(request, requestBody, writer)
  handler.Handle()
}

func readRequestBody(request *http.Request) string {
  body, _ := ioutil.ReadAll(request.Body)
  return string(body)
}
