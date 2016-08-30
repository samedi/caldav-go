package data

import (
  "os"
  "errors"
)

var (
  ErrResourceNotFound = errors.New("caldav: resource not found")
)

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
