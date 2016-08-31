package data

import (
  "fmt"
  "os"
  "path"
  "strings"
  "strconv"
  "io/ioutil"
)

type Resource struct {
  Name string
  Path string
  PathSplit []string
  File os.FileInfo
  User *CalUser
}

func NewResource(filePath string, fileInfo os.FileInfo) Resource {
  pClean  := path.Clean(filePath)
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

  pwd, _ := os.Getwd()
  data, err := ioutil.ReadFile(pwd + r.Path)
  if err != nil {
    // TODO: Log error
    return "", false
  }

  return string(data), true
}
