package data

import (
  "testing"
  "time"
  "strings"
)

func TestParseFilter(t *testing.T) {
  filter, err := ParseResourceFilters(`<C:filter xmlns:C="urn:ietf:params:xml:ns:caldav"></C:filter>`)
  if err != nil {
    t.Error("Parsing filter from a valid XML returned an error:", err)
  }

  if filter == nil {
    t.Error("Parsing filter from a valid XML returned a nil filter")
  }

  invalidXMLs := []string{
    `<C:filterzzzzz xmlns:C="urn:ietf:params:xml:ns:caldav"></C:filterzzzzz>`,
    `<C:comp-filter xmlns:C="urn:ietf:params:xml:ns:caldav"></C:comp-filter>`,
  }

  for _, invalidXML := range invalidXMLs {
    filter, err = ParseResourceFilters(invalidXML)
    if err == nil {
      t.Error("Parsing filter from an invalid XML should return an error")
    }
  }
}

func TestMatch1(t *testing.T) {
  filterXML := `
  <filter>
  </filter>`

  assertFilterDoesNotMatch(filterXML, FakeResource{}, t)
}

func TestMatch2(t *testing.T) {
  filterXML := `
  <filter>
    <comp-filter name="VCALENDAR">
    </comp-filter>
  </filter>`

  // <comp-filter name="VEVENT"> matches if the resource's component type is "VCALENDAR"
  assertFilterMatch(filterXML, FakeResource{comp: "VCALENDAR"}, t)
  assertFilterDoesNotMatch(filterXML, FakeResource{comp: "VEVENT"}, t)
}

func TestMatch3(t *testing.T) {
  filterXML := `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
      </comp-filter>
    </comp-filter>
  </filter>`

  // <comp-filter name="VEVENT"> matches if the resource's component type is "VEVENT"
  assertFilterMatch(filterXML, FakeResource{comp: "VEVENT"}, t)
  assertFilterDoesNotMatch(filterXML, FakeResource{comp: "VCALENDAR"}, t)
}

func TestMatch4(t *testing.T) {
  filterXML := `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <is-not-defined/>
      </comp-filter>
    </comp-filter>
  </filter>`

  // the `is-not-defined` inside the `comp-filter` works as a boolean `not`,
  // therefore matching if the resource's component type is NOT "VEVENT"
  assertFilterMatch(filterXML, FakeResource{comp: "VCALENDAR"}, t)
  assertFilterDoesNotMatch(filterXML, FakeResource{comp: "VEVENT"}, t)
}

func TestMatch5(t *testing.T) {
  filterXML := `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <time-range/>
      </comp-filter>
    </comp-filter>
  </filter>`

  // A `time-range` without `start` and `end` properties is not valid.
  assertFilterDoesNotMatch(filterXML, FakeResource{}, t)
}

