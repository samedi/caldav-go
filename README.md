# go CalDAV

This is a Go lib that aims to implement the CalDAV specification ([RFC4791]). It allows the quick implementation of a CalDAV server in Go. Basically, it provides the request handlers that will handle the several CalDAV HTTP requests, fetch the appropriate resources, build and return the responses.

### How to install

```
go get github.com/samedi/caldav-go
```

### Dependencies

For dependency management, `glide` is used.

```bash
# install glide (once!)
curl https://glide.sh/get | sh

# install dependencies
glide install
```

### How to use it

The easiest way to quickly implement a CalDAV server is by just using the lib's request handler. Example:

```go
package mycaldav

import (
  "net/http"
  "github.com/samedi/caldav-go"
)

func runServer() {
  http.HandleFunc(PATH, caldav.RequestHandler)
  http.ListenAndServe(PORT, nil)
}
```

With that, all the HTTP requests (GET, PUT, REPORT, PROPFIND, etc) will be handled and responded by the `caldav` handler. In case of any HTTP methods not supported by the lib, a `501 Not Implemented` response will be returned.

In case you want more flexibility to handle the requests, e.g., if you wanted to access the generated response before being sent back to the caller, you could do like:

```go
package mycaldav

import (
  "net/http"
  "github.com/samedi/caldav-go"
)

func runServer() {
  http.HandleFunc(PATH, myHandler)
  http.ListenAndServe(PORT, nil)
}

func myHandler(writer http.ResponseWriter, request *http.Request) {
  response := caldav.HandleRequest(writer, request)
  // ... do something with the `response` ...
  // the response is written with the current `http.ResponseWriter` and ready to be sent back
  response.Write(writer)
}
```

### Storage & Resources

The storage is where the caldav resources are stored. To interact with that, the caldav lib needs only a type that conforms with the  `data.StorageInterface` to operate on top of the storage. Basically, this interface defines all the CRUD functions to work on top of the resources. With that, resources can be stored anywhere: in the filesystem, in the cloud, database, etc. As long as the used storage implements all the required storage interface functions, the caldav lib will work fine.

For example, we could use the following dummy storage implementation:

```go
type DummyStorage struct{
}

func (d *DummyStorage) GetResources(rpath string, withChildren bool) ([]Resource, error) {
  return []Resource{}, nil
}

func (d *DummyStorage) GetResourcesByFilters(rpath string, filters *ResourceFilter) ([]Resource, error) {
  return []Resource{}, nil
}

func (d *DummyStorage) GetResourcesByList(rpaths []string) ([]Resource, error) {
  return []Resource{}, nil
}

func (d *DummyStorage) GetResource(rpath string) (*Resource, bool, error) {
  return nil, false, nil
}

func (d *DummyStorage) CreateResource(rpath, content string) (*Resource, error) {
  return nil, nil
}

func (d *DummyStorage) UpdateResource(rpath, content string) (*Resource, error) {
  return nil, nil
}

func (d *DummyStorage) DeleteResource(rpath string) error {
  return nil
}
```

Then we just need to tell the caldav lib to use our dummy storage:

```go
dummyStg := new(DummyStorage)
caldav.SetupStorage(dummyStg)
```

All the CRUD operations on resources will then be forwarded to our dummy storage.

The default storage used (if none is explicitly set) is the `data.FileStorage` which deals with resources as files in the File System.

The resources can be of two types: collection and non-collection. A collection resource is basically a resource that has children resources, but does not have any data content. A non-collection resource is a resource that does not have children, but has data. In the case of a file storage, collections correspond to directories and non-collection to plain files. The data of a caldav resource is all the info that shows up in the calendar client, in the [iCalendar](https://en.wikipedia.org/wiki/ICalendar) format.

### Features

Please check the **CHANGELOG** to see specific features that are currently implemented.

### Contributing and testing

Everyone is welcome to contribute. Please raise an issue or pull request accordingly.

To run the tests:

```
./test.sh
```

### License

MIT License.

[RFC4791]: https://tools.ietf.org/html/rfc4791
