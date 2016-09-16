package data

import (
  "fmt"
  "os"
  "strings"
  "strconv"
  "time"
  "io/ioutil"

  "caldav/lib"
  "caldav/files"
)

type ResourceInterface interface {
  ComponentName() string
  StartTimeUTC() time.Time
  EndTimeUTC() time.Time
  Recurrences() []ResourceRecurrence
  HasProperty(propName string) bool
  GetPropertyValue(propName string) string
  HasPropertyParam(propName, paramName string) bool
  GetPropertyParamValue(propName, paramName string) string
}

type ResourceAdapter interface {
  IsCollection() bool
  CalculateEtag() string
  GetContent() string
  GetContentSize() int64
  GetCollectionChildPaths() []string
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
  adapter ResourceAdapter
}

func NewResource(resPath string, adp ResourceAdapter) Resource {
  pClean  := lib.ToSlashPath(resPath)
  pSplit  := strings.Split(strings.Trim(pClean, "/"), "/")

  return Resource{
    Name: pSplit[len(pSplit) - 1],
    Path: pClean,
    pathSplit: pSplit,
    adapter: adp,
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
  // TODO: implement based on the vevent component table - section 9.9
  return time.Time{}
}

func (r *Resource) EndTimeUTC() time.Time {
  // TODO: implement based on the vevent component table - section 9.9
  return time.Time{}
}

func (r *Resource) Recurrences() []ResourceRecurrence {
  // TODO: implement
  return nil
}

func (r *Resource) HasProperty(propName string) bool {
  // TODO: implement
  return false
}

func (r *Resource) GetPropertyValue(propName string) string {
  // TODO: implement
  return ""
}

func (r *Resource) HasPropertyParam(propName, paramName string) bool {
  // TODO: implement
  return false
}

func (r *Resource) GetPropertyParamValue(propName, paramName string) string {
  // TODO: implement
  return ""
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
  if r.IsCollection() {
    return "", false
  }

  data := r.adapter.GetContent()
  return data, true
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

func (r *Resource) GetCollectionChildPaths() ([]string, bool) {
  if !r.IsCollection() {
    return nil, false
  }

  paths := r.adapter.GetCollectionChildPaths()

  if paths == nil {
    return nil, false
  }

  return paths, true
}

type FileResourceAdapter struct {
  finfo        os.FileInfo
  resourcePath string
}

func (adp *FileResourceAdapter) IsCollection() bool {
  return adp.finfo.IsDir()
}

func (adp *FileResourceAdapter) GetContent() string {
  data, err := ioutil.ReadFile(files.AbsPath(adp.resourcePath))
  if err != nil {
    // TODO: Log error
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

func (adp *FileResourceAdapter) GetCollectionChildPaths() []string {
  content, err := ioutil.ReadDir(files.AbsPath(adp.resourcePath))
	if err != nil {
    // TODO: Log error
    return nil
	}

  result := []string{}
	for _, file := range content {
    fpath := files.JoinPaths(adp.resourcePath, file.Name())
    result = append(result, fpath)
	}

  return result
}
