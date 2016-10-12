# go CalDAV

This is a Go lib that aims to implement the CalDAV specification ([RFC4791]). It allows the quick implementation of a CalDAV server in Go. Basically, it provides the request handlers that will handle the several CalDAV HTTP requests, fetch the appropriate resources, build and return the responses.

### How to install

```
go get git.samedi.cc/ferraz/caldav
```

### How to use it

The easiest way to quickly implement a CalDAV server is by just using the lib's request handler. Example:

```go
package mycaldav

import (
  "net/http"
  "git.samedi.cc/ferraz/caldav"
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
  "git.samedi.cc/ferraz/caldav"
)

func runServer() {
  http.HandleFunc(PATH, myHandler)
  http.ListenAndServe(PORT, nil)
}

func myHandler(writer http.ResponseWriter, request *http.Request) {
  response := caldav.HandleRequest(writer, request)
  // ... do something with the `response` ...
  response.Write(writer) // the response is written in the current `ResponseWriter` and ready to be sent back
}
```

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
