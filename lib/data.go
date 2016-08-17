package data

import (
  "fmt"
  "os"
  "path"
  "errors"
  "strings"
  "strconv"
)

var (
  ErrResourceNotFound = errors.New("caldav: resource not found")
)

type CalUser struct {
  Name string
}

type Resource struct {
  Name string
  Path string
  PathSplit []string
  File os.FileInfo
  Owner string
  User *CalUser
}

func NewResource(filePath string, fileInfo os.FileInfo) Resource {
  pClean  := path.Clean(filePath)
  pSplit  := strings.Split(strings.Trim(pClean, "/"), "/")

  // owner is the parent collection
  var owner string
  if len(pSplit) > 1 {
    owner = pSplit[0]
  } else {
    owner = ""
  }

  return Resource{
    Name: fileInfo.Name(),
    Path: pClean,
    PathSplit: pSplit,
    File: fileInfo,
    Owner: owner,
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
    return "", false
  }

  mimetype := "text/calendar"
  return fmt.Sprintf("%s; component=%s", mimetype, strings.ToLower(r.Name)), true
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

func (r *Resource) GetOwnerPath() (string, bool) {
  if r.Owner != "" {
    return fmt.Sprintf("/%s/", r.Owner), true
  } else {
    return "", false
  }
}

type Storage interface {
  GetResources(path string, depth int) ([]Resource, error)
}

type FileStorage struct {
}

func (fs *FileStorage) GetResources(path string, depth int, user *CalUser) ([]Resource, error) {
  result := []Resource{}

  // tries to open the file by the given path
  pwd, _ := os.Getwd()
  f, e := os.Open(pwd + path)
  if e != nil {
    if os.IsNotExist(e) {
			return nil, ErrResourceNotFound
		}
		return nil, e
  }

  // add it as a resource to the result list
  finfo, _ := f.Stat()
  resource := NewResource(path, finfo)
  resource.User = user
  result = append(result, resource)

  // if depth is 1 and the file is a dir, add its children to the result list
  if depth == 1 && finfo.IsDir() {
    files, _ := f.Readdir(0)
    for _, finfo := range files {
      resource = NewResource(path + finfo.Name(), finfo)
      resource.User = user
      result = append(result, resource)
    }
  }

  return result, nil
}
