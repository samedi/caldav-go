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

// TODO: rename it to Resource
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

type ResourceRecurrence struct {
  StartTime time.Time
  EndTime   time.Time
}

// TODO: rename it to ResourceFile
type Resource struct {
  Name string
  Path string
  PathSplit []string
  File os.FileInfo
}

func NewResource(filePath string, fileInfo os.FileInfo) Resource {
  pClean  := files.ToSlashPath(filePath)
  pSplit  := strings.Split(strings.Trim(pClean, "/"), "/")

  return Resource{
    Name: fileInfo.Name(),
    Path: pClean,
    PathSplit: pSplit,
    File: fileInfo,
  }
}

func (r *Resource) IsCollection() bool {
  return r.File.IsDir()
}

func (r *Resource) IsPrincipal() bool {
  return len(r.PathSplit) <= 1
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
  // returns ETag as the concatenated hex values of a file's
	// modification time and size. This is not a reliable synchronization
	// mechanism for directories, so for collections we mark it as not found.
  if r.IsCollection() {
    return "", false
  }

  fi := r.File
  return fmt.Sprintf(`"%x%x"`, fi.ModTime().UnixNano(), fi.Size()), true
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

func (r *Resource) GetContentLength() (string, bool) {
  // If its collection, it does not have any content, so mark it as not found
  if r.IsCollection() {
    return "", false
  }

  return strconv.FormatInt(r.File.Size(), 10), true
}

func (r *Resource) GetLastModified(format string) (string, bool) {
	return r.File.ModTime().Format(format), true
}

func (r *Resource) GetOwner() (string, bool) {
  var owner string
  if len(r.PathSplit) > 1 {
    owner = r.PathSplit[0]
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

func (r *Resource) GetData() (string, bool) {
  if r.IsCollection() {
    return "", false
  }

  data, err := ioutil.ReadFile(files.AbsPath(r.Path))
  if err != nil {
    // TODO: Log error
    return "", false
  }

  return string(data), true
}

func (r *Resource) GetCollectionContentPaths() ([]string, bool) {
  if !r.IsCollection() {
    return nil, false
  }

  content, err := ioutil.ReadDir(files.AbsPath(r.Path))
	if err != nil {
    // TODO: Log error
    return nil, false
	}

  result := []string{}
	for _, file := range content {
    fpath := files.JoinPaths(r.Path, file.Name())
    result = append(result, fpath)
	}

  return result, true
}
