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

type ResourceAdapter interface {
	IsCollection() bool
	CalculateEtag() string
	GetContent() string
	GetContentSize() int64
	GetModTime() time.Time
}

type ResourceRecurrence struct {
	StartTime time.Time
	EndTime   time.Time
}

type Resource struct {
	Name string
	Path string

	pathSplit []string
	adapter   ResourceAdapter

	emptyTime time.Time
}

func NewResource(resPath string, adp ResourceAdapter) Resource {
	pClean := lib.ToSlashPath(resPath)
	pSplit := strings.Split(strings.Trim(pClean, "/"), "/")

	return Resource{
		Name:      pSplit[len(pSplit)-1],
		Path:      pClean,
		pathSplit: pSplit,
		adapter:   adp,
	}
}

func (r *Resource) IsCollection() bool {
	return r.adapter.IsCollection()
}

func (r *Resource) IsPrincipal() bool {
	return len(r.pathSplit) <= 1
}

func (r *Resource) ComponentName() string {
	if r.IsCollection() {
		return lib.VCALENDAR
	} else {
		return lib.VEVENT
	}
}

func (r *Resource) StartTimeUTC() time.Time {
	vevent := r.icalVEVENT()
	dtstart := vevent.PropDate(ical.DTSTART, r.emptyTime)

	if dtstart == r.emptyTime {
		log.Printf("WARNING: The property DTSTART was not found in the resource's ical data.\nResource path: %s", r.Path)
		return r.emptyTime
	}

	return dtstart.UTC()
}

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

func (r *Resource) HasProperty(propPath ...string) bool {
	return r.GetPropertyValue(propPath...) != ""
}

func (r *Resource) GetPropertyValue(propPath ...string) string {
	if propPath[0] == ical.VCALENDAR {
		propPath = propPath[1:]
	}

	prop, _ := r.icalendar().DigProperty(propPath...)
	return prop
}

func (r *Resource) HasPropertyParam(paramPath ...string) bool {
	return r.GetPropertyParamValue(paramPath...) != ""
}

func (r *Resource) GetPropertyParamValue(paramPath ...string) string {
	if paramPath[0] == ical.VCALENDAR {
		paramPath = paramPath[1:]
	}

	param, _ := r.icalendar().DigParameter(paramPath...)
	return param
}

func (r *Resource) GetEtag() (string, bool) {
	if r.IsCollection() {
		return "", false
	}

	return r.adapter.CalculateEtag(), true
}

func (r *Resource) GetContentType() (string, bool) {
	if r.IsCollection() {
		return "text/calendar", true
	} else {
		return "text/calendar; component=vcalendar", true
	}
}

func (r *Resource) GetDisplayName() (string, bool) {
	return r.Name, true
}

func (r *Resource) GetContentData() (string, bool) {
	data := r.adapter.GetContent()
	found := data != ""

	return data, found
}

func (r *Resource) GetContentLength() (string, bool) {
	// If its collection, it does not have any content, so mark it as not found
	if r.IsCollection() {
		return "", false
	}

	contentSize := r.adapter.GetContentSize()
	return strconv.FormatInt(contentSize, 10), true
}

func (r *Resource) GetLastModified(format string) (string, bool) {
	return r.adapter.GetModTime().Format(format), true
}

func (r *Resource) GetOwner() (string, bool) {
	var owner string
	if len(r.pathSplit) > 1 {
		owner = r.pathSplit[0]
	} else {
		owner = ""
	}

	return owner, true
}

func (r *Resource) GetOwnerPath() (string, bool) {
	owner, _ := r.GetOwner()

	if owner != "" {
		return fmt.Sprintf("/%s/", owner), true
	} else {
		return "", false
	}
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

    log.Println("UNTIL ", until)
    c := 0
    stmp := start

    for  c < count {
        c += 1
        stmp = rule.GetNext(stmp)
        if (!stmp.Before(until)) {
            break
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

type FileResourceAdapter struct {
	finfo        os.FileInfo
	resourcePath string
}

func (adp *FileResourceAdapter) IsCollection() bool {
	return adp.finfo.IsDir()
}

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

func (adp *FileResourceAdapter) GetContentSize() int64 {
	return adp.finfo.Size()
}

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
    var rex = regexp.MustCompile("(\\w+)=(\\w+)")
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

func (r *RecurrenceRule) GetNext(start time.Time) time.Time {
    interval := r.getIntParam("INTERVAL", 1)
    var inc time.Duration

    freq := r.getParam("FREQ", "")
    switch freq {
        case "SECONDLY":
            inc,_ = time.ParseDuration("1s")
            inc = time.Duration(int64(inc)*int64(interval))
            return start.Add(inc)
        case "MINUTELY":
            inc,_ = time.ParseDuration("1m")
            inc = time.Duration(int64(inc)*int64(interval))
            return start.Add(inc)
        case "HOURLY":
            inc,_ = time.ParseDuration("1h")
            inc = time.Duration(int64(inc)*int64(interval))
            return start.Add(inc)
        case "DAILY":
            inc,_ = time.ParseDuration("24h")
            inc = time.Duration(int64(inc)*int64(interval))
            return start.Add(inc)
        case "WEEKLY":
            inc,_ = time.ParseDuration("168h")
            inc = time.Duration(int64(inc)*int64(interval))
            return start.Add(inc)
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
            return time.Date(year, time.Month(month+1), day, hour, minute, second, nanosecond, time.UTC)
        case "YEARLY":
            year:=start.Year()
            year = year + interval

            month:=start.Month()
            day:=start.Day()
            hour:=start.Hour()
            minute:=start.Minute()
            second:=start.Second()
            nanosecond:=start.Nanosecond()
            return time.Date(year, month, day, hour, minute, second, nanosecond, time.UTC)
        default:
            inc,_ = time.ParseDuration("24h")
            inc = time.Duration(int64(inc)*int64(interval))
            return start.Add(inc)
    }

}
