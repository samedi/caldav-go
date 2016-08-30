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

  "caldav/data"
  "caldav/ixml"
)

// Supported ICal components.
// Currently only VEVENT is supported. VTODO and VJOURNAL are not.
var SupportedComponents = []string{"VEVENT"}

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

  precond := RequestPreconditions{request}

  switch request.Method {
  case "GET": HandleGET(writer, request)
  case "PUT": HandlePUT(writer, request, precond, requestBody)
  case "DELETE": HandleDELETE(writer, request, precond)
  case "PROPFIND": HandlePROPFIND(writer, request, requestBody, nil)
  // case "OPTIONS": HandleOPTIONS(writer, request, requestBody)
  case "REPORT": HandleREPORT(writer, request, requestBody)
  }
}

func HandleGET(writer http.ResponseWriter, request *http.Request) {
  // TODO: Handle GET on collections

  // get the event from the storage
  eventID := extractEventID(request.URL.Path)
  event, found := eventsStorage[eventID]

  if found {
    writer.Header().Set("ETag", event.Etag)
    respond(http.StatusOK, event.Content, writer)
  } else {
    respond(http.StatusNotFound, "", writer)
  }
}

func HandlePUT(writer http.ResponseWriter, request *http.Request, precond RequestPreconditions, requestBody string) {
  // TODO: Handle PUT on collections

  // get the event from the storage
  eventID := extractEventID(request.URL.Path)
  event, found := eventsStorage[eventID]

  // PUT is allowed in 2 cases:
  //
  // 1. Item NOT FOUND and there is NO ETAG match header: CREATE a new item
  if !found && !precond.IfMatchPresent() {
    // create new item
    newEvent := CalendarEvent{Content: requestBody, Etag: hash(requestBody)}
    eventsStorage[eventID] = newEvent

    writer.Header().Set("ETag", newEvent.Etag)
    respond(http.StatusCreated, "", writer)
    return
  }

  // 2. Item exists, the event etag is verified and there's no IF-NONE-MATCH=* header: UPDATE the item
  if found && precond.IfMatch(event.Etag) && !precond.IfNoneMatch("*") {
    // update event
    event.Content = requestBody
    event.Etag = hash(requestBody)
    eventsStorage[eventID] = event

    writer.Header().Set("ETag", event.Etag)
    respond(http.StatusCreated, "", writer)
    return
  }

  respond(http.StatusPreconditionFailed, "", writer)
}

func HandleDELETE(writer http.ResponseWriter, request *http.Request, precond RequestPreconditions) {
  // TODO: Handle delete on collections

  // get the event from the storage
  eventID := extractEventID(request.URL.Path)
  event, found := eventsStorage[eventID]

  if found {
    // check etag pre-condition
    if !precond.IfMatch(event.Etag) {
      respond(http.StatusPreconditionFailed, "", writer)
      return
    }

    // delete event if pre-condition passes
    delete(eventsStorage, eventID)
    respond(http.StatusNoContent, "", writer)
  } else {
    respond(http.StatusNotFound, "", writer)
  }
}

