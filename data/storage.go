package data

import (
  "os"
  "log"
  "io/ioutil"
  "git.samedi.cc/ferraz/caldav/errs"
  "git.samedi.cc/ferraz/caldav/files"
)

// The storage is the responsible for the CRUD operations on the caldav resources.
type Storage interface {
  GetResources(rpath string, withChildren bool) ([]Resource, error)
  GetResourcesByFilters(rpath string, filters *ResourceFilter) ([]Resource, error)
  /* Fetch a list of resources by path from the storage
  *
  * This method should fetch all the `rpaths` and return an array
  * of the reosurces found. No error 404 will be returned if an
  * element cannot be found.
  *
  * errors can be returned if errors other than "not found" happened. */
  GetResourcesByList(rpaths []string) ([]Resource, error)
  GetResource(rpath string) (*Resource, bool, error)
  IsResourcePresent(rpath string) bool
  CreateResource(rpath, content string) (*Resource, error)
  UpdateResource(rpath, content string) (*Resource, error)
  DeleteResource(rpath string) error
}

type FileStorage struct {
}

func (fs *FileStorage) GetResources(rpath string, withChildren bool) ([]Resource, error) {
  result := []Resource{}

  // tries to open the file by the given path
  f, e := fs.openResourceFile(rpath, os.O_RDONLY)
  if e != nil {
		return nil, e
  }

  // add it as a resource to the result list
  finfo, _ := f.Stat()
  resource := NewResource(rpath, &FileResourceAdapter{finfo, rpath})
  result = append(result, resource)

  // if the file is a dir, add its children to the result list
  if withChildren && finfo.IsDir() {
    dirFiles, _ := f.Readdir(0)
    for _, finfo := range dirFiles {
      childPath := files.JoinPaths(rpath, finfo.Name())
      resource = NewResource(childPath, &FileResourceAdapter{finfo, childPath})
      result = append(result, resource)
    }
  }

  return result, nil
}

func (fs *FileStorage) GetResourcesByFilters(rpath string, filters *ResourceFilter) ([]Resource, error) {
  result := []Resource{}

  childPaths := fs.getDirectoryChildPaths(rpath)
  for _, path := range childPaths {
    resource, _, err := fs.GetResource(path)

    if err != nil {
      // if we can't find this resource, something weird went wrong, but not that serious, so we log it and continue
      log.Printf("WARNING: returned error when trying to get resource with path %s from collection with path %s. Error: %s", path, rpath, err)
      continue
    }

    // only add it if the resource matches the filters
    if filters == nil || filters.Match(resource) {
      result = append(result, *resource)
    }
  }

  return result, nil
}

/*
 * Since file access is realtively cheap, we just read by fanning out to `GetResource`
 */
func (fs *FileStorage) GetResourcesByList(rpaths []string) ([]Resource, error) {
  results := []Resource{}

  for _, rpath := range rpaths {
    resource, found, err := fs.GetResource(rpath)

    if err != nil && err != errs.ResourceNotFoundError {
      return nil, err
    }

    if found {
      results = append(results, *resource)
    }
  }

  return results, nil
}

func (fs *FileStorage) GetResource(rpath string) (*Resource, bool, error) {
  resources, err := fs.GetResources(rpath, false)

  if err != nil {
    return nil, false, err
  }

  if resources == nil || len(resources) == 0 {
    return nil, false, errs.ResourceNotFoundError
  }

  res := resources[0]
  return &res, true, nil
}

func (fs *FileStorage) IsResourcePresent(rpath string) bool {
  _, found, _ := fs.GetResource(rpath)

  return found
}

func (fs *FileStorage) CreateResource(rpath, content string) (*Resource, error) {
  rAbsPath := files.AbsPath(rpath)

  if fs.IsResourcePresent(rAbsPath) {
    return nil, errs.ResourceAlreadyExistsError
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
  res := NewResource(rpath, &FileResourceAdapter{finfo, rpath})
  return &res, nil
}

func (fs *FileStorage) UpdateResource(rpath, content string) (*Resource, error) {
  f, e := fs.openResourceFile(rpath, os.O_RDWR)
  if e != nil {
		return nil, e
  }

  // update content
  f.Truncate(0)
  f.WriteString(content)

  finfo, _ := f.Stat()
  res := NewResource(rpath, &FileResourceAdapter{finfo, rpath})
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
			return nil, errs.ResourceNotFoundError
		}
		return nil, e
  }

  return f, nil
}

func (fs *FileStorage) getDirectoryChildPaths(dirpath string) []string {
  content, err := ioutil.ReadDir(files.AbsPath(dirpath))
	if err != nil {
    log.Printf("ERROR: Could not read resource as file directory.\nError: %s.\nResource path: %s.", err, dirpath)
    return nil
	}

  result := []string{}
	for _, file := range content {
    fpath := files.JoinPaths(dirpath, file.Name())
    result = append(result, fpath)
	}

  return result
}
