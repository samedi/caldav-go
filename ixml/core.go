package ixml

import (
  "fmt"
  "net/http"
  "encoding/xml"
)

func Tag(xmlName xml.Name, content string) string {
  name := xmlName.Local
  ns  := ""
  switch xmlName.Space {
  case "DAV:":
      ns = "D:"
  case "urn:ietf:params:xml:ns:caldav":
      ns = "C:"
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
