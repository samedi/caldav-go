package caldav

import (
  "bytes"
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

// This function reads the request body and restore its content, so that
// the request body can be read a second time.
func readRequestBody(request *http.Request) string {
  // Read the content
  body, _ := ioutil.ReadAll(request.Body)
  // Restore the io.ReadCloser to its original state
  request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
  // Use the content
  return string(body)
}