func TestMatch6(t *testing.T) {
  filterXML := `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <time-range start="20160914T000000Z" end="20160916T000000Z"/>
      </comp-filter>
    </comp-filter>
  </filter>`

  // set of tests when the resource's `start` and `end` property are different

  // out of the interval - doesnt match!
  assertFilterDoesNotMatch(filterXML, FakeResource{start: "20160913T000000Z", end: "20160914T000000Z"}, t)
  // resource's `end` is inside the interval - match!
  assertFilterMatch(filterXML, FakeResource{start: "20160913T000000Z", end: "20160915T000000Z"}, t)
  // resource's `start` is inside the interval - match!
  assertFilterMatch(filterXML, FakeResource{start: "20160915T000000Z", end: "20160917T000000Z"}, t)
  // out of the interval - doesnt match!
  assertFilterDoesNotMatch(filterXML, FakeResource{start: "20160916T000000Z", end: "20160917T000000Z"}, t)

  // set of tests when the resource's `start` and `end` property are equal

  // out of the interval - doesnt match!
  assertFilterDoesNotMatch(filterXML, FakeResource{start: "20160913T000000Z", end: "20160913T000000Z"}, t)
  // in the interval - match!
  assertFilterMatch(filterXML, FakeResource{start: "20160914T000000Z", end: "20160914T000000Z"}, t)
  // in the interval - match!
  assertFilterMatch(filterXML, FakeResource{start: "20160915T000000Z", end: "20160915T000000Z"}, t)
  // out of the interval - doesnt match!
  assertFilterDoesNotMatch(filterXML, FakeResource{start: "20160916T000000Z", end: "20160916T000000Z"}, t)

  // set of tests to check for the resource's recurrences. If any of the recurrences overlaps the interval,
  // it should match the filter.

  res := FakeResource{start: "20140913T000000Z", end: "20140915T000000Z"}
  // out of the interval - doesnt match!
  assertFilterDoesNotMatch(filterXML, res, t)

  res.addRecurrence("20150913T000000Z", "20150915T000000Z")
  // recurrence is out of the interval - still doesnt match!
  assertFilterDoesNotMatch(filterXML, res, t)

  res.addRecurrence("20160913T000000Z", "20160915T000000Z")
  // recurrence is in the interval - match!
  assertFilterMatch(filterXML, res, t)
}

func TestMatch7(t *testing.T) {
  // when the `end` attribute is not defined, it is open ended (to infinity)
  filterXML := `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <time-range start="20160914T000000Z"/>
      </comp-filter>
    </comp-filter>
  </filter>`

  // out of the interval - doesnt match!
  assertFilterDoesNotMatch(filterXML, FakeResource{start: "20160912T000000Z", end: "20160913T000000Z"}, t)
  // in the interval - match!
  assertFilterMatch(filterXML, FakeResource{start: "20160912T000000Z", end: "20170913T000000Z"}, t)
}

func TestMatch8(t *testing.T) {
  // when the `start` attribute is not defined, it is open ended (to infinity)
  filterXML := `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <time-range end="20160916T000000Z"/>
      </comp-filter>
    </comp-filter>
  </filter>`

  // out of the interval - doesnt match!
  assertFilterDoesNotMatch(filterXML, FakeResource{start: "20160917T000000Z", end: "20160918T000000Z"}, t)
  // in the interval - match!
  assertFilterMatch(filterXML, FakeResource{start: "20150917T000000Z", end: "20160918T000000Z"}, t)
}

func TestMatch9(t *testing.T) {
  filterXML := `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <prop-filter name="UID">
        </prop-filter>
      </comp-filter>
    </comp-filter>
  </filter>`

  res := FakeResource{}
  // does not contain the property UID - doesnt match!
  assertFilterDoesNotMatch(filterXML, res, t)
  // now contains the property UID - match!
  res.addProperty("VCALENDAR:VEVENT:UID", "")
  assertFilterMatch(filterXML, res, t)
}

func TestMatch10(t *testing.T) {
  filterXML := `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <prop-filter name="UID">
          <is-not-defined/>
        </prop-filter>
      </comp-filter>
    </comp-filter>
  </filter>`

  // the `is-not-defined` works as a boolean `not`,
  // therefore matching if the resource DOES NOT have the property UID
  res := FakeResource{}
  assertFilterMatch(filterXML, res, t)
  res.addProperty("VCALENDAR:VEVENT:UID", "")
  assertFilterDoesNotMatch(filterXML, res, t)
}

