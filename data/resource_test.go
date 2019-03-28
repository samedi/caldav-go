package data

import (
	"fmt"
	"testing"
	"time"
)

func TestNewResource(t *testing.T) {
	res := NewResource("/foo///bar/123.ics//", FakeResourceAdapter{})

	if res.Name != "123.ics" {
		t.Error("Expected name to be 123.ics, got", res.Name)
	}

	// it cleans (sanitize) the path
	if res.Path != "/foo/bar/123.ics" {
		t.Error("Expected name to be /foo/bar/123.ics, got", res.Path)
	}
}

func TestIsCollection(t *testing.T) {
	adp := new(FakeResourceAdapter)
	res := NewResource("/foo/bar/", adp)

	adp.collection = false
	if res.IsCollection() {
		t.Error("Resource should not be a collection")
	}

	adp.collection = true
	if !res.IsCollection() {
		t.Error("Resource should be a collection")
	}
}

func TestIsPrincipal(t *testing.T) {
	res := NewResource("/foo", FakeResourceAdapter{})
	if !res.IsPrincipal() {
		t.Error("Resource should be principal")
	}

	res = NewResource("/foo/bar", FakeResourceAdapter{})
	if res.IsPrincipal() {
		t.Error("Resource should not be principal")
	}
}

func TestComponentName(t *testing.T) {
	adp := new(FakeResourceAdapter)
	res := NewResource("/foo", adp)

	adp.collection = false
	if res.ComponentName() != "VEVENT" {
		t.Error("Resource should be a VEVENT")
	}

	adp.collection = true
	if res.ComponentName() != "VCALENDAR" {
		t.Error("Resource should be a VCALENDAR")
	}
}

func TestEtag(t *testing.T) {
	adp := new(FakeResourceAdapter)
	res := NewResource("/foo", adp)

	adp.collection = false
	adp.etag = "1111"
	etag, found := res.GetEtag()
	if etag != "1111" || !found {
		t.Error("Etag should be 1111")
	}

	adp.etag = "2222"
	etag, found = res.GetEtag()
	if etag != "2222" || !found {
		t.Error("Etag should be 2222")
	}

	adp.collection = true
	etag, found = res.GetEtag()
	if etag != "" || found {
		t.Error("Collections should not have etags associated")
	}
}

func TestContentType(t *testing.T) {
	adp := new(FakeResourceAdapter)
	res := NewResource("/foo", adp)

	adp.collection = false
	ctype, found := res.GetContentType()
	if ctype != "text/calendar; component=vcalendar" || !found {
		t.Error("Content Type should be `text/calendar; component=vcalendar`")
	}

	adp.collection = true
	ctype, found = res.GetContentType()
	if ctype != "text/calendar" || !found {
		t.Error("Content Type should be `text/calendar`")
	}
}

func TestDisplayName(t *testing.T) {
	res := NewResource("foo/bar", FakeResourceAdapter{})

	// it just returns the resource Name
	name, found := res.GetDisplayName()
	if name != res.Name || !found {
		t.Error("Display name should be", res.Name)
	}
}

func TestContentData(t *testing.T) {
	adp := new(FakeResourceAdapter)
	res := NewResource("/foo", adp)

	adp.contentData = "EVENT;"
	adp.collection = false

	data, found := res.GetContentData()
	if data != "EVENT;" || !found {
		t.Error("Content data should be EVENT;")
	}
}

func TestContentLength(t *testing.T) {
	adp := new(FakeResourceAdapter)
	res := NewResource("foo", adp)

	adp.contentSize = 42

	adp.collection = false
	clength, found := res.GetContentLength()
	if clength != "42" || !found {
		t.Error("Content length should be 42")
	}

	adp.collection = true
	clength, found = res.GetContentLength()
	if clength != "" || found {
		t.Error("Content length should be marked as not found for collections")
	}
}

func TestLastModified(t *testing.T) {
	adp := new(FakeResourceAdapter)
	res := NewResource("foo", adp)

	adp.modtime = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	timeFormat := "2006-01-02 15:04:05"
	lastmod, found := res.GetLastModified(timeFormat)

	if lastmod != "2009-11-10 23:00:00" || !found {
		t.Error("Last modified should be equal `2009-11-10 23:00:00`")
	}
}

