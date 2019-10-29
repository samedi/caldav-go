package data

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
    "regexp"

	"github.com/laurent22/ical-go"

	"github.com/samedi/caldav-go/files"
	"github.com/samedi/caldav-go/lib"
)

// ResourceInterface defines the main interface of a CalDAV resource object. This
// interface exists only to define the common resource operation and should not be custom-implemented.
// The default and canonical implementation is provided by `data.Resource`, convering all the commonalities.
// Any specifics in implementations should be handled by the `data.ResourceAdapter`.
type ResourceInterface interface {
	ComponentName() string
	StartTimeUTC() time.Time
	EndTimeUTC() time.Time
	Recurrences() []ResourceRecurrence
	HasProperty(propPath ...string) bool
	GetPropertyValue(propPath ...string) string
	HasPropertyParam(paramName ...string) bool
	GetPropertyParamValue(paramName ...string) string
}

// ResourceAdapter serves as the object to abstract all the specicities in different resources implementations.
// For example, the way to tell whether a resource is a collection or how to read its content differentiates
// on resources stored in the file system, coming from a relational DB or from the cloud as JSON. These differentiations
// should be covered by providing a specific implementation of the `ResourceAdapter` interface. So, depending on the current
// resource storage strategy, a matching resource adapter implementation should be provided whenever a new resource is initialized.
type ResourceAdapter interface {
	IsCollection() bool
	CalculateEtag() string
	GetContent() string
	GetContentSize() int64
	GetModTime() time.Time
}

// ResourceRecurrence represents a recurrence for a resource.
// NOTE: recurrences are not supported yet.
type ResourceRecurrence struct {
	StartTime time.Time
	EndTime   time.Time
}

// Resource represents the CalDAV resource. Basically, it has a name it's accessible based on path.
// A resource can be a collection, meaning it doesn't have any data content, but it has child resources.
// A non-collection is the actual resource which has the data in iCal format and which will feed the calendar.
// When visualizing the whole resources set in a tree representation, the collection resource would be the inner nodes and
// the non-collection would be the leaves.
type Resource struct {
	Name string
	Path string

	pathSplit []string
	adapter   ResourceAdapter

	emptyTime time.Time
}

// NewResource initializes a new `Resource` instance based on its path and the `ResourceAdapter` implementation to be used.
func NewResource(rawPath string, adp ResourceAdapter) Resource {
	pClean := lib.ToSlashPath(rawPath)
	pSplit := strings.Split(strings.Trim(pClean, "/"), "/")
    p := pClean
    if (adp.IsCollection()) {
        p+="/"
    }
	return Resource{
		Name:      pSplit[len(pSplit)-1],
		Path:      p,
		pathSplit: pSplit,
		adapter:   adp,
	}
}

// IsCollection tells whether a resource is a collection or not.
func (r *Resource) IsCollection() bool {
	return r.adapter.IsCollection()
}

// IsPrincipal tells whether a resource is the principal resource or not.
// A principal resource means it's a root resource.
func (r *Resource) IsPrincipal() bool {
	return len(r.pathSplit) <= 1
}

// ComponentName returns the type of the resource. VCALENDAR for collection resources, VEVENT otherwise.
func (r *Resource) ComponentName() string {
	if r.IsCollection() {
		return lib.VCALENDAR
	}

	return lib.VEVENT
}

// StartTimeUTC returns the start time in UTC of a VEVENT resource.
func (r *Resource) StartTimeUTC() time.Time {
	vevent := r.icalVEVENT()
	dtstart := vevent.PropDate(ical.DTSTART, r.emptyTime)

	if dtstart == r.emptyTime {
		log.Printf("WARNING: The property DTSTART was not found in the resource's ical data.\nResource path: %s", r.Path)
		return r.emptyTime
	}

	return dtstart.UTC()
}