func TestMatch11(t *testing.T) {
  filterXML := `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <prop-filter name="UID">
          <text-match>
            @ExAmplE.coM
          </text-match>
        </prop-filter>
      </comp-filter>
    </comp-filter>
  </filter>`

  res := FakeResource{}
  // the resource does not have the property - doesnt match!
  assertFilterDoesNotMatch(filterXML, res, t)
  // the property content does not have the substring - doesnt match!
  res.addProperty("VCALENDAR:VEVENT:UID", "DC6C50A017428C5216A2F1CD@foobar.com")
  assertFilterDoesNotMatch(filterXML, res, t)
  // the property content has the substring - match!
  res.addProperty("VCALENDAR:VEVENT:UID", "DC6C50A017428C5216A2F1CD@example.com")
  assertFilterMatch(filterXML, res, t)

  // with `negate-condition` as "no"
  filterXML = `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <prop-filter name="UID">
          <text-match negate-condition="no">
            @ExAmplE.coM
          </text-match>
        </prop-filter>
      </comp-filter>
    </comp-filter>
  </filter>`
  // the property content has the substring - match!
  assertFilterMatch(filterXML, res, t)

  // with `negate-condition` as "yes"
  filterXML = `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <prop-filter name="UID">
          <text-match negate-condition="yes">
            @ExAmplE.coM
          </text-match>
        </prop-filter>
      </comp-filter>
    </comp-filter>
  </filter>`
  // the property content has the substring - doesnt match!
  assertFilterDoesNotMatch(filterXML, res, t)
}

func TestMatch12(t *testing.T) {
  filterXML := `
  <filter>
   <comp-filter name="VCALENDAR">
     <comp-filter name="VEVENT">
       <prop-filter name="ATTENDEE">
         <param-filter name="PARTSTAT">
         </param-filter>
       </prop-filter>
     </comp-filter>
   </comp-filter>
  </filter>`

  res := FakeResource{}
  // does not contain the property param ATTENDEE:PARTSTAT - doesnt match!
  assertFilterDoesNotMatch(filterXML, res, t)
  // now contains the property param ATTENDEE:PARTSTAT - match!
  res.addPropertyParam("VCALENDAR:VEVENT:ATTENDEE:PARTSTAT", "")
  assertFilterMatch(filterXML, res, t)
}

func TestMatch13(t *testing.T) {
  filterXML := `
  <filter>
   <comp-filter name="VCALENDAR">
     <comp-filter name="VEVENT">
       <prop-filter name="ATTENDEE">
         <param-filter name="PARTSTAT">
           <is-not-defined/>
         </param-filter>
       </prop-filter>
     </comp-filter>
   </comp-filter>
  </filter>`

  // the `is-not-defined` works as a boolean `not`,
  // therefore matching if the resource DOES NOT have the property param ATTENDEE:PARTSTAT
  res := FakeResource{}
  assertFilterMatch(filterXML, res, t)
  res.addPropertyParam("VCALENDAR:VEVENT:ATTENDEE:PARTSTAT", "")
  assertFilterDoesNotMatch(filterXML, res, t)
}

func TestMatch14(t *testing.T) {
  filterXML := `
  <filter>
   <comp-filter name="VCALENDAR">
     <comp-filter name="VEVENT">
       <prop-filter name="ATTENDEE">
         <param-filter name="PARTSTAT">
           <text-match>NEEDS ACTION</text-match>
         </param-filter>
       </prop-filter>
     </comp-filter>
   </comp-filter>
  </filter>`

  res := FakeResource{}
  // the resource does not have the property param - doesnt match!
  assertFilterDoesNotMatch(filterXML, res, t)
  // the property param content does not have the substring - doesnt match!
  res.addPropertyParam("VCALENDAR:VEVENT:ATTENDEE:PARTSTAT", "FOO BAR")
  assertFilterDoesNotMatch(filterXML, res, t)
  // the property param content has the substring - match!
  res.addPropertyParam("VCALENDAR:VEVENT:ATTENDEE:PARTSTAT", "FOO BAR NEEDS ACTION")
  assertFilterMatch(filterXML, res, t)
}

