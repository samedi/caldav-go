package server

import (
  "log"
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
  var msg lib.StringBuffer
  msg.Write("\n** %s REQUEST: %s **\n", request.Method, request.URL.Path)
  msg.Write("\nRequest headers:\n")
  for hkey, hvalue := range request.Header {
    msg.Write("%s: %s\n", hkey, hvalue)
  }
  if body != "" {
    msg.Write("\nRequest content:\n%s\n\n", gohtml.Format(body))
  }

  log.Printf(msg.String())
}

func respond(status int, body string, writer http.ResponseWriter) {
  var msg lib.StringBuffer
  if body != "" {
    msg.Write("\nResponse content:\n%s\n", gohtml.Format(body))
  }

  msg.Write("\nResponse headers:\n")
  for hkey, hvalue := range writer.Header() {
    msg.Write("%s: %s\n", hkey, hvalue)
  }

  writer.WriteHeader(status)
  msg.Write("\nAnswer status: %d %s\n\n", status, http.StatusText(status))

  log.Printf(msg.String())
  io.WriteString(writer, body)
}

func respondWithError(err error, writer http.ResponseWriter) {
  log.Printf("\n*** Error: %s ***\n", err)
  respond(http.StatusInternalServerError, "", writer)
}
