package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/samedi/caldav-go/ixml"
	"github.com/samedi/caldav-go/test"
)

// Test 1: when the URL path points to a collection and passing the list of hrefs in the body.
func TestHandle1(t *testing.T) {
	stg := test.NewFakeStorage()
	r1Data := "BEGIN:VEVENT\nSUMMARY:Party\nEND:VEVENT"
	stg.AddFakeResource("/test-data/report/", "123-456-789.ics", r1Data)
	r2Data := "BEGIN:VEVENT\nSUMMARY:Watch movies\nEND:VEVENT"
	stg.AddFakeResource("/test-data/report/", "789-456-123.ics", r2Data)

	handler := reportHandler{
		handlerData{
			requestPath: "/test-data/report/",
			requestBody: `
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
			`,
			response: NewResponse(),
			storage:  stg,
		},
	}

	// The response should contain only the hrefs that belong to the collection.
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
	`, ixml.EscapeText(r1Data), ixml.EscapeText(r2Data))

	resp := handler.Handle()
	test.AssertMultistatusXML(resp.Body, expectedRespBody, t)
}

// Test 2: when the URL path points to an actual resource.
func TestHandle2(t *testing.T) {
	stg := test.NewFakeStorage()
	r1Data := "BEGIN:VEVENT\nSUMMARY:Party\nEND:VEVENT"
	stg.AddFakeResource("/test-data/report/", "123-456-789.ics", r1Data)
	stg.AddFakeResource("/test-data/report/", "789-456-123.ics", "BEGIN:VEVENT\nSUMMARY:Watch movies\nEND:VEVENT")

	handler := reportHandler{
		handlerData{
			requestPath: "/test-data/report/123-456-789.ics",
			requestBody: `
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
			`,
			response: NewResponse(),
			storage:  stg,
		},
	}

	// The response should contain only the resource from the URL. The rest are ignored
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
  </D:multistatus>
  `, ixml.EscapeText(r1Data))

	resp := handler.Handle()
	test.AssertMultistatusXML(resp.Body, expectedRespBody, t)
}

// Test 3: when the URL points to a collection and passing filter rules in the body.
func TestHandle3(t *testing.T) {
	stg := test.NewFakeStorage()
	stg.AddFakeResource("/test-data/report/", "volleyball.ics", "BEGIN:VCALENDAR\nBEGIN:VEVENT\nSUMMARY:Volleyball\nEND:VEVENT\nEND:VCALENDAR")
	r1Data := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nSUMMARY:Football\nEND:VEVENT\nEND:VCALENDAR"
	stg.AddFakeResource("/test-data/report/", "football.ics", r1Data)
	r2Data := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nSUMMARY:Footsteps\nEND:VEVENT\nEND:VCALENDAR"
	stg.AddFakeResource("/test-data/report/", "footsteps.ics", r2Data)

	handler := reportHandler{
		handlerData{
			requestPath: "/test-data/report/",
			requestBody: `
			<?xml version="1.0" encoding="UTF-8"?>
			<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
				<D:prop>
					<D:getetag/>
					<C:calendar-data/>
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
			`,
			response: NewResponse(),
			storage:  stg,
		},
	}

	expectedRespBody := fmt.Sprintf(`
	<?xml version="1.0" encoding="UTF-8"?>
	<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
		<D:response>
			<D:href>/test-data/report/football.ics</D:href>
			<D:propstat>
				<D:prop>
					<D:getetag>?</D:getetag>
					<C:calendar-data>%s</C:calendar-data>
				</D:prop>
				<D:status>HTTP/1.1 200 OK</D:status>
			</D:propstat>
		</D:response>
		<D:response>
			<D:href>/test-data/report/footsteps.ics</D:href>
			<D:propstat>
				<D:prop>
					<D:getetag>?</D:getetag>
					<C:calendar-data>%s</C:calendar-data>
				</D:prop>
				<D:status>HTTP/1.1 200 OK</D:status>
			</D:propstat>
		</D:response>
	</D:multistatus>
	`, ixml.EscapeText(r1Data), ixml.EscapeText(r2Data))

	resp := handler.Handle()
	test.AssertMultistatusXML(resp.Body, expectedRespBody, t)
}

// Test 4: when making a request with a `Prefer` header.
func TestHandle4(t *testing.T) {
	httpHeader := http.Header{}
	httpHeader.Add("Prefer", "return=minimal")

	stg := test.NewFakeStorage()
	stg.AddFakeResource("/test-data/report/", "123-456-789.ics", "BEGIN:VEVENT\nSUMMARY:Party\nEND:VEVENT")

	handler := reportHandler{
		handlerData{
			requestPath: "/test-data/report/",
			requestBody: `
			<?xml version="1.0" encoding="UTF-8"?>
		  <C:calendar-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
		    <D:prop>
		      <D:getetag/>
		      <unknown-property/>
		    </D:prop>
		    <D:href>/test-data/report/123-456-789.ics</D:href>
		  </C:calendar-multiget>
		  `,
			headers:  headers{httpHeader},
			response: NewResponse(),
			storage:  stg,
		},
	}

	// The response should omit all the <propstat> nodes with status 404.
	expectedRespBody := fmt.Sprintf(`
  <?xml version="1.0" encoding="UTF-8"?>
  <D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
    <D:response>
      <D:href>/test-data/report/123-456-789.ics</D:href>
      <D:propstat>
        <D:prop>
          <D:getetag>?</D:getetag>
        </D:prop>
        <D:status>HTTP/1.1 200 OK</D:status>
      </D:propstat>
    </D:response>
  </D:multistatus>`)

	resp := handler.Handle()
	test.AssertMultistatusXML(resp.Body, expectedRespBody, t)

	if test.AssertInt(len(resp.Header["Preference-Applied"]), 1, t) {
		test.AssertStr(resp.Header.Get("Preference-Applied"), "return=minimal", t)
	}
}
