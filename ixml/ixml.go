package ixml

import (
  "fmt"
  "bytes"
  "net/http"
  "encoding/xml"
)

func Namespaces() string {
  return `xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/"`
}

func Tag(xmlName xml.Name, content string) string {
  name := xmlName.Local
  ns  := ""
  switch xmlName.Space {
  case "DAV:":
      ns = "D:"
  case "urn:ietf:params:xml:ns:caldav":
      ns = "C:"
  case "http://calendarserver.org/ns/":
      ns = "CS:"
  }

  if content != "" {
    return fmt.Sprintf("<%s%s>%s</%s%s>", ns, name, content, ns, name)
  } else {
    return fmt.Sprintf("<%s%s/>", ns, name)
  }
}

func StatusTag(status int) (tag string) {
  tag = fmt.Sprintf("<D:status>HTTP/1.1 %d %s</D:status>", status, http.StatusText(status))
  return
}

func EscapeText(content string) string {
  buffer := bytes.NewBufferString("")
  xml.EscapeText(buffer, []byte(content))

  return buffer.String()
}
