package main

import (
  "fmt"
  "regexp"
	"io"
  "io/ioutil"
	"net/http"
)

var events_map map[string]string = make(map[string]string)

func main() {
  http.HandleFunc("/", RequestHandler)
	http.ListenAndServe(":8000", nil)
}

func RequestHandler(writer http.ResponseWriter, request *http.Request) {
  // fmt.Printf("\n== REQUEST ==\n%s\n\n", r)
  //
  // events[uuid] =
  //
  // sample_response = `
  // BEGIN:VCALENDAR
  // VERSION:2.0
  // PRODID:-//hacksw/handcal//NONSGML v1.0//EN
  // BEGIN:VEVENT
  // UID:uid1@example.com
  // DTSTAMP:19970714T170000Z
  // ORGANIZER;CN=John Doe:MAILTO:john.doe@example.com
  // DTSTART:19970714T170000Z
  // DTEND:19970715T035959Z
  // SUMMARY:Bastille Day Party
  // END:VEVENT
  // END:VCALENDAR
  // `
  //
  //
  //
  // io.WriteString(w, sample_response)

  switch request.Method {
  case "GET": HandleGET(writer, request)
  case "PUT": HandlePUT(writer, request)
  }
}

func HandleGET(writer http.ResponseWriter, request *http.Request) {
  // Logs
  fmt.Printf("\n== GET REQUEST ==\n%s\n\n", request)

  // Core
  event_id := ExtractEventID(request.URL.Path)
  event, found := events_map[event_id]

  // Responds
  if found {
      fmt.Printf("\n== FOUND EVENT ==\n%s\n\n", event)
      io.WriteString(writer, event)
  } else {
      http.NotFound(writer, request)
  }
}

func HandlePUT(writer http.ResponseWriter, request *http.Request) {
  // Logs
  fmt.Printf("\n== PUT REQUEST ==\n%s", request)
  body, _ := ioutil.ReadAll(request.Body)
  fmt.Printf("\n== BODY ==\n%s\n\n", body)

  // Core
  event_id := ExtractEventID(request.URL.Path)
  event := string(body)
  events_map[event_id] = event
  fmt.Printf("\n== UPDATED EVENT ==\n%s\n\n", event)

  // Responds
  io.WriteString(writer, event)
}

// Extracts the event ID from the request's URL path
func ExtractEventID(request_path string) string {
  pattern, _ := regexp.Compile("\\/user\\/calendar\\/(.+)\\.ics")
  matches    := pattern.FindStringSubmatch(request_path)
  id         := matches[1]

  return id
}
