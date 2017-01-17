package caldav

import (
  "fmt"
  "os"
  "time"
  "testing"
  "strings"
  "net/http"
  "io/ioutil"

  "git.samedi.cc/ferraz/caldav/test"
)

// ============= TESTS ======================

func TestMain(m *testing.M) {
  go startServer()

  // wait for the server to be started
  time.Sleep(time.Second/3)
  os.Exit(m.Run())
}

const (
  TEST_SERVER_PORT = "8001"
)

func startServer() {
  http.HandleFunc("/", RequestHandler)
  http.ListenAndServe(":" + TEST_SERVER_PORT, nil)
}

func TestOPTIONS(t *testing.T) {
  resp := doRequest("OPTIONS", "/test-data/", "", nil)

  if test.AssertInt(len(resp.Header["Allow"]), 1, t) {
    test.AssertStr(resp.Header["Allow"][0], "GET, HEAD, PUT, DELETE, OPTIONS, PROPFIND, REPORT", t)
  }

  if test.AssertInt(len(resp.Header["Dav"]), 1, t) {
    test.AssertStr(resp.Header["Dav"][0], "1, 3, calendar-access", t)
  }

  test.AssertInt(resp.StatusCode, http.StatusOK, t)
}

func TestGET(t *testing.T) {
  collection := "/test-data/get/"
  rName := "123-456-789.ics"
  rPath := collection + rName
  rData := "BEGIN:VEVENT; SUMMARY:Party; END:VEVENT"
  createResource(collection, rName, rData)

  resp := doRequest("GET", rPath, "", nil)
  body := readResponseBody(resp)

  test.AssertInt(len(resp.Header["Etag"]), 1, t)
  test.AssertInt(len(resp.Header["Last-Modified"]), 1, t)
  test.AssertInt(len(resp.Header["Content-Type"]), 1, t)
  test.AssertStr(resp.Header["Content-Type"][0], "text/calendar; component=vcalendar", t)
  test.AssertStr(body, rData, t)
  test.AssertInt(resp.StatusCode, http.StatusOK, t)
}

func TestPUT(t *testing.T) {
  rpath := "/test-data/put/123-456-789.ics"

  // test when trying to create a new resource and a IF-MATCH header is present
  headers := map[string]string{
    "If-Match": "1111111111111",
  }
  resp := doRequest("PUT", rpath, "", headers)
  test.AssertInt(resp.StatusCode, http.StatusPreconditionFailed, t)
  test.AssertResourceDoesNotExist(rpath, t)

  // test when trying to create a new resource (no headers this time)
  resourceData := "BEGIN:VEVENT; SUMMARY:Lunch; END:VEVENT"
  resp = doRequest("PUT", rpath, resourceData, nil)
  test.AssertInt(resp.StatusCode, http.StatusCreated, t)
  if !test.AssertInt(len(resp.Header["Etag"]), 1, t) {
    return
  }
  etag := resp.Header["Etag"][0]
  test.AssertResourceExists(rpath, t)
  test.AssertResourceData(rpath, resourceData, t)

  // test when trying to update a collection (folder)
  resp = doRequest("PUT", "/test-data/put/", "", nil)
  test.AssertInt(resp.StatusCode, http.StatusPreconditionFailed, t)

  // test when trying to update the resource but the ETag check (IF-MATCH header) does not match
  originalData := resourceData
  updatedData := "BEGIN:VEVENT; SUMMARY:Meeting; END:VEVENT"
  resp = doRequest("PUT", rpath, updatedData, headers)
  test.AssertInt(resp.StatusCode, http.StatusPreconditionFailed, t)
  test.AssertResourceData(rpath, originalData, t)

  // test when trying to update the resource with the correct ETag check
  headers["If-Match"] = etag
  resp = doRequest("PUT", rpath, updatedData, headers)
  test.AssertInt(resp.StatusCode, http.StatusCreated, t)
  test.AssertResourceData(rpath, updatedData, t)

  // test when trying to force update the resource by not passing any ETag check
  originalData = updatedData
  updatedData = "BEGIN:VEVENT; SUMMARY:Gym; END:VEVENT"
  delete(headers, "If-Match")
  resp = doRequest("PUT", rpath, updatedData, headers)
  test.AssertInt(resp.StatusCode, http.StatusCreated, t)
  test.AssertResourceData(rpath, updatedData, t)

  // test when trying to update the resource but there is a IF-NONE-MATCH=*
  originalData = updatedData
  updatedData = "BEGIN:VEVENT; SUMMARY:Party; END:VEVENT"
  headers["If-None-Match"] = "*"
  resp = doRequest("PUT", rpath, updatedData, headers)
  test.AssertInt(resp.StatusCode, http.StatusPreconditionFailed, t)
  test.AssertResourceData(rpath, originalData, t)
}