// EndTimeUTC returns the end time in UTC of a VEVENT resource.
func (r *Resource) EndTimeUTC() time.Time {
	vevent := r.icalVEVENT()
	dtend := vevent.PropDate(ical.DTEND, r.emptyTime)

	// when the DTEND property is not present, we just add the DURATION (if any) to the DTSTART
	if dtend == r.emptyTime {
		duration := vevent.PropDuration(ical.DURATION)
		dtend = r.StartTimeUTC().Add(duration)
	}

	return dtend.UTC()
}

// Recurrences returns an array of resource recurrences.
// NOTE: Recurrences are not supported yet. An empty array will always be returned.
func (r *Resource) Recurrences() []ResourceRecurrence {
    vevent := r.icalVEVENT()
    rrule := vevent.PropString("RRULE", "")

    if ( rrule != "" ) {
        log.Printf("RECURRENCE : %s, Path: %s", rrule, r.Path )
        start := r.StartTimeUTC()
        end := r.EndTimeUTC()
        duration := end.Sub(start)
        result := r.calcRecurrences( start, duration, rrule )
        return result

    } else  {
	    return []ResourceRecurrence{}
    }
}

// HasProperty tells whether the resource has the provided property in its iCal content.
// The path to the property should be provided in case of nested properties.
// Example, suppose the resource has this content:
//
// 	BEGIN:VCALENDAR
// 	BEGIN:VEVENT
// 	DTSTART:20160914T170000
// 	END:VEVENT
// 	END:VCALENDAR
//
// HasProperty("VEVENT", "DTSTART") => returns true
// HasProperty("VEVENT", "DTEND") => returns false
func (r *Resource) HasProperty(propPath ...string) bool {
	return r.GetPropertyValue(propPath...) != ""
}

// GetPropertyValue gets a property value from the resource's iCal content.
// The path to the property should be provided in case of nested properties.
// Example, suppose the resource has this content:
//
// 	BEGIN:VCALENDAR
// 	BEGIN:VEVENT
// 	DTSTART:20160914T170000
// 	END:VEVENT
// 	END:VCALENDAR
//
// GetPropertyValue("VEVENT", "DTSTART") => returns "20160914T170000"
// GetPropertyValue("VEVENT", "DTEND") => returns ""
func (r *Resource) GetPropertyValue(propPath ...string) string {
	if propPath[0] == ical.VCALENDAR {
		propPath = propPath[1:]
	}

	prop, _ := r.icalendar().DigProperty(propPath...)
	return prop
}

// HasPropertyParam tells whether the resource has the provided property param in its iCal content.
// The path to the param should be provided in case of nested params.
// Example, suppose the resource has this content:
//
// 	BEGIN:VCALENDAR
// 	BEGIN:VEVENT
// 	ATTENDEE;PARTSTAT=NEEDS-ACTION:FOO
// 	END:VEVENT
// 	END:VCALENDAR
//
// HasPropertyParam("VEVENT", "ATTENDEE", "PARTSTAT") => returns true
// HasPropertyParam("VEVENT", "ATTENDEE", "OTHER") => returns false
func (r *Resource) HasPropertyParam(paramPath ...string) bool {
	return r.GetPropertyParamValue(paramPath...) != ""
}

// GetPropertyParamValue gets a property param value from the resource's iCal content.
// The path to the param should be provided in case of nested params.
// Example, suppose the resource has this content:
//
// 	BEGIN:VCALENDAR
// 	BEGIN:VEVENT
// 	ATTENDEE;PARTSTAT=NEEDS-ACTION:FOO
// 	END:VEVENT
// 	END:VCALENDAR
//
// GetPropertyParamValue("VEVENT", "ATTENDEE", "PARTSTAT") => returns "NEEDS-ACTION"
// GetPropertyParamValue("VEVENT", "ATTENDEE", "OTHER") => returns ""
func (r *Resource) GetPropertyParamValue(paramPath ...string) string {
	if paramPath[0] == ical.VCALENDAR {
		paramPath = paramPath[1:]
	}

	param, _ := r.icalendar().DigParameter(paramPath...)
	return param
}

