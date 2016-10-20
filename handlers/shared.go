package handlers

import (
  "net/http"
  "git.samedi.cc/ferraz/caldav/lib"
)

// Supported ICal components on this server.
var supportedComponents = []string{lib.VCALENDAR, lib.VEVENT}

// parseResourceDepth parses the Depth value from the request header and returns a boolean flag,
// where `true` means to include the children on subsequent searches, and `false` to not include.
// This is used on request methods (e.g. PROPFIND) that are requesting a specific resource and may or
// may not want to include the resource's children in the response.
func parseResourceDepth(request *http.Request) bool {
  var depth string
  if depthHeader := request.Header["Depth"]; len(depthHeader) != 0 {
    depth = depthHeader[0]
  }

  if depth == "1" {
    return true
  }

	return false
}
