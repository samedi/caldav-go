package handlers

import (
  "io"
  "net/http"
  "git.samedi.cc/ferraz/caldav/errs"
)

type Response struct {
  Status int
  Header http.Header
  Body string
  Error error
}

func NewResponse() *Response {
  return &Response{
    Header: make(http.Header),
  }
}

func (this *Response) Set(status int, body string) *Response {
  this.Status = status
  this.Body = body

  return this
}

func (this *Response) SetHeader(key, value string) *Response {
  this.Header.Set(key, value)

  return this
}

func (this *Response) SetError(err error) *Response {
  this.Error = err

  switch err {
  case errs.ResourceNotFoundError:
    this.Status = http.StatusNotFound
  case errs.UnauthorizedError:
    this.Status = http.StatusUnauthorized
  case errs.ForbiddenError:
    this.Status = http.StatusForbidden
  default:
    this.Status = http.StatusInternalServerError
  }

  return this
}

func (this *Response) Write(writer http.ResponseWriter) {
  if this.Error == errs.UnauthorizedError {
    this.SetHeader("WWW-Authenticate", `Basic realm="Restricted"`)
  }

  for key, values := range this.Header {
    for _, value := range values {
      writer.Header().Set(key, value)
    }
  }

  writer.WriteHeader(this.Status)
  io.WriteString(writer, this.Body)
}
