package main

import (
  "bytes"
  "strings"
  "fmt"
  "regexp"
	"io"
  "io/ioutil"
	"net/http"
  "crypto/md5"
  "encoding/hex"
  "github.com/yosssi/gohtml"
  "encoding/xml"
)

type CalendarEvent struct {
  Content string
  Etag    string
}
var eventsStorage map[string]CalendarEvent = make(map[string]CalendarEvent)

func main() {
  http.HandleFunc("/", RequestHandler)
	http.ListenAndServe(":8000", nil)
}

func RequestHandler(writer http.ResponseWriter, request *http.Request) {
  requestBody := readRequestBody(request)

  logRequest(request, requestBody)

  switch request.Method {
  // case "GET": HandleGET(writer, request)
  case "PUT": HandlePUT(writer, request, requestBody)
  case "DELETE": HandleDELETE(writer, request)
  case "PROPFIND": HandlePROPFIND(writer, request, requestBody)
  case "OPTIONS": HandleOPTIONS(writer, request, requestBody)
  case "REPORT": HandleREPORT(writer, request, requestBody)
  }
}

func HandleGET(writer http.ResponseWriter, request *http.Request) {
  // Logs
  // fmt.Printf("\n== GET REQUEST ==\n%s\n\n", request)
  //
  // // Core
  // event_id := ExtractEventID(request.URL.Path)
  // event, found := events_map[event_id]
  //
  // // Responds
  // if found {
  //     fmt.Printf("\n== FOUND EVENT ==\n%s\n\n", event)
  //     io.WriteString(writer, event)
  // } else {
  //     http.NotFound(writer, request)
  // }
}

func HandlePUT(writer http.ResponseWriter, request *http.Request, requestBody string) {
  // Core
  eventID := extractEventID(request.URL.Path)
  event   := CalendarEvent{Content: requestBody, Etag: hash(requestBody)}
  eventsStorage[eventID] = event

  // Responds
  respond(201, "", writer)
}

func HandleDELETE(writer http.ResponseWriter, request *http.Request) {
  // Logs
  // fmt.Printf("\n== DELETE REQUEST ==\n%s", request)
  //
  // // Core
  // event_id := ExtractEventID(request.URL.Path)
  // event, found := events_map[event_id]
  //
  // // Responds
  // if found {
  //     fmt.Printf("\n== FOUND EVENT ==\n%s\n\n", event)
  //     delete(events_map, event_id)
  //     fmt.Println("\n== EVENT DELETED ==\n\n")
  //
  //     io.WriteString(writer, event)
  // } else {
  //     http.NotFound(writer, request)
  // }
}

