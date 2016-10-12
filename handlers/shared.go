package handlers

import (
  "net/http"
  "git.samedi.cc/ferraz/caldav/lib"
  "git.samedi.cc/ferraz/caldav/data"
)

// Supported ICal components on this server.
var supportedComponents = []string{lib.VCALENDAR, lib.VEVENT}

// TODO: implement after integrate authentication
func getCurrentUser() *data.CalUser {
  return nil
}

const (
	infiniteDepth = -1
	invalidDepth  = -2
)

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