// GetEtag returns the ETag of the resource and a flag saying if the ETag is present.
// For collection resource, it returns an empty string and false.
func (r *Resource) GetEtag() (string, bool) {
	if r.IsCollection() {
		return "", false
	}

	return r.adapter.CalculateEtag(), true
}

// GetContentType returns the type of the content of the resource.
// Collection resources are "text/calendar". Non-collection resources are "text/calendar; component=vcalendar".
func (r *Resource) GetContentType() (string, bool) {
	if r.IsCollection() {
		return "text/calendar", true
	}

	return "text/calendar; component=vcalendar", true
}

// GetDisplayName returns the name/identifier of the resource.
func (r *Resource) GetDisplayName() (string, bool) {
	return r.Name, true
}

// GetContentData reads and returns the raw content of the resource as string and flag saying if the content was found.
// If the resource does not have content (like collection resource), it returns an empty string and false.
func (r *Resource) GetContentData() (string, bool) {
	data := r.adapter.GetContent()
	found := data != ""

	return data, found
}

// GetContentLength returns the length of the resource's content and flag saying if the length is present.
// If the resource does not have content (like collection resource), it returns an empty string and false.
func (r *Resource) GetContentLength() (string, bool) {
	// If its collection, it does not have any content, so mark it as not found
	if r.IsCollection() {
		return "", false
	}

	contentSize := r.adapter.GetContentSize()
	return strconv.FormatInt(contentSize, 10), true
}

// GetLastModified returns the last time the resource was modified. The returned time
// is returned formatted in the provided `format`.
func (r *Resource) GetLastModified(format string) (string, bool) {
	return r.adapter.GetModTime().Format(format), true
}

// GetOwner returns the owner of the resource. This is usually the principal resource associated (the root resource).
// If the resource does not have a owner (for example it's a principal resource alread), it returns an empty string.
func (r *Resource) GetOwner() (string, bool) {
	var owner string
	if len(r.pathSplit) > 1 {
		owner = r.pathSplit[0]
	} else {
		owner = ""
	}

	return owner, true
}

// GetOwnerPath returns the path to this resource's owner, or an empty string when the resource does not have any owner.
func (r *Resource) GetOwnerPath() (string, bool) {
	owner, _ := r.GetOwner()

	if owner != "" {
		return fmt.Sprintf("/%s/", owner), true
	}

	return "", false
}

// TODO: memoize
func (r *Resource) icalVEVENT() *ical.Node {
	vevent := r.icalendar().ChildByName(ical.VEVENT)

	// if nil, log it and return an empty vevent
	if vevent == nil {
		log.Printf("WARNING: The resource's ical data is missing the VEVENT property.\nResource path: %s", r.Path)

		return &ical.Node{
			Name: ical.VEVENT,
		}
	}

	return vevent
}

// TODO: memoize
func (r *Resource) icalendar() *ical.Node {
	data, found := r.GetContentData()

	if !found {
		log.Printf("WARNING: The resource's ical data does not have any data.\nResource path: %s", r.Path)
		return &ical.Node{
			Name: ical.VCALENDAR,
		}
	}

	icalNode, err := ical.ParseCalendar(data)
	if err != nil {
		log.Printf("ERROR: Could not parse the resource's ical data.\nError: %s.\nResource path: %s", err, r.Path)
		return &ical.Node{
			Name: ical.VCALENDAR,
		}
	}

	return icalNode
}

