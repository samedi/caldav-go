package main

import (
  "bytes"
  "fmt"
  "regexp"
	"io"
  "io/ioutil"
	"net/http"
  "github.com/yosssi/gohtml"
  "encoding/xml"

  "caldav/data"
  "caldav/ixml"
)

// Supported ICal components.
// Currently only VEVENT is supported. VTODO and VJOURNAL are not.
var SupportedComponents = []string{"VEVENT"}

func main() {
  http.HandleFunc("/", RequestHandler)
	http.ListenAndServe(":8000", nil)
}

func RequestHandler(writer http.ResponseWriter, request *http.Request) {
  requestBody := readRequestBody(request)

  logRequest(request, requestBody)

  precond := RequestPreconditions{request}

  switch request.Method {
  case "GET": HandleGET(writer, request, false)
  case "HEAD": HandleHEAD(writer, request)
  case "PUT": HandlePUT(writer, request, precond, requestBody)
  case "DELETE": HandleDELETE(writer, request, precond)
  case "PROPFIND": HandlePROPFIND(writer, request, requestBody, nil)
  case "OPTIONS": HandleOPTIONS(writer, request)
  case "REPORT": HandleREPORT(writer, request, requestBody)
  }
}

func HandleGET(writer http.ResponseWriter, request *http.Request, onlyheaders bool) {
  storage := new(data.FileStorage)

  resource, found, err := storage.GetResource(request.URL.Path)
  if err != nil && err != data.ErrResourceNotFound {
    respondWithError(err, writer)
    return
  }

  if !found {
    respond(http.StatusNotFound, "", writer)
    return
  }

  etag, _ := resource.GetEtag()
  writer.Header().Set("ETag", etag)
  lastm, _ := resource.GetLastModified(http.TimeFormat)
  writer.Header().Set("Last-Modified", lastm)
  ctype, _ := resource.GetContentType()
  writer.Header().Set("Content-Type", ctype)

  var response string
  if onlyheaders {
    response = ""
  } else {
    response, _ = resource.GetData()
  }

  respond(http.StatusOK, response, writer)
}

func HandleHEAD(writer http.ResponseWriter, request *http.Request) {
  HandleGET(writer, request, true)
}

func HandlePUT(writer http.ResponseWriter, request *http.Request, precond RequestPreconditions, requestBody string) {
  storage := new(data.FileStorage)
  success := false

  // check if resource exists
  resourcePath := request.URL.Path
  resource, found, err := storage.GetResource(resourcePath)
  if err != nil && err != data.ErrResourceNotFound {
    respondWithError(err, writer)
    return
  }

  // PUT is allowed in 2 cases:
  //
  // 1. Item NOT FOUND and there is NO ETAG match header: CREATE a new item
  if !found && !precond.IfMatchPresent() {
    // create new event resource
    resource, err = storage.CreateResource(resourcePath, requestBody)
    if err != nil {
      respondWithError(err, writer)
      return
    }

    success = true
  }

  if found {
    // TODO: Handle PUT on collections
    if resource.IsCollection() {
      respond(http.StatusPreconditionFailed, "", writer)
      return
    }

    // 2. Item exists, the resource etag is verified and there's no IF-NONE-MATCH=* header: UPDATE the item
    resourceEtag, _ := resource.GetEtag()
    if found && precond.IfMatch(resourceEtag) && !precond.IfNoneMatch("*") {
      // update resource
      resource, err = storage.UpdateResource(resourcePath, requestBody)
      if err != nil {
        respondWithError(err, writer)
        return
      }

      success = true
    }
  }

  if success {
    resourceEtag, _ := resource.GetEtag()
    writer.Header().Set("ETag", resourceEtag)
    respond(http.StatusCreated, "", writer)
    return
  }

  respond(http.StatusPreconditionFailed, "", writer)
}

