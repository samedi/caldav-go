package data

import (
  "os"
  "log"
  "io/ioutil"
  "git.samedi.cc/ferraz/caldav/errs"
  "git.samedi.cc/ferraz/caldav/files"
)

// The Storage is the responsible for the CRUD operations on the caldav resources.
type Storage interface {
  // GetResources gets a list of resources based on a given `rpath`. The
  // `rpath` is the path to the original resource that's being requested. The resultant list
  // will/must contain that original resource in it, apart from any additional resources. It also receives
  // `withChildren` flag to say if the result must also include all the original resource`s
  // children (if original is a collection resource). If `true`, the result will have the requested resource + children.
  // If `false`, it will have only the requested original resource (from the `rpath` path).
  // It returns errors if anything went wrong or if it could not find any resource on `rpath` path.
  GetResources(rpath string, withChildren bool) ([]Resource, error)
  // GetResourcesByList fetches a list of resources by path from the storage.
  // This method fetches all the `rpaths` and return an array of the reosurces found.
  // No error 404 will be returned if one of the resources cannot be found.
  // Errors are returned if any errors other than "not found" happens.
  GetResourcesByList(rpaths []string) ([]Resource, error)
  // GetResourcesByFilters returns the filtered children of a target collection resource.
  // The target collection resource is the one pointed by the `rpath` parameter. All of its children
  // will be checked against a set of `filters` and the matching ones are returned. The results
  // contains only the filtered children and does NOT include the target resource. If the target resource
  // is not a collection, an empty array is returned as the result.
  GetResourcesByFilters(rpath string, filters *ResourceFilter) ([]Resource, error)
  // GetResource gets the requested resource based on a given `rpath` path. It returns the resource (if found) or
  // nil (if not found). Also returns a flag specifying if the resource was found or not.
  GetResource(rpath string) (*Resource, bool, error)
  // IsResourcePresent checks if any resource exists on the given `rpath` path.
  IsResourcePresent(rpath string) bool
  // CreateResource creates a new resource on the `rpath` path with a given `content`.
  CreateResource(rpath, content string) (*Resource, error)
  // UpdateResource udpates a resource on the `rpath` path with a given `content`.
  UpdateResource(rpath, content string) (*Resource, error)
  // DeleteResource deletes a resource on the `rpath` path.
  DeleteResource(rpath string) error
}

// FileStorage is the storage that deals with resources as files in the file system. So, a collection resource
// is treated as a folder/directory and its children resources are the files it contains. On the other hand, non-collection
// resources are just plain files.
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