func (r *Resource) calcRecurrences( start time.Time, duration time.Duration, rrule string) ([]ResourceRecurrence) {
    result := []ResourceRecurrence{}
    rule := NewRecurrenceRule(rrule)

    count := rule.getIntParam("COUNT", 1000)
    until := rule.getTimeParam("UNTIL", time.Date(9999,12,31,23,59,59,00,time.UTC))

    //log.Println("UNTIL ", until)
    c := 0
    stmp := start
    var skip bool
    for  c < count {
        c += 1
        stmp, skip = rule.GetNext(stmp)
        if (!stmp.Before(until)) {
            break
        }
        if (skip) {
            continue
        }
        recurrence := ResourceRecurrence{ stmp, stmp.Add(duration) }
        result = append(result, recurrence)
    }

    // TODO Parse rrule
    // start:
    // FREQ (SECONDLY, MINUTELY, HOURLY, DAILY, WEEKLY, MONTHLY, YEARLY)
    // INTERVAL (done)
    // These can either be filters or set a value directly - if the freq is smaller than the unit it filters otherwise sets to a fixed value
    // cond: 
    // BYSECOND
    // BYMINUTE
    // BYHOUR
    // BYDAY
    // BYMONTHDAY
    // BYYEARDAY
    // BYWEEKNO
    // BYMONTH
    // BYSETPOS
    // WKST

    // end
    // COUNT (done)
    // UNTIL (done)

    // TODO add rdate 
    // TODO remove exdate
    return result;
}

// FileResourceAdapter implements the `ResourceAdapter` for resources stored as files in the file system.
type FileResourceAdapter struct {
	finfo        os.FileInfo
	resourcePath string
}

// IsCollection tells whether the file resource is a directory or not.
func (adp *FileResourceAdapter) IsCollection() bool {
	return adp.finfo.IsDir()
}

// GetContent reads the file content and returns it as string. For collection resources (directories), it
// returns an empty string.
func (adp *FileResourceAdapter) GetContent() string {
	if adp.IsCollection() {
		return ""
	}

	data, err := ioutil.ReadFile(files.AbsPath(adp.resourcePath))
	if err != nil {
		log.Printf("ERROR: Could not read file content for the resource.\nError: %s.\nResource path: %s.", err, adp.resourcePath)
		return ""
	}

	return string(data)
}

// GetContentSize returns the content length.
func (adp *FileResourceAdapter) GetContentSize() int64 {
	return adp.finfo.Size()
}

// CalculateEtag calculates an ETag based on the file current modification status and returns it.
func (adp *FileResourceAdapter) CalculateEtag() string {
	// returns ETag as the concatenated hex values of a file's
	// modification time and size. This is not a reliable synchronization
	// mechanism for directories, so for collections we return empty.
	if adp.IsCollection() {
		return ""
	}

	fi := adp.finfo
	return fmt.Sprintf(`"%x%x"`, fi.ModTime().UnixNano(), fi.Size())
}

// GetModTime returns the time when the file was last modified.
func (adp *FileResourceAdapter) GetModTime() time.Time {
	return adp.finfo.ModTime()
}


type RecurrenceRuleInterface interface {
    GetIntParam(name string, defaultValue int) int
    GetStringParam(name string, defaultValue string) string
    GetTimeParam(name string, defaultValue time.Time) time.Time
}

type RecurrenceRule struct {
    rrule string
    params map[string]string
}

func NewRecurrenceRule(rrule string) RecurrenceRule {
    var rex = regexp.MustCompile("(\\w+)=([a-zA-Z0-9-]+)")
    data := rex.FindAllStringSubmatch(rrule, -1)

    p := make(map[string]string)
    for _, kv := range data {
        k := kv[1]
        v := kv[2]
        p[k] = v
    }

    return RecurrenceRule{
        rrule: rrule,
        params: p,
    }
}

func (r *RecurrenceRule) getIntParam(name string, defaultValue int) int {
    v := defaultValue
    if val, ok := r.params[name]; ok {
        tmp, err := strconv.Atoi(val)
        if err == nil {
            v = tmp
        }
    }
    return v
}

func( r *RecurrenceRule) hasParam(name string) bool {
    if _, ok := r.params[name]; ok {
        return true;
    }
    return false
}