func TestOwnerPath(t *testing.T) {
	res := NewResource("/foo", FakeResourceAdapter{})
	owner, found := res.GetOwnerPath()
	if owner != "" || found {
		t.Error("Path owner should have been empty")
	}

	res = NewResource("/foo/bar", FakeResourceAdapter{})
	owner, found = res.GetOwnerPath()
	if owner != "/foo/" || !found {
		t.Error("Path owner should have been `/foo/`")
	}
}

func TestStartEndTimesUTC(t *testing.T) {
	newResource := func(timeInfo string) Resource {
		adp := new(FakeResourceAdapter)
		adp.contentData = fmt.Sprintf(`
    BEGIN:VCALENDAR
    BEGIN:VTIMEZONE
    TZID:Europe/Berlin
    BEGIN:DAYLIGHT
    TZOFFSETFROM:+0100
    TZOFFSETTO:+0200
    TZNAME:CEST
    DTSTART:19700329T020000
    RRULE:FREQ=YEARLY;BYDAY=-1SU;BYMONTH=3
    END:DAYLIGHT
    BEGIN:STANDARD
    TZOFFSETFROM:+0200
    TZOFFSETTO:+0100
    TZNAME:CET
    DTSTART:19701025T030000
    RRULE:FREQ=YEARLY;BYDAY=-1SU;BYMONTH=10
    END:STANDARD
    END:VTIMEZONE
    BEGIN:VEVENT
    %s
    END:VEVENT
    END:VCALENDAR
    `, timeInfo)

		return NewResource("/foo", adp)
	}

	assertTime := func(target, expected time.Time) {
		if !(target == expected) {
			t.Error("Wrong resource time. Expected:", expected, "Got:", target)
		}
	}

	res := newResource(`
    DTSTART;TZID=Europe/Berlin:20160914T170000
    DTEND;TZID=Europe/Berlin:20160915T180000
  `)

	// test start time in UTC
	assertTime(res.StartTimeUTC(), time.Date(2016, 9, 14, 15, 0, 0, 0, time.UTC))
	// test end time in UTC
	assertTime(res.EndTimeUTC(), time.Date(2016, 9, 15, 16, 0, 0, 0, time.UTC))

	// test `end` time in UTC when DTEND is not present but DURATION is
	// in this case, the `end` time has to be DTSTART + DURATION

	res = newResource(`
    DTSTART;TZID=Europe/Berlin:20160914T170000
    DURATION:PT3H10M1S
  `)

	assertTime(res.EndTimeUTC(), time.Date(2016, 9, 14, 18, 10, 1, 0, time.UTC))

	res = newResource(`
    DTSTART;TZID=Europe/Berlin:20160914T170000
    DURATION:PT10M
  `)

	assertTime(res.EndTimeUTC(), time.Date(2016, 9, 14, 15, 10, 0, 0, time.UTC))

	res = newResource(`
    DTSTART;TZID=Europe/Berlin:20160914T170000
    DURATION:PT1S
  `)

	assertTime(res.EndTimeUTC(), time.Date(2016, 9, 14, 15, 0, 1, 0, time.UTC))

	// test end time in UTC when DTEND and DURATION are not present
	// in this case, the `end` time has to be equals to DTSTART time

	res = newResource(`
    DTSTART;TZID=Europe/Berlin:20160914T170000
  `)

	assertTime(res.EndTimeUTC(), time.Date(2016, 9, 14, 15, 0, 0, 0, time.UTC))
}

func TestProperties(t *testing.T) {
	adp := new(FakeResourceAdapter)
	res := NewResource("/foo", adp)

	adp.contentData = `
  BEGIN:VCALENDAR
  BEGIN:VEVENT
  DTSTART:20160914T170000
  END:VEVENT
  END:VCALENDAR
  `

	// asserts that the resource does not have the property VEVENT->DTEND
	if res.HasProperty("VEVENT", "DTEND") || res.GetPropertyValue("VEVENT", "DTEND") != "" {
		t.Error("Resource should not have the property")
	}

	adp.contentData = `
  BEGIN:VCALENDAR
  BEGIN:VEVENT
  DTSTART:20160914T170000
  DTEND:20160915T170000
  END:VEVENT
  END:VCALENDAR
  `

	// asserts that the resource has the property VEVENT->DTEND and it returns the correct value
	if !res.HasProperty("VEVENT", "DTEND") || res.GetPropertyValue("VEVENT", "DTEND") != "20160915T170000" {
		t.Error("Resource should have the property")
	}

	// asserts that the resource has the property VCALENDAR->VEVENT->DTEND and it returns the correct value
	// (VCALENDAR is ignored when passing the prop path)
	if !res.HasProperty("VCALENDAR", "VEVENT", "DTEND") || res.GetPropertyValue("VCALENDAR", "VEVENT", "DTEND") == "" {
		t.Error("Resource should have the property")
	}
}