func TestDELETE(t *testing.T) {
  collection := "/test-data/delete/"
  rName := "123-456-789.ics"
  rpath := collection + rName
  createResource(collection, rName, "BEGIN:VEVENT; SUMMARY:Party; END:VEVENT")

  // test deleting a resource that does not exist
  resp := doRequest("DELETE", "/foo/bar", "", nil)
  test.AssertInt(resp.StatusCode, http.StatusNotFound, t)

  // test deleting a collection (folder)
  resp = doRequest("DELETE", collection, "", nil)
  test.AssertInt(resp.StatusCode, http.StatusMethodNotAllowed, t)
  test.AssertResourceExists(rpath, t)

  // test trying deleting when ETag check fails
  headers := map[string]string{
    "If-Match": "1111111111111",
  }
  resp = doRequest("DELETE", rpath, "", headers)
  test.AssertInt(resp.StatusCode, http.StatusPreconditionFailed, t)
  test.AssertResourceExists(rpath, t)

  // test finally deleting the resource
  resp = doRequest("DELETE", rpath, "", nil)
  test.AssertInt(resp.StatusCode, http.StatusNoContent, t)
  test.AssertResourceDoesNotExist(rpath, t)
}

func TestPROPFIND(t *testing.T) {
  // test when resource does not exist
  resp := doRequest("PROPFIND", "/foo/bar/", "", nil)
  test.AssertInt(resp.StatusCode, http.StatusNotFound, t)

  collection := "/test-data/propfind/"
  rName := "123-456-789.ics"
  rpath := collection + rName
  createResource(collection, rName, "BEGIN:VEVENT; SUMMARY:Party; END:VEVENT")

  currentUser := "foo-bar-baz"
  SetupUser(currentUser)

  // Next test will check for properties that have been found for the resource

  propfindXML := `
  <?xml version="1.0" encoding="utf-8" ?>
  <D:propfind xmlns:D="DAV:" xmlns:CS="http://calendarserver.org/ns/" xmlns:C="urn:ietf:params:xml:ns:caldav">
   <D:prop>
     <D:getetag/>
     <D:getcontenttype/>
     <D:getcontentlength/>
     <D:displayname/>
     <D:getlastmodified/>
     <D:owner/>
     <CS:getctag/>
     <D:principal-URL/>
     <D:principal-collection-set/>
     <C:calendar-user-address-set/>
     <C:calendar-home-set/>
     <D:resourcetype/>
     <D:current-user-principal/>
   </D:prop>
  </D:propfind>
  `
  expectedRespBody := fmt.Sprintf(`
  <?xml version="1.0" encoding="UTF-8"?>
  <D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
    <D:response>
      <D:href>/test-data/propfind/123-456-789.ics</D:href>
      <D:propstat>
        <D:prop>
          <D:getetag>?</D:getetag>
          <D:getcontenttype>text/calendar; component=vcalendar</D:getcontenttype>
          <D:getcontentlength>39</D:getcontentlength>
          <D:displayname>123-456-789.ics</D:displayname>
          <D:getlastmodified>?</D:getlastmodified>
          <D:owner>/test-data/</D:owner>
          <CS:getctag>?</CS:getctag>
          <D:principal-URL>
            <D:href>/test-data/propfind/123-456-789.ics</D:href>
          </D:principal-URL>
          <D:principal-collection-set>
            <D:href>/test-data/propfind/123-456-789.ics</D:href>
          </D:principal-collection-set>
          <C:calendar-user-address-set>
            <D:href>/test-data/propfind/123-456-789.ics</D:href>
          </C:calendar-user-address-set>
          <C:calendar-home-set>
            <D:href>/test-data/propfind/123-456-789.ics</D:href>
          </C:calendar-home-set>
          <D:resourcetype/>
          <D:current-user-principal>
            <D:href>/%s/</D:href>
          </D:current-user-principal>
        </D:prop>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
    </D:response>
  </D:multistatus>
  `, currentUser)

  resp = doRequest("PROPFIND", rpath, propfindXML, nil)
  respBody := readResponseBody(resp)
  test.AssertInt(resp.StatusCode, 207, t)
  test.AssertMultistatusXML(respBody, expectedRespBody, t)

  // Next test will check for properties that have not been found for the resource

  propfindXML = `
  <?xml version="1.0" encoding="utf-8" ?>
  <D:propfind xmlns:D="DAV:" xmlns:CS="http://calendarserver.org/ns/" xmlns:C="urn:ietf:params:xml:ns:caldav">
   <D:prop>
     <C:supported-calendar-component-set/>
   </D:prop>
  </D:propfind>
  `
  expectedRespBody = fmt.Sprintf(`
  <?xml version="1.0" encoding="UTF-8"?>
  <D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
    <D:response>
      <D:href>/test-data/propfind/123-456-789.ics</D:href>
      <D:propstat>
        <D:prop>
          <C:supported-calendar-component-set/>
        </D:prop>
        <D:status>HTTP/1.1 404 Not Found</D:status>
      </D:propstat>
    </D:response>
  </D:multistatus>
  `)

  resp = doRequest("PROPFIND", rpath, propfindXML, nil)
  respBody = readResponseBody(resp)
  test.AssertInt(resp.StatusCode, 207, t)
  test.AssertMultistatusXML(respBody, expectedRespBody, t)

  // Next tests will check the Depth feature

  headers := make(map[string]string)

  propfindXML = `
  <?xml version="1.0" encoding="utf-8" ?>
  <D:propfind xmlns:D="DAV:">
   <D:prop>
     <D:getcontenttype/>
   </D:prop>
  </D:propfind>
  `

  // test PROPFIND with depth 0
  headers["Depth"] = "0"

  expectedRespBody = `
  <?xml version="1.0" encoding="UTF-8"?>
  <D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
    <D:response>
      <D:href>/test-data/propfind</D:href>
      <D:propstat>
        <D:prop>
          <D:getcontenttype>text/calendar</D:getcontenttype>
        </D:prop>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
    </D:response>
  </D:multistatus>
  `

  resp = doRequest("PROPFIND", "/test-data/propfind/", propfindXML, headers)
  respBody = readResponseBody(resp)
  test.AssertMultistatusXML(respBody, expectedRespBody, t)

  // test PROPFIND with depth 1
  headers["Depth"] = "1"

  expectedRespBody = `
  <?xml version="1.0" encoding="UTF-8"?>
  <D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
    <D:response>
      <D:href>/test-data/propfind</D:href>
      <D:propstat>
        <D:prop>
          <D:getcontenttype>text/calendar</D:getcontenttype>
        </D:prop>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
    </D:response>
    <D:response>
      <D:href>/test-data/propfind/123-456-789.ics</D:href>
      <D:propstat>
        <D:prop>
          <D:getcontenttype>text/calendar; component=vcalendar</D:getcontenttype>
        </D:prop>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
    </D:response>
  </D:multistatus>
  `

  resp = doRequest("PROPFIND", "/test-data/propfind/", propfindXML, headers)
  respBody = readResponseBody(resp)
  test.AssertMultistatusXML(respBody, expectedRespBody, t)

  // the same test as before but without the trailing '/' on the collection's path
  resp = doRequest("PROPFIND", "/test-data/propfind", propfindXML, headers)
  respBody = readResponseBody(resp)
  test.AssertMultistatusXML(respBody, expectedRespBody, t)
}