func (r *RecurrenceRule) getParam(name string, defaultValue string) string {
    v := defaultValue
    if val, ok := r.params[name]; ok {
        v = val
    }
    return v
}

func (r *RecurrenceRule) getTimeParam(name string, defaultValue time.Time) time.Time {
    v := defaultValue
    if tmp,ok := r.params[name]; ok {
        if d,ok2 := time.Parse("20060102T150405Z", tmp);ok2==nil {
            v = d
        }
    }
    return v
}

func (r *RecurrenceRule) GetNext(start time.Time) (time.Time, bool) {
    interval := r.getIntParam("INTERVAL", 1)
    var inc time.Duration

    freq := r.getParam("FREQ", "")
    var res time.Time

    switch freq {
        case "SECONDLY":
            inc,_ = time.ParseDuration("1s")
            inc = time.Duration(int64(inc)*int64(interval))
            res = start.Add(inc)
        case "MINUTELY":
            inc,_ = time.ParseDuration("1m")
            inc = time.Duration(int64(inc)*int64(interval))
            res = start.Add(inc)
        case "HOURLY":
            inc,_ = time.ParseDuration("1h")
            inc = time.Duration(int64(inc)*int64(interval))
            res = start.Add(inc)
        case "DAILY":
            inc,_ = time.ParseDuration("24h")
            inc = time.Duration(int64(inc)*int64(interval))
            res = start.Add(inc)
        case "WEEKLY":
            inc,_ = time.ParseDuration("168h")
            inc = time.Duration(int64(inc)*int64(interval))
            res = start.Add(inc)
        case "MONTHLY":
            year:=start.Year()
            month:=int(start.Month())-1
            if  month + interval >= 12 {
                year = year + 1
            }
            month = (month + interval) % 12
            day:=start.Day()
            hour:=start.Hour()
            minute:=start.Minute()
            second:=start.Second()
            nanosecond:=start.Nanosecond()
            res = time.Date(year, time.Month(month+1), day, hour, minute, second, nanosecond, time.UTC)
        case "YEARLY":
            year:=start.Year()
            year = year + interval

            month:=start.Month()
            day:=start.Day()
            hour:=start.Hour()
            minute:=start.Minute()
            second:=start.Second()
            nanosecond:=start.Nanosecond()
            res = time.Date(year, month, day, hour, minute, second, nanosecond, time.UTC)
        default:
            inc,_ = time.ParseDuration("24h")
            inc = time.Duration(int64(inc)*int64(interval))
            res = start.Add(inc)
    }
    return r.replaceBy(res), r.skipBy(res)
}