func TestPropertyParams(t *testing.T) {
	adp := new(FakeResourceAdapter)
	res := NewResource("/foo", adp)

	adp.contentData = `
  BEGIN:VCALENDAR
  BEGIN:VEVENT
  ATTENDEE:FOO
  END:VEVENT
  END:VCALENDAR
  `

	// asserts that the resource does not have the property param VEVENT->ATTENDEE->PARTSTAT
	if res.HasPropertyParam("VEVENT", "ATTENDEE", "PARTSTAT") || res.GetPropertyParamValue("VEVENT", "ATTENDEE", "PARTSTAT") != "" {
		t.Error("Resouce should not have the property param")
	}

	adp.contentData = `
  BEGIN:VCALENDAR
  BEGIN:VEVENT
  ATTENDEE;PARTSTAT=NEEDS-ACTION:FOO
  END:VEVENT
  END:VCALENDAR
  `

	// asserts that the resource has the property param VEVENT->ATTENDEE->PARTSTAT and it returns the correct value
	if !res.HasPropertyParam("VEVENT", "ATTENDEE", "PARTSTAT") || res.GetPropertyParamValue("VEVENT", "ATTENDEE", "PARTSTAT") != "NEEDS-ACTION" {
		t.Error("Resource should have the property param")
	}

	// asserts that the resource has the property VEVENT->ATTENDEE->PARTSTAT and it returns the correct value
	// (VCALENDAR is ignored when passing the prop path)
	if !res.HasPropertyParam("VCALENDAR", "VEVENT", "ATTENDEE", "PARTSTAT") || res.GetPropertyParamValue("VCALENDAR", "VEVENT", "ATTENDEE", "PARTSTAT") != "NEEDS-ACTION" {
		t.Error("Resource should have the property param")
	}
}

func TestRecurrenceOnce(t *testing.T) {
	adp := new(FakeResourceAdapter)
	res := NewResource("/foo", adp)

	adp.contentData = `
  BEGIN:VCALENDAR
  BEGIN:VEVENT
  DTSTART:20160914T170000Z
  DTEND:20160914T180000Z
  RRULE: FREQ=DAILY;COUNT=1
  END:VEVENT
  END:VCALENDAR
  `
  if len(res.Recurrences()) != 1 {
    t.Error("Expected 1 Recurrencies got, ", len(res.Recurrences()))
  }

    if (res.Recurrences()[0].StartTime != time.Date(2016,9,15,17,0,0,0, time.UTC)) {
        t.Error("Unexpected Start time, ", res.Recurrences()[0].StartTime);
    }
}

func TestRecurrenceCountInterval(t *testing.T) {
	adp := new(FakeResourceAdapter)
	res := NewResource("/foo", adp)

	adp.contentData = `
  BEGIN:VCALENDAR
  BEGIN:VEVENT
  DTSTART:20160914T170000Z
  DTEND:20160914T180000Z
  RRULE: FREQ=DAILY;COUNT=2;INTERVAL=2
  END:VEVENT
  END:VCALENDAR
  `
  if len(res.Recurrences()) != 2 {
    t.Error("Expected 2 Recurrencies got, ", len(res.Recurrences()))
  }

    if (res.Recurrences()[1].StartTime != time.Date(2016,9,18,17,0,0,0, time.UTC)) {
        t.Error("Unexpected Start time, ", res.Recurrences()[1].StartTime);
    }
}


type FakeResourceAdapter struct {
	collection  bool
	etag        string
	contentData string
	contentSize int64
	modtime     time.Time
}

func (adp FakeResourceAdapter) IsCollection() bool {
	return adp.collection
}

func (adp FakeResourceAdapter) GetContent() string {
	return adp.contentData
}

func (adp FakeResourceAdapter) GetContentSize() int64 {
	return adp.contentSize
}

func (adp FakeResourceAdapter) CalculateEtag() string {
	return adp.etag
}

func (adp FakeResourceAdapter) GetModTime() time.Time {
	return adp.modtime
}