func HandlePROPFIND(writer http.ResponseWriter, request *http.Request, requestBody string)  {
  // collectionItems := [request.Path]
  //
  // for item in collectionItems {
  //   response.append()
  // }
  // fmt.Printf("\nOKKKKKKKKKKKKKK\n\n")
  // var buffer bytes.Buffer
  //
  // buffer.WriteString("<?xml version=\"1.0\"?>")
  // buffer.WriteString("<multistatus xmlns=\"DAV:\" xmlns:C=\"urn:ietf:params:xml:ns:caldav\" xmlns:CR=\"urn:ietf:params:xml:ns:carddav\">")
  //
  // itemsCollection := []string{request.URL.Path}
  //
  // for _, item := range itemsCollection {
  //   buffer.WriteString("<item>")
  //   buffer.WriteString(item)
  //   buffer.WriteString("</item>")
  // }
  //
  // buffer.WriteString("</multistatus>")
  //
  // io.WriteString(writer, buffer.String())

  expRequestBody := `<?xml version="1.0" encoding="UTF-8"?><D:propfind xmlns:D="DAV:" xmlns:CS="http://calendarserver.org/ns/" xmlns:C="urn:ietf:params:xml:ns:caldav"><D:prop><D:resourcetype/><D:owner/><D:current-user-principal/><D:supported-report-set/><C:supported-calendar-component-set/><CS:getctag/></D:prop></D:propfind>`
  responseBody := `<?xml version="1.0"?><multistatus xmlns="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/"><response><href>/user/calendar</href><propstat><prop><resourcetype><C:calendar /><collection /></resourcetype><owner>/user/</owner><supported-report-set><supported-report><report>principal-property-search</report></supported-report><supported-report><report>sync-collection</report></supported-report><supported-report><report>expand-property</report></supported-report><supported-report><report>principal-search-property-set</report></supported-report></supported-report-set><C:supported-calendar-component-set><C:comp name="VTODO" /><C:comp name="VEVENT" /><C:comp name="VJOURNAL" /></C:supported-calendar-component-set><CS:getctag>"b9cf1a7cd5507061d91993409ba61a81"</CS:getctag></prop><status>HTTP/1.1 200 OK</status></propstat><propstat><prop><current-user-principal /></prop><status>HTTP/1.1 404 Not Found</status></propstat></response></multistatus>`

  if request.URL.Path == "/user/calendar/" && hash(requestBody) == hash(expRequestBody) {
    respond(207, responseBody, writer)
  } else {
    expRequestBody = `<?xml version="1.0" encoding="UTF-8"?><D:propfind xmlns:D="DAV:"><D:prop><D:getcontenttype/><D:resourcetype/><D:getetag/></D:prop></D:propfind>`
    responseBody = `<?xml version="1.0"?> <multistatus xmlns="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav"> <response> <href>/user/calendar</href> <propstat> <prop> <getcontenttype>text/calendar</getcontenttype> <resourcetype> <C:calendar /> <collection /> </resourcetype> <getetag>"b9cf1a7cd5507061d91993409ba61a81"</getetag> </prop> <status>HTTP/1.1 200 OK</status> </propstat> </response> <response> <href>/user/calendar/9b91abda-3b47-434e-9fc7-01cf841de175.ics</href> <propstat> <prop> <getcontenttype>text/calendar; component=vcalendar</getcontenttype> <resourcetype /> <getetag>"5ecc95ff25345aecd462052f7bb3d80a"</getetag> </prop> <status>HTTP/1.1 200 OK</status> </propstat> </response> </multistatus>`

    if request.URL.Path == "/user/calendar/" && hash(requestBody) == hash(expRequestBody) {
      respond(207, responseBody, writer)
    }
  }
}

func HandleOPTIONS(writer http.ResponseWriter, request *http.Request, requestBody string) {
  expRequestBody := ""

  if request.URL.Path == "/user/" && hash(requestBody) == hash(expRequestBody) {
    respond(200, "", writer)
  }
}

// =============== REPORT BEGIN ====================================

func HandleREPORT(writer http.ResponseWriter, request *http.Request, requestBody string) {
  
}



// =============== REPORT END ====================================




// =============== OTHERS ====================================

func readRequestBody(request *http.Request) string {
  body, _ := ioutil.ReadAll(request.Body)
  return string(body)
}

func logRequest(request *http.Request, requestBody string) {
  fmt.Printf("\n** %s REQUEST: %s **", request.Method, request.URL.Path)
  fmt.Printf("\nRequest headers:\n")
  for hkey, hvalue := range request.Header {
    fmt.Printf("%s: %s\n", hkey, hvalue)
  }
  fmt.Printf("\nRequest content:\n%s\n", gohtml.Format(requestBody))
}

// Extracts the event ID from the request's URL path
func extractEventID(requestPath string) string {
  id         := ""
  pattern, _ := regexp.Compile("\\/user\\/calendar\\/(.+)\\.ics")
  matches    := pattern.FindStringSubmatch(requestPath)
  if len(matches) > 1 {
    id = matches[1]
  }

  return id
}

func respond(status int, body string, writer http.ResponseWriter) {
  if body != "" {
    fmt.Printf("\nResponse content:\n%s\n", gohtml.Format(body))
  }
  fmt.Printf("\nAnswer status: %d %s\n\n", status, http.StatusText(status))

  writer.WriteHeader(status)
  io.WriteString(writer, body)
}

func hash(s string) string {
  s = strings.Replace(s, "\n", "", -1)
  s = strings.Replace(s, "\r", "", -1)
  hash := md5.Sum([]byte(s))
  return hex.EncodeToString(hash[:])
}