func (r *RecurrenceRule) replaceBy(start time.Time) time.Time {
    // TODO WKST
    // TODO LISTS of by values
    // TODO BYEASTER
    // TODO BYWEEKNO
    fint := r.freqToInt(r.getParam("FREQ", ""))
    t := start
    if (fint == 4 && r.hasParam("BYDAY")) { // Weekly
        w1 := int(t.Weekday())
        w2 := r.parseWeekday(r.getParam("BYDAY", ""))
        wdiff := w2-w1
        if wdiff < 0 {
            wdiff += 7
        }
        inc,_ := time.ParseDuration("24h")
        inc = time.Duration(int64(inc)*int64(wdiff))
        t = start.Add(inc)
    }

    year:=t.Year()
    month:=int(t.Month())-1
    if (fint > 5 && r.hasParam("BYMONTH")) {
        month = r.getIntParam("BYMONTH", 0)-1
    }
    day:=t.Day()
    if (fint == 6 && r.hasParam("BYYEARDAY")) {
        first := time.Date(year, time.Month(1), 1, 0, 0, 0, 0, time.UTC)
        inc,_ := time.ParseDuration("24h")
        days := r.getIntParam("BYYEARDAY",0)
        inc = time.Duration(int64(inc)*int64(days))
        tmp := first.Add(inc)
        day = tmp.Day()
        month =int(tmp.Month())-1
    }

    if (fint > 4 && r.hasParam("BYMONTHDAY")) {
        day = r.getIntParam("BYMONTHDAY", 0)
    }
    if (fint == 5 && r.hasParam("BYDAY")) { // Monthly
        byday := r.getParam("BYDAY", "")
        d:=0
        pos:= r.getIntParam("BYPOS", 1)
        if (len(byday) <= 2) {
            d = r.parseWeekday(byday)
        } else {
            d = r.parseWeekday(byday[len(byday)-1:])
            tmp,err:=strconv.Atoi(byday[:len(byday)-2])
            if err == nil {
                pos = tmp
            }
        }
        if pos > 0 {
            first := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)

            w1 := int(first.Weekday())
            wdiff := d - w1;
            if wdiff < 0 {
                wdiff+=7
            }
            day = 1+ wdiff + 7*(pos-1)
        } else {
            yofs := 0
            if  month == 11 {
                yofs = 1 // handle december
            }
            first := time.Date(year+yofs, time.Month(month+2), 1, 0, 0, 0, 0, time.UTC)

            w1 := int(first.Weekday())
            wdiff := d - w1;
            if wdiff < 0 {
                wdiff+=7
            }
            wdiff += pos * 7
            inc,_ := time.ParseDuration("24h")
            inc = time.Duration(int64(inc)*int64(wdiff))
            day = first.Add(inc).Day()
        }
    }

    hour:=t.Hour()
    if (fint > 3 && r.hasParam("BYHOUR")) {
        hour = r.getIntParam("BYHOUR", 0)
    }
    minute:=t.Minute()
    if (fint > 2 && r.hasParam("BYMINUTE")) {
        minute = r.getIntParam("BYMINUTE", 0)
    }
    second:=t.Second()
    if (fint > 1 && r.hasParam("BYSECOND")) {
        second = r.getIntParam("BYSECOND", 0)
    }
    nanosecond:=t.Nanosecond()

    t = time.Date(year, time.Month(month+1), day, hour, minute, second, nanosecond, time.UTC)
    return t;
}

func (r *RecurrenceRule) freqToInt(freq string) int {
    switch freq {
        case "SECONDLY":
            return 0
        case "MINUTELY":
            return 1
        case "HOURLY":
            return 2
        case "DAILY":
            return 3
        case "WEEKLY":
            return 4
        case "MONTHLY":
            return 5
        case "YEARLY":
            return 6
        default:
            return 6
    }
}

func (r *RecurrenceRule) parseWeekday(day string) int {
    i, err := strconv.Atoi(day)
    if err == nil {
        return i
    }

    switch day {
        case "SO":
            return 0
        case "MO":
            return 1
        case "TU":
            return 2
        case "WE":
            return 3
        case "TH":
            return 4
        case "FR":
            return 5
        case "SA":
            return 6
        default:
            return 1
    }
}

func (r *RecurrenceRule) skipBy(t time.Time) bool {
    fint := r.freqToInt(r.getParam("FREQ", ""))
    month:=int(t.Month())
    hour:=t.Hour()
    minute:=t.Minute()
    second:=t.Second()

    if (fint <= 5 && r.hasParam("BYMONTH")) {
        return r.getIntParam("BYMONTH",1) != month;
    }
    if (fint <= 3 && r.hasParam("BYDAY")) {
        d1 := int(t.Weekday())
        d2 := r.parseWeekday( r.getParam("BYDAY",""))
        return d1 != d2
    }
    if (fint <= 2 && r.hasParam("BYHOUR")) {
        return r.getIntParam("BYHOUR",0) != hour
    }
    if (fint <= 1 && r.hasParam("BYMINUTE")) {
        return r.getIntParam("BYMINUTE",0) != minute
    }
    if (fint == 0 && r.hasParam("BYSECOND")) {
        return r.getIntParam("BYSECOND",0) != second
    }
    return false
}