func HandlePROPFIND(writer http.ResponseWriter, request *http.Request, requestBody string, currentUser *data.CalUser)  {
  // Wraps a prop that was processed for a given resource.
  type PropValue struct {
    Tag      xml.Name
    Content  string
    Contents []string
    Status   int
  }

  propToXML := func(pv PropValue) string {
    for _, content := range pv.Contents {
      pv.Content += content
    }
    xmlString := ixml.Tag(pv.Tag, pv.Content)
    return xmlString
  }

  // This is the response of the `propfind` function. It includes all the
  // props processed for a given target resource.
  type Propfind struct {
    // The target resource path. Ex: /user/calendars/c1.ics
    Href  string
    // The set of props (PropValue) processed. Each prop is mapped to a HTTP status code.
    // So if a prop is found and processed ok, it'll be mapped to 200. If it's not found,
    // it'll be mapped to 404, and so on.
    Props map[int][]PropValue
  }

  // Function that processes all the required props for a given resource.
  // ## Params
  // resource: the target calendar resource.
  // reqprops: set of required props that must be processed for the resource.
  // ## Returns
  // A `Propfind` struct.
  propfind := func(resource data.Resource, reqprops []xml.Name) Propfind {
    result := make(map[int][]PropValue)

    for _, ptag := range reqprops {
      pvalue := PropValue{
        Tag: ptag,
        Status: http.StatusOK,
      }

      pfound := false
      switch ptag {
      case xml.Name{Space: "DAV:", Local: "getetag"}:
        pvalue.Content, pfound = resource.GetEtag()
      case xml.Name{Space: "DAV:", Local: "getcontenttype"}:
        pvalue.Content, pfound = resource.GetContentType()
      case xml.Name{Space: "DAV:", Local: "getcontentlength"}:
        pvalue.Content, pfound = resource.GetContentLength()
      case xml.Name{Space: "DAV:", Local: "displayname"}:
        pvalue.Content, pfound = resource.GetDisplayName()
      case xml.Name{Space: "DAV:", Local: "getlastmodified"}:
        pvalue.Content, pfound = resource.GetLastModified(http.TimeFormat)
      case xml.Name{Space: "DAV:", Local: "owner"}:
        pvalue.Content, pfound = resource.GetOwnerPath()
      case xml.Name{Space: "http://calendarserver.org/ns/", Local: "getctag"}:
        pvalue.Content, pfound = resource.GetEtag()
      case xml.Name{Space: "DAV:", Local: "principal-URL"},
           xml.Name{Space: "DAV:", Local: "principal-collection-set"},
           xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "calendar-user-address-set"},
           xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "calendar-home-set"}:
        pvalue.Content, pfound = fmt.Sprintf("<D:href>%s</D:href>", resource.Path), true
      case xml.Name{Space: "DAV:", Local: "resourcetype"}:
        if resource.IsCollection() {
          pvalue.Content, pfound = "<D:collection/><C:calendar/>", true

          if resource.IsPrincipal() {
            pvalue.Content += "<D:principal/>"
          }
        } else {
          // resourcetype must be returned empty for non-collection elements
          pvalue.Content, pfound = "", true
        }
      case xml.Name{Space: "DAV:", Local: "current-user-principal"}:
        if resource.User != nil {
          pvalue.Content, pfound = fmt.Sprintf("<D:href>/%s/</D:href>", resource.User.Name), true
        }
      case xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "supported-calendar-component-set"}:
        if resource.IsCollection() {
          for _, component := range SupportedComponents {
            compTag := fmt.Sprintf(`<C:comp name="%s"/>`, component)
            pvalue.Contents = append(pvalue.Contents, compTag)
          }
          pfound = true
        }
      }

      if !pfound {
        pvalue.Status = http.StatusNotFound
      }

      result[pvalue.Status] = append(result[pvalue.Status], pvalue)
    }

    return Propfind {
      Href: resource.Path,
      Props:  result,
    }
  }

  // get the target resources based on the request URL
  storage := new(data.FileStorage)
  resources, err := storage.GetResources(request.URL.Path, getDepth(request), currentUser)
  if err != nil {
    if err == data.ErrResourceNotFound {
      respond(http.StatusNotFound, "", writer)
      return
    }
    respond(http.StatusMethodNotAllowed, "", writer)
    return
  }

  // read body string to xml struct
  type XMLProp2 struct {
    Tags []xml.Name `xml:",any"`
  }
  type XMLRoot2 struct {
    XMLName xml.Name
    Prop    XMLProp2  `xml:"DAV: prop"`
  }
  var requestXML XMLRoot2
  xml.Unmarshal([]byte(requestBody), &requestXML)

  // init response
  var response bytes.Buffer
  response.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
  response.WriteString(`<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">`)

  // for each resource, fetch the requested props and build the response
  for _, resource := range resources {
    pf := propfind(resource, requestXML.Prop.Tags)

    response.WriteString("<D:response>")
    response.WriteString(fmt.Sprintf("<D:href>%s</D:href>", pf.Href))

    for status, props := range pf.Props {
      response.WriteString("<D:propstat>")
      response.WriteString("<D:prop>")
      for _, prop := range props {
        response.WriteString(propToXML(prop))
      }
      response.WriteString("</D:prop>")
      response.WriteString(ixml.StatusTag(status))
      response.WriteString("</D:propstat>")
    }

    response.WriteString("</D:response>")
  }
  response.WriteString("</D:multistatus>")

  respond(207, response.String(), writer) // Multi-Status
}

