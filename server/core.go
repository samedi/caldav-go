package server

import (
  "fmt"
	"io"
  "io/ioutil"
  "net/http"
  "encoding/xml"

  "github.com/yosssi/gohtml"

  "caldav/data"
)

// Supported ICal components.
// Currently only VEVENT is supported. VTODO and VJOURNAL are not.
var SupportedComponents = []string{"VEVENT"}

func StartServer() {
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

  multistatus := NewMultistatusResp()
  // for each href, build the multistatus responses
  for _, resource := range resources {
    propstats := multistatus.Propstats(&resource, requestXML.Prop.Tags)
    multistatus.AddResponse(resource.Path, true, propstats)
  }

  respond(207, multistatus.ToXML(), writer)
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
