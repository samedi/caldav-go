package server

import (
  "fmt"
	"io"
  "io/ioutil"
  "net/http"
  "github.com/yosssi/gohtml"

  "caldav/lib"
  "caldav/data"
)

// Supported ICal components on this server.
var SupportedComponents = []string{lib.VCALENDAR, lib.VEVENT}

const (
	infiniteDepth = -1
	invalidDepth  = -2
)

var storage data.FileStorage = data.FileStorage{}

func getDepth(request *http.Request) int {
  d := "infinity"

  if hd := request.Header["Depth"]; len(hd) != 0 {
    d = hd[0]
  }

	switch d {
	case "0":
		return 0
	case "1":
		return 1
	case "infinity":
		return infiniteDepth
	}
	return invalidDepth
}

// TODO: implement after integrate authentication
func getCurrentUser() *data.CalUser {
  return nil
}

func readRequestBody(request *http.Request) string {
  body, _ := ioutil.ReadAll(request.Body)
  return string(body)
}

func logRequest(request *http.Request, body string) {
  fmt.Printf("\n** %s REQUEST: %s **", request.Method, request.URL.Path)
  fmt.Printf("\nRequest headers:\n")
  for hkey, hvalue := range request.Header {
    fmt.Printf("%s: %s\n", hkey, hvalue)
  }
  if body != "" {
    fmt.Printf("\nRequest content:\n%s\n", gohtml.Format(body))
  }
}

func respond(status int, body string, writer http.ResponseWriter) {
  if body != "" {
    fmt.Printf("\nResponse content:\n%s\n", gohtml.Format(body))
  }

  fmt.Printf("\nResponse headers:\n")
  for hkey, hvalue := range writer.Header() {
    fmt.Printf("%s: %s\n", hkey, hvalue)
  }

  writer.WriteHeader(status)
  fmt.Printf("\nAnswer status: %d %s\n\n", status, http.StatusText(status))

  io.WriteString(writer, body)
}

func respondWithError(err error, writer http.ResponseWriter) {
  // TODO: Better logging
  fmt.Printf("\n*** Error: %s ***\n", err)
  respond(http.StatusInternalServerError, "", writer)
}
