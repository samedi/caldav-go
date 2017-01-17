package handlers

import (
  "testing"
  "encoding/xml"

  "git.samedi.cc/ferraz/caldav/test"
)

// Tests the XML serialization when the option to return a minimal content is set or not.
func TestToXML(t *testing.T)  {
  ms := new(multistatusResp)
  propstats := msPropstats{
    200: msProps{
      msProp{Tag: xml.Name{Local: "getetag"}},
    },
    404: msProps{
      msProp{Tag: xml.Name{Local: "owner"}},
    },
  }
  ms.Responses = append(ms.Responses, msResponse{
    Href: "/123",
    Found: true,
    Propstats: propstats,
  })

  // First test when the minimal flag is false. It should return
  // all serialize props, including the ones not found

  ms.Minimal = false
  expected := `
  <?xml version="1.0" encoding="UTF-8"?>
  <D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
    <D:response>
      <D:href>/123</D:href>
      <D:propstat>
        <D:prop>
          <getetag/>
        </D:prop>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
      <D:propstat>
        <D:prop>
          <owner/>
        </D:prop>
        <D:status>HTTP/1.1 404 Not Found</D:status>
      </D:propstat>
    </D:response>
  </D:multistatus>`

  test.AssertMultistatusXML(ms.ToXML(), expected, t)

  // Now test when the minimal flag is true. It should omit
  // all props that were not found

  ms.Minimal = true
  expected = `
  <?xml version="1.0" encoding="UTF-8"?>
  <D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
    <D:response>
      <D:href>/123</D:href>
      <D:propstat>
        <D:prop>
          <getetag/>
        </D:prop>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
    </D:response>
  </D:multistatus>`

  test.AssertMultistatusXML(ms.ToXML(), expected, t)

  // adding this just to make sure that the following test does not affect the other DAV:responses
  ms.Responses = append(ms.Responses, msResponse{
    Href: "/456",
    Found: false,
  })

  // If in the propstats there are only not found props, then instead of having an empty
  // <DAV:propstat> node, the expected should be as the below.

  expected = `
  <?xml version="1.0" encoding="UTF-8"?>
  <D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
    <D:response>
      <D:href>/123</D:href>
      <D:propstat>
        <D:prop/>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
    </D:response>
    <D:response>
      <D:href>/456</D:href>
      <D:status>HTTP/1.1 404 Not Found</D:status>
    </D:response>
  </D:multistatus>`

  delete(propstats, 200)

  test.AssertMultistatusXML(ms.ToXML(), expected, t)
}
