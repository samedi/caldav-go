package handlers

import (
	"bytes"
	"github.com/samedi/caldav-go/lib"
	"io/ioutil"
	"net/http"
)

// Supported ICal components on this server.
var supportedComponents = []string{lib.VCALENDAR, lib.VEVENT}

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
