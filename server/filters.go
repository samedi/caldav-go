package server

import (
  "fmt"
  "time"
  "strings"
  "errors"
  "github.com/beevik/etree"

  "caldav/lib"
  "caldav/data"
)

// ================ FILTERS ==================
// Filters are a set of rules used to retrieve a range of resources. It is used primarily
// on REPORT requests and is described in details here (RFC4791#7.8).

const (
  TAG_FILTER = "filter"
  TAG_COMP_FILTER = "comp-filter"
  TAG_PROP_FILTER = "prop-filter"
  TAG_PARAM_FILTER = "param-filter"
  TAG_TIME_RANGE = "time-range"
  TAG_TEXT_MATCH = "text-match"
  TAG_IS_NOT_DEFINED = "is-not-defined"
)

type ResourceFilter struct {
  name       string
  text       string
  attrs      map[string]string
  children   []ResourceFilter // collection of child filters.
  etreeElem  *etree.Element // holds the parsed XML node/tag as an `etree` element.
}

// This function creates a new filter object from a piece of XML string.
func ParseFilterFromXML(xml string) (*ResourceFilter, error) {
  doc := etree.NewDocument()
  if err := doc.ReadFromString(xml); err != nil {
    return nil, err
  }

  // Right now we're searching for a <filter> tag to initialize the filter struct from it.
  // It SHOULD be a valid XML CALDAV:filter tag (RFC4791#9.7). We're not checking namespaces yet.
  // TODO: check for XML namespaces and restrict it to accept only CALDAV:filter tag.
  elem := doc.FindElement("//" + TAG_FILTER)
  if elem == nil {
    return nil, errors.New(fmt.Sprintf("the parsed XML should contain a <%s></%s> element", TAG_FILTER, TAG_FILTER))
  }

  filter := newFilterFromEtreeElem(elem)
  return &filter, nil
}

func newFilterFromEtreeElem(elem *etree.Element) ResourceFilter {
  // init filter from etree element
  filter := ResourceFilter{
    name:      elem.Tag,
    text:      strings.TrimSpace(elem.Text()),
    etreeElem: elem,
    attrs:     make(map[string]string),
  }

  // set attributes
  for _, attr := range elem.Attr {
    filter.attrs[attr.Key] = attr.Value
  }

  return filter
}

func (f *ResourceFilter) Match(target data.ResourceInterface) bool {
  if f.name == TAG_FILTER {
    return f.rootFilterMatch(target)
  }

  return false
}

func (f *ResourceFilter) rootFilterMatch(target data.ResourceInterface) bool {
  if f.isEmpty() {
    return false
  }

  return f.rootChildrenMatch(target)
}

// checks if all the root's child filters match the target resource
func (f *ResourceFilter) rootChildrenMatch(target data.ResourceInterface) bool {
  for _, child := range f.getChildren() {
    // root filters only accept comp filters as children
    if child.name != TAG_COMP_FILTER || !child.compMatch(target) {
      return false
    }
  }

  return true
}

// See RFC4791-9.7.1.
func (f *ResourceFilter) compMatch(target data.ResourceInterface) bool {
  compName := target.ComponentName()

  if f.isEmpty() {
    // Point #1 of RFC4791#9.7.1
    return f.attrs["name"] == compName
  } else if f.contains(TAG_IS_NOT_DEFINED) {
    // Point #2 of RFC4791#9.7.1
    return f.attrs["name"] != compName
  } else {
    // check each child of the current filter if they all match.
    return f.compChildrenMatch(target)
  }
}

// checks if all the comp's child filters match the target resource
func (f *ResourceFilter) compChildrenMatch(target data.ResourceInterface) bool {
  for _, child := range f.getChildren() {
    var match bool

    switch child.name {
    case TAG_TIME_RANGE:
      // Point #3 of RFC4791#9.7.1
      match = child.timeRangeMatch(target)
    case TAG_PROP_FILTER:
      // Point #4 of RFC4791#9.7.1
      match = child.propMatch(target)
    case TAG_COMP_FILTER:
      // Point #4 of RFC4791#9.7.1
      match = child.compMatch(target)
    }

    if !match {
      return false
    }
  }

  return true
}