func TestGetTimeRangeFilter(t *testing.T) {
  // First testing when the filters contain a time-range filter
  filterXML := `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
        <time-range start="20150916T000000Z" end="20160916T000000Z"/>
      </comp-filter>
    </comp-filter>
  </filter>`
  filters, err := ParseResourceFilters(filterXML); panicerr(err)

  timeRange := filters.GetTimeRangeFilter()

  if timeRange == nil {
    t.Error("should have returned the time range filter, not nil.")
    return
  }

  if timeRange.Attr("start") != "20150916T000000Z" || timeRange.Attr("end") != "20160916T000000Z" {
    t.Error("should have returned the correct time range filter with the correct attributes")
  }

  // Now testing when the filters DO NOT contain a time-range filter
  filterXML = `
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VEVENT">
      </comp-filter>
    </comp-filter>
  </filter>`

  filters, err = ParseResourceFilters(filterXML); panicerr(err)

  timeRange = filters.GetTimeRangeFilter()

  if timeRange != nil {
    t.Error("should not have returned time range filter")
  }
}

func assertFilterMatch(filterXML string, res FakeResource, t *testing.T) {
  filter, err := ParseResourceFilters(filterXML); panicerr(err)
  if !filter.Match(&res) {
    t.Error("Filter should have been matched. Filter XML:", filterXML)
  }
}

func assertFilterDoesNotMatch(filterXML string, res FakeResource, t *testing.T) {
  filter, err := ParseResourceFilters(filterXML); panicerr(err)
  if filter.Match(&res) {
    t.Error("Filter should not have been matched. Filter XML:", filterXML)
  }
}

// Fake resource, that implements the ResourceInterface, to be used throughout the tests.
type FakeResource struct {
  comp  string
  start string
  end   string
  recurrences     []ResourceRecurrence
  properties      map[string]string
  propertyParams  map[string]string
}

func (r *FakeResource) ComponentName() string {
  if r.comp == "" {
    return "VEVENT"
  }

  return r.comp
}

func (r *FakeResource) StartTimeUTC() time.Time {
  return r.parseTime(r.start)
}

func (r *FakeResource) EndTimeUTC() time.Time {
  return r.parseTime(r.end)
}

func (r *FakeResource) Recurrences() []ResourceRecurrence {
  return r.recurrences
}

func (r *FakeResource) addRecurrence(startStr string, endStr string) {
  if r.recurrences == nil {
    r.recurrences = []ResourceRecurrence{}
  }

  r.recurrences = append(r.recurrences, ResourceRecurrence{
    StartTime: r.parseTime(startStr),
    EndTime:   r.parseTime(endStr),
  })
}

func (r *FakeResource) HasProperty(propPath... string) bool {
  if r.properties == nil {
    return false
  }

  propKey := r.getPropParamKey(propPath...)
  _, found := r.properties[propKey]
  return found
}

func (r *FakeResource) GetPropertyValue(propPath... string) string {
  if r.properties == nil {
    return ""
  }

  propKey := r.getPropParamKey(propPath...)
  return r.properties[propKey]
}

func (r *FakeResource) HasPropertyParam(paramPath... string) bool {
  if r.propertyParams == nil {
    return false
  }

  paramKey := r.getPropParamKey(paramPath...)
  _, found := r.propertyParams[paramKey]
  return found
}

func (r *FakeResource) GetPropertyParamValue(paramPath... string) string {
  if r.propertyParams == nil {
    return ""
  }

  paramKey := r.getPropParamKey(paramPath...)
  return r.propertyParams[paramKey]
}

func (r *FakeResource) addProperty(propPath string, propValue string) {
  if r.properties == nil {
    r.properties = make(map[string]string)
  }

  r.properties[propPath] = propValue
}

func (r *FakeResource) addPropertyParam(paramPath string, paramValue string) {
  if r.propertyParams == nil {
    r.propertyParams = make(map[string]string)
  }

  r.propertyParams[paramPath] = paramValue
}

func (r *FakeResource) getPropParamKey(ppath... string) string {
  return strings.Join(ppath, ":")
}

func (r *FakeResource) parseTime(timeStr string) time.Time {
  timeParseFormat := "20060102T150405Z"
  t, _ := time.Parse(timeParseFormat, timeStr)
  return t
}

func panicerr(err error) {
  if err != nil {
    panic(err)
  }
}
