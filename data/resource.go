package data

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

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
	// TODO: Implement. This server does not support ical recurrences yet. We just return an empty array.
	return []ResourceRecurrence{}
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