func HandleOPTIONS(writer http.ResponseWriter, request *http.Request, requestBody string) {
  // expRequestBody := ""
  //
  // if request.URL.Path == "/user/" && hash(requestBody) == hash(expRequestBody) {
  //   respond(200, "", writer)
  // }
}

func HandleREPORT(writer http.ResponseWriter, request *http.Request, requestBody string) {
  // TODO: HANDLE FILTERS, DEPTH

  type XMLProp struct {
    Tags []xml.Name `xml:",any"`
  }

  type XMLRoot struct {
    XMLName xml.Name
    Prop    XMLProp  `xml:"DAV: prop"`
    Hrefs   []string `xml:"DAV: href"`
  }

  // read body string to xml struct
  var requestXML XMLRoot
  xml.Unmarshal([]byte(requestBody), &requestXML)

  // declare props and other stuff that will be checked/used later
  etagProp := xml.Name{Space:"DAV:", Local:"getetag"}
  dataProp := xml.Name{Space:"urn:ietf:params:xml:ns:caldav", Local:"calendar-data"}
  emptyEvent := CalendarEvent{}

  // init response
  var response bytes.Buffer
  response.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
  response.WriteString(`<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">`)

  // The hrefs can come from the request URL (in this case will be only one) or from the request body itself.
  // The one in the URL will have priority (see rfc4791#section-7.9).
  var reportHrefs []string
  if extractEventID(request.URL.Path) != "" {
    reportHrefs = []string{request.URL.Path}
  } else {
    reportHrefs = requestXML.Hrefs
  }

  // iterate over event hrefs and build response xml on the fly
  for _, href := range reportHrefs {
    response.WriteString("<D:response>")
    response.WriteString(fmt.Sprintf("<D:href>%s</D:href>", href))

    eventID := extractEventID(href)
    event   := eventsStorage[eventID]

    if event == emptyEvent {
      // if does not find the event set 404
      response.WriteString(ixml.StatusTag(http.StatusNotFound))
    } else {
      // if it finds the event, proceed on checking each prop against it
      foundProps     := []string{}
      notFoundProps  := []string{}

      for _, prop := range requestXML.Prop.Tags {
        if prop == etagProp {
          foundProps = append(foundProps, ixml.Tag(etagProp, event.Etag))
        } else if prop == dataProp {
          foundProps = append(foundProps, ixml.Tag(dataProp, event.Content))
        } else {
          notFoundProps = append(notFoundProps, ixml.Tag(prop, ""))
        }
      }

      if len(foundProps) > 0 {
        response.WriteString("<D:propstat>")
        response.WriteString("<D:prop>")
        for _, propTag := range foundProps {
          response.WriteString(propTag)
        }
        response.WriteString("</D:prop>")
        response.WriteString(ixml.StatusTag(http.StatusOK))
        response.WriteString("</D:propstat>")
      }

      if len(notFoundProps) > 0 {
        response.WriteString("<D:propstat>")
        response.WriteString("<D:prop>")
        for _, propTag := range notFoundProps {
          response.WriteString(propTag)
        }
        response.WriteString("</D:prop>")
        response.WriteString(ixml.StatusTag(http.StatusNotFound))
        response.WriteString("</D:propstat>")
      }
    }
    response.WriteString("</D:response>")
  }
  response.WriteString("</D:multistatus>")

  respond(207, response.String(), writer) // Multi-Status
}

// =============== OTHERS ====================================

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

type RequestPreconditions struct {
  request *http.Request
}

func (p *RequestPreconditions) IfMatch(etag string) bool {
  etagMatch := p.request.Header["If-Match"]
  return len(etagMatch) == 0 || etagMatch[0] == "*" || etagMatch[0] == etag
}

func (p *RequestPreconditions) IfMatchPresent() bool {
  return len(p.request.Header["If-Match"]) != 0
}

func (p *RequestPreconditions) IfNoneMatch(value string) bool {
  valueMatch := p.request.Header["If-None-Match"]
  return len(valueMatch) == 1 && valueMatch[0] == value
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