func TestREPORT(t *testing.T) {
  collection := "/test-data/report/"
  r1Name := "123-456-789.ics"
  r1Data := "BEGIN:VEVENT\nSUMMARY:Party\nEND:VEVENT"
  createResource(collection, r1Name, r1Data)
  r2Name := "789-456-123.ics"
  r2Data := "BEGIN:VEVENT\nSUMMARY:Watch movies\nEND:VEVENT"
  createResource(collection, r2Name, r2Data)

  // Test 1: when the URL path points to a collection and passing the list of hrefs in the body.
  path := collection

  reportXML := `
  <?xml version="1.0" encoding="UTF-8"?>
  <C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
    <D:prop>
      <D:getetag/>
      <C:calendar-data/>
    </D:prop>
    <D:href>/test-data/report/123-456-789.ics</D:href>
    <D:href>/foo/bar</D:href>
    <D:href>/test-data/report/789-456-123.ics</D:href>
    <D:href>/test-data/report/000-000-000.ics</D:href>
  </C:calendar-multiget>
  `

  // the response should contain only the hrefs that belong to the collection.
  // the ones that do not belong are ignored.
  expectedRespBody := fmt.Sprintf(`
  <?xml version="1.0" encoding="UTF-8"?>
  <D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
    <D:response>
      <D:href>/test-data/report/123-456-789.ics</D:href>
      <D:propstat>
        <D:prop>
          <D:getetag>?</D:getetag>
          <C:calendar-data>%s</C:calendar-data>
        </D:prop>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
    </D:response>
    <D:response>
      <D:href>/test-data/report/789-456-123.ics</D:href>
      <D:propstat>
        <D:prop>
          <D:getetag>?</D:getetag>
          <C:calendar-data>%s</C:calendar-data>
        </D:prop>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
    </D:response>
    <D:response>
      <D:href>/test-data/report/000-000-000.ics</D:href>
      <D:status>HTTP/1.1 404 Not Found</D:status>
    </D:response>
  </D:multistatus>
  `, r1Data, r2Data)

  resp := doRequest("REPORT", path, reportXML, nil)
  respBody := readResponseBody(resp)
  test.AssertMultistatusXML(respBody, expectedRespBody, t)

  // Test 2: when the URL path points to an actual resource and using the same body as before
  path = collection + r1Name

  // the response should contain only the resource from the URL.
  // the rest are ignored
  expectedRespBody = fmt.Sprintf(`
  <?xml version="1.0" encoding="UTF-8"?>
  <D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
    <D:response>
      <D:href>/test-data/report/123-456-789.ics</D:href>
      <D:propstat>
        <D:prop>
          <D:getetag>?</D:getetag>
          <C:calendar-data>%s</C:calendar-data>
        </D:prop>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
    </D:response>
  </D:multistatus>
  `, r1Data)

  resp = doRequest("REPORT", path, reportXML, nil)
  respBody = readResponseBody(resp)
  test.AssertMultistatusXML(respBody, expectedRespBody, t)

  // Test 3: when the URL points to a collection and passing filter rules in the body
  path = collection

  reportXML = `
  <?xml version="1.0" encoding="UTF-8"?>
  <C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
    <D:prop>
      <D:getetag/>
    </D:prop>
    <C:filter>
      <C:comp-filter name="VCALENDAR">
        <C:comp-filter name="VEVENT">
          <C:prop-filter name="SUMMARY">
            <C:text-match>FOO</C:text-match>
          </C:prop-filter>
        </C:comp-filter>
      </C:comp-filter>
    </C:filter>
  </C:calendar-query>
  `

  createResource(collection, "football.ics", "BEGIN:VCALENDAR\nBEGIN:VEVENT\nSUMMARY:Football\nEND:VEVENT\nEND:VCALENDAR")
  createResource(collection, "volleyball.ics", "BEGIN:VCALENDAR\nBEGIN:VEVENT\nSUMMARY:Volleyball\nEND:VEVENT\nEND:VCALENDAR")

  expectedRespBody = `
  <?xml version="1.0" encoding="UTF-8"?>
  <D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
    <D:response>
      <D:href>/test-data/report/football.ics</D:href>
      <D:propstat>
        <D:prop>
          <D:getetag>?</D:getetag>
        </D:prop>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
    </D:response>
  </D:multistatus>
  `

  resp = doRequest("REPORT", path, reportXML, nil)
  respBody = readResponseBody(resp)
  test.AssertMultistatusXML(respBody, expectedRespBody, t)
}

// ================ FUNCS ========================

func doRequest(method, path, body string, headers map[string]string) *http.Response {
  client := &http.Client{}
  url := "http://localhost:" + TEST_SERVER_PORT + path
  req, err := http.NewRequest(method, url, strings.NewReader(body))
  panicerr(err)
  for k, v := range headers {
    req.Header.Add(k, v)
  }
  resp, err := client.Do(req)
  panicerr(err)

  return resp
}

func readResponseBody(resp *http.Response) string {
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  panicerr(err)

  return string(body)
}

func readResource(path string) string {
  pwd, _ := os.Getwd()
  data, err := ioutil.ReadFile(pwd + path)
  panicerr(err)

  return string(data)
}

func createResource(collection, rName, data string) {
  pwd, _ := os.Getwd()
  err := os.MkdirAll(pwd + collection, os.ModePerm)
  panicerr(err)
  f, err := os.Create(pwd + collection + rName)
  panicerr(err)
  f.WriteString(data)
}

func panicerr(err error) {
  if err != nil {
    panic(err)
  }
}