func HandleDELETE(writer http.ResponseWriter, request *http.Request, precond RequestPreconditions) {
  storage := new(data.FileStorage)

  // get the event from the storage
  resource, found, err := storage.GetResource(request.URL.Path)
  if err != nil && err != data.ErrResourceNotFound {
    respondWithError(err, writer)
    return
  }

  if !found {
    respond(http.StatusNotFound, "", writer)
    return
  }

  // TODO: Handle delete on collections
  if resource.IsCollection() {
    respond(http.StatusMethodNotAllowed, "", writer)
    return
  }

  // check ETag pre-condition
  resourceEtag, _ := resource.GetEtag()
  if !precond.IfMatch(resourceEtag) {
    respond(http.StatusPreconditionFailed, "", writer)
    return
  }

  // delete event after pre-condition passed
  err = storage.DeleteResource(resource.Path)
  if err != nil {
    respondWithError(err, writer)
    return
  }

  respond(http.StatusNoContent, "", writer)
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

  // Function that processes all the required props for a given resource.
  // ## Params
  // resource: the target calendar resource.
  // reqprops: set of required props that must be processed for the resource.
  // ## Returns
  // The set of props (PropValue) processed. Each prop is mapped to a HTTP status code.
  // So if a prop is found and processed ok, it'll be mapped to 200. If it's not found,
  // it'll be mapped to 404, and so on.
  propfind := func(resource *data.Resource, reqprops []xml.Name) map[int][]PropValue {
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

    return result
  }

  // get the target resources based on the request URL
  storage := new(data.FileStorage)
  resources, err := storage.GetResources(request.URL.Path, getDepth(request), currentUser)
  if err != nil {
    if err == data.ErrResourceNotFound {
      respond(http.StatusNotFound, "", writer)
      return
    }
    respondWithError(err, writer)
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
  response.WriteString(fmt.Sprintf(`<D:multistatus %s>`, ixml.Namespaces()))

  // for each resource, fetch the requested props and build the response
  for _, resource := range resources {
    response.WriteString("<D:response>")
    response.WriteString(fmt.Sprintf("<D:href>%s</D:href>", resource.Path))

    propsMap := propfind(&resource, requestXML.Prop.Tags)

    for status, props := range propsMap {
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

// Returns the allowed methods and the DAV features implemented by the current server.
// For more information about the values and format read RFC4918 Sections 10.1 and 18.
func HandleOPTIONS(writer http.ResponseWriter, request *http.Request) {
  writer.Header().Set("Allow", "GET, HEAD, PUT, DELETE, OPTIONS, PROPFIND, REPORT")
  // Set the DAV compliance header:
  // 1: Server supports all the requirements specified in RFC2518
  // 3: Server supports all the revisions specified in RFC4918
  // calendar-access: Server supports all the extensions specified in RFC4791
  writer.Header().Set("DAV", "1, 3, calendar-access")

  respond(http.StatusOK, "", writer)
}

func HandleREPORT(writer http.ResponseWriter, request *http.Request, requestBody string) {
  // TODO: HANDLE FILTERS, DEPTH, COLLECTIONS

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

  report := func(resource *data.Resource, reqprops []xml.Name) map[int][]PropValue {
    result := make(map[int][]PropValue)

    for _, ptag := range reqprops {
      pvalue := PropValue{
        Tag: ptag,
        Status: http.StatusOK,
      }

      pfound := false

      switch ptag {
      case xml.Name{Space:"DAV:", Local:"getetag"}:
        pvalue.Content, pfound = resource.GetEtag()
      case xml.Name{Space: "DAV:", Local: "getcontenttype"}:
        pvalue.Content, pfound = resource.GetContentType()
      case xml.Name{Space:"urn:ietf:params:xml:ns:caldav", Local:"calendar-data"}:
        pvalue.Content, pfound = resource.GetData()
      }

      if !pfound {
        pvalue.Status = http.StatusNotFound
      }

      result[pvalue.Status] = append(result[pvalue.Status], pvalue)
    }

    return result
  }

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

  // The hrefs can come from the request URL (in this case will be only one) or from the request body itself.
  // The one in the URL will have priority (see RFC4791#section-7.9).
  var reportHrefs []string
  if extractEventID(request.URL.Path) != "" {
    reportHrefs = []string{request.URL.Path}
  } else {
    reportHrefs = requestXML.Hrefs
  }

  storage := new(data.FileStorage)

  // init response
  var response bytes.Buffer
  response.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
  response.WriteString(fmt.Sprintf(`<D:multistatus %s>`, ixml.Namespaces()))

  // iterate over event hrefs and build response xml on the fly
  for _, href := range reportHrefs {
    resource, found, err := storage.GetResource(href)
    if err != nil && err != data.ErrResourceNotFound {
      respondWithError(err, writer)
      return
    }

    response.WriteString("<D:response>")
    response.WriteString(fmt.Sprintf("<D:href>%s</D:href>", href))

    if found {
      reportMap := report(resource, requestXML.Prop.Tags)

      for status, props := range reportMap {
        response.WriteString("<D:propstat>")
        response.WriteString("<D:prop>")
        for _, prop := range props {
          response.WriteString(propToXML(prop))
        }
        response.WriteString("</D:prop>")
        response.WriteString(ixml.StatusTag(status))
        response.WriteString("</D:propstat>")
      }
    } else {
      // if does not find the resource set 404
      response.WriteString(ixml.StatusTag(http.StatusNotFound))
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
