package caldav

import (
  "net/http"
)

type RequestPreconditions struct {
  request *http.Request
}

func (p *RequestPreconditions) IfMatch(etag string) bool {
  etagMatch := p.request.Header["If-Match"]
  return len(etagMatch) == 0 || etagMatch[0] == "*" || etagMatch[0] == etag
}

func (p *RequestPreconditions) IfMatchPresent() bool {
  return len(p.request.Header["If-Match"]) != 0
}

func (p *RequestPreconditions) IfNoneMatch(value string) bool {
  valueMatch := p.request.Header["If-None-Match"]
  return len(valueMatch) == 1 && valueMatch[0] == value
}