// See RFC4791-9.9
func (f *ResourceFilter) timeRangeMatch(target data.ResourceInterface) bool {
  startAttr := f.attrs["start"]
  endAttr   := f.attrs["end"]

  // at least one of the two MUST be present
  if startAttr == "" && endAttr == "" {
    // if both of them are missing, return false
    return false
  } else if startAttr == "" {
    // if missing only the `start`, set it open ended to the left
    startAttr = "00010101T000000Z"
  } else if endAttr == "" {
    // if missing only the `end`, set it open ended to the right
    endAttr = "99991231T235959Z"
  }

  // The logic below is only applicable for VEVENT components. So
  // we return false if the resource is not a VEVENT component.
  if target.ComponentName() != lib.VEVENT {
    return false
  }

  // from the RFC, the `start` and `end` attributes MUST be in UTC and in this specific format
  timeParseFormat := "20060102T150405Z"

  rangeStart, err := time.Parse(timeParseFormat, startAttr)
  if err != nil {
    // TODO: Log error
    return false
  }

  rangeEnd, err := time.Parse(timeParseFormat, endAttr)
  if err != nil {
    // TODO: Log error
    return false
  }

  // the following logic is inferred from the rules table for VEVENT components,
  // described in RFC4791-9.9.
  overlapRange := func(dtStart, dtEnd, rangeStart, rangeEnd time.Time) bool {
    if dtStart.Equal(dtEnd) {
      // Lines 3 and 4 of the table deal when the DTSTART and DTEND dates are equals.
      // In this case we use the rule: (start <= DTSTART && end > DTSTART)
      return (rangeStart.Before(dtStart) || rangeStart.Equal(dtStart)) && rangeEnd.After(dtStart)
    } else {
      // Lines 1, 2 and 6 of the table deal when the DTSTART and DTEND dates are different.
      // In this case we use the rule: (start < DTEND && end > DTSTART)
      return rangeStart.Before(dtEnd) && rangeEnd.After(dtStart)
    }
  }

  // first we check each of the target recurrences (if any).
  for _, recurrence := range target.Recurrences() {
    // if any of them overlap the filter range, we return true right away
    if overlapRange(recurrence.StartTime, recurrence.EndTime, rangeStart, rangeEnd) {
      return true
    }
  }

  // if none of the recurrences match, we just return if the actual
  // resource's `start` and `end` times match the filter range
  return overlapRange(target.StartTimeUTC(), target.EndTimeUTC(), rangeStart, rangeEnd)
}

// See RFC4791-9.7.2.
func (f *ResourceFilter) propMatch(target data.ResourceInterface) bool {
  propName := f.attrs["name"]

  if f.isEmpty() {
    // Point #1 of RFC4791#9.7.2
    return target.HasProperty(propName)
  } else if f.contains(TAG_IS_NOT_DEFINED) {
    // Point #2 of RFC4791#9.7.2
    return !target.HasProperty(propName)
  } else {
    // check each child of the current filter if they all match.
    return f.propChildrenMatch(target, propName)
  }

  return false
}

// checks if all the prop's child filters match the target resource
func (f *ResourceFilter) propChildrenMatch(target data.ResourceInterface, propName string) bool {
  for _, child := range f.getChildren() {
    var match bool

    switch child.name {
    case TAG_TIME_RANGE:
      // Point #3 of RFC4791#9.7.2
      // TODO: this point is not very clear on how to match time range against properties.
      // So we're returning `false` in the meantime.
      match = false
    case TAG_TEXT_MATCH:
      // Point #4 of RFC4791#9.7.2
      propText := target.GetPropertyValue(propName)
      match = child.textMatch(propText)
    case TAG_PARAM_FILTER:
      // Point #4 of RFC4791#9.7.2
      match = child.paramMatch(target, propName)
    }

    if !match {
      return false
    }
  }

  return true
}

// See RFC4791-9.7.3
func (f *ResourceFilter) paramMatch(target data.ResourceInterface, parentProp string) bool {
  paramName := f.attrs["name"]

  if f.isEmpty() {
    // Point #1 of RFC4791#9.7.3
    return target.HasPropertyParam(parentProp, paramName)
  } else if f.contains(TAG_IS_NOT_DEFINED) {
    // Point #2 of RFC4791#9.7.3
    return !target.HasPropertyParam(parentProp, paramName)
  } else {
    child := f.getChildren()[0]

    // param filters can also have (only-one) nested text-match filter
    if child.name == TAG_TEXT_MATCH {
      paramValue := target.GetPropertyParamValue(parentProp, paramName)
      return child.textMatch(paramValue)
    }
  }

  return false
}

// See RFC4791-9.7.5
func (f *ResourceFilter) textMatch(targetText string) bool {
  // TODO: collations are not being considered/supported yet.
  // Texts are lowered to be case-insensitive, almost as the "i;ascii-casemap" value.

  targetText = strings.ToLower(targetText)
  expectedSubstr := strings.ToLower(f.text)

  match := strings.Contains(targetText, expectedSubstr)

  if f.attrs["negate-condition"] == "yes" {
    return !match
  }

  return match
}

func (f *ResourceFilter) isEmpty() bool {
  return len(f.getChildren()) == 0 && f.text == ""
}

func (f *ResourceFilter) contains(filterName string) bool {
  for _, child := range f.getChildren() {
    if child.name == filterName {
      return true
    }
  }

  return false
}

// lazy evaluation of the child filters
func (f *ResourceFilter) getChildren() []ResourceFilter {
  if f.children == nil {
    f.children = []ResourceFilter{}

    for _, childElem := range f.etreeElem.ChildElements() {
      childFilter := newFilterFromEtreeElem(childElem)
      f.children = append(f.children, childFilter)
    }
  }

  return f.children
}
