package data

import (
  "os"
  "errors"
  "caldav/files"
)

var (
  ErrResourceNotFound = errors.New("caldav: resource not found")
  ErrResourceAlreadyCreated = errors.New("caldav: resource already exists")
)

type Storage interface {
  GetResources(rpath string, depth int) ([]Resource, error)
  GetResource(rpath string) (Resource, error)
  IsResourcePresent(rpath string) bool
  CreateResource(content string) (Resource, error)
  UpdateResource(rpath string, content string) (Resource, error)
}

type FileStorage struct {
}

func (fs *FileStorage) GetResources(rpath string, depth int) ([]Resource, error) {
  result := []Resource{}

  // tries to open the file by the given path
  f, e := fs.openResourceFile(rpath, os.O_RDONLY)
  if e != nil {
		return nil, e
  }

  // add it as a resource to the result list
  finfo, _ := f.Stat()
  resource := NewResource(rpath, finfo)
  result = append(result, resource)

  // if depth is 1 and the file is a dir, add its children to the result list
  if depth == 1 && finfo.IsDir() {
    dirFiles, _ := f.Readdir(0)
    for _, finfo := range dirFiles {
      resource = NewResource(files.JoinPaths(rpath, finfo.Name()), finfo)
      result = append(result, resource)
    }
  }

  return result, nil
}

func (fs *FileStorage) GetResource(rpath string) (*Resource, bool, error) {
  resources, err := fs.GetResources(rpath, 0)

  if err != nil {
    return nil, false, err
  }

  if resources == nil || len(resources) == 0 {
    return nil, false, ErrResourceNotFound
  }

  res := resources[0]
  return &res, true, nil
}

func (fs *FileStorage) IsResourcePresent(rpath string) bool {
  _, found, _ := fs.GetResource(rpath)

  return found
}

func (fs *FileStorage) CreateResource(rpath string, content string) (*Resource, error) {
  rAbsPath := files.AbsPath(rpath)

  if fs.IsResourcePresent(rAbsPath) {
    return nil, ErrResourceAlreadyCreated
  }

  // create parent directories (if needed)
  if err := os.MkdirAll(files.DirPath(rAbsPath), os.ModePerm); err != nil {
    return nil, err
  }

  // create file/resource and write content
  f, err := os.Create(rAbsPath)
  if err != nil {
    return nil, err
  }
  f.WriteString(content)

  finfo, _ := f.Stat()
  res := NewResource(rpath, finfo)
  return &res, nil
}

func (fs *FileStorage) UpdateResource(rpath string, content string) (*Resource, error) {
  f, e := fs.openResourceFile(rpath, os.O_RDWR)
  if e != nil {
		return nil, e
  }

  // update content
  f.Truncate(0)
  f.WriteString(content)

  finfo, _ := f.Stat()
  res := NewResource(rpath, finfo)
  return &res, nil
}

func (fs *FileStorage) DeleteResource(rpath string) error {
  err := os.Remove(files.AbsPath(rpath))

  return err
}

func (fs *FileStorage) openResourceFile(filepath string, mode int) (*os.File, error) {
  f, e := os.OpenFile(files.AbsPath(filepath), mode, 0666)
  if e != nil {
    if os.IsNotExist(e) {
			return nil, ErrResourceNotFound
		}
		return nil, e
  }

  return f, nil
}
