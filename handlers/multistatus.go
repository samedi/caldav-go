package handlers

import (
  "fmt"
  "net/http"
  "encoding/xml"
  "git.samedi.cc/ferraz/caldav/lib"
  "git.samedi.cc/ferraz/caldav/data"
  "git.samedi.cc/ferraz/caldav/ixml"
)

// Wraps a multistatus response. It contains the set of `Responses`
// that will serve to build the final XML. Multistatus responses are
// used by the REPORT and PROPFIND methods.
type multistatusResp struct {
  Responses []msResponse
}

func newMultistatusResp() multistatusResp {
  ms := multistatusResp{}
  ms.Responses = []msResponse{}

  return ms
}

type msResponse struct {
  Href string
  Found bool
  Propstats map[int][]msPropValue
}

type msPropValue struct {
  Tag      xml.Name
  Content  string
  Contents []string
  Status   int
}

// Function that processes all the required props for a given resource.
// ## Params
// resource: the target calendar resource.
// reqprops: set of required props that must be processed for the resource.
// ## Returns
// The set of props (msPropValue) processed. Each prop is mapped to a HTTP status code.
// So if a prop is found and processed ok, it'll be mapped to 200. If it's not found,
// it'll be mapped to 404, and so on.
func (ms *multistatusResp) Propstats(resource *data.Resource, reqprops []xml.Name) map[int][]msPropValue {
  if resource == nil {
    return nil
  }

  result := make(map[int][]msPropValue)

  for _, ptag := range reqprops {
    pvalue := msPropValue{
      Tag: ptag,
      Status: http.StatusOK,
    }

    pfound := false
    switch ptag {
    case xml.Name{Space:"urn:ietf:params:xml:ns:caldav", Local:"calendar-data"}:
      pvalue.Content, pfound = resource.GetContentData()
    case xml.Name{Space: "DAV:", Local: "getetag"}:
      pvalue.Content, pfound = resource.GetEtag()
    case xml.Name{Space: "DAV:", Local: "getcontenttype"}:
      pvalue.Content, pfound = resource.GetContentType()
    case xml.Name{Space: "DAV:", Local: "getcontentlength"}:
      pvalue.Content, pfound = resource.GetContentLength()
    case xml.Name{Space: "DAV:", Local: "displayname"}:
      pvalue.Content, pfound = resource.GetDisplayName()
    case xml.Name{Space: "DAV:", Local: "getlastmodified"}:
      pvalue.Content, pfound = resource.GetLastModified(http.TimeFormat)
    case xml.Name{Space: "DAV:", Local: "owner"}:
      pvalue.Content, pfound = resource.GetOwnerPath()
    case xml.Name{Space: "http://calendarserver.org/ns/", Local: "getctag"}:
      pvalue.Content, pfound = resource.GetEtag()
    case xml.Name{Space: "DAV:", Local: "principal-URL"},
         xml.Name{Space: "DAV:", Local: "principal-collection-set"},
         xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "calendar-user-address-set"},
         xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "calendar-home-set"}:
      pvalue.Content, pfound = fmt.Sprintf("<D:href>%s</D:href>", resource.Path), true
    case xml.Name{Space: "DAV:", Local: "resourcetype"}:
      if resource.IsCollection() {
        pvalue.Content, pfound = "<D:collection/><C:calendar/>", true

        if resource.IsPrincipal() {
          pvalue.Content += "<D:principal/>"
        }
      } else {
        // resourcetype must be returned empty for non-collection elements
        pvalue.Content, pfound = "", true
      }
    case xml.Name{Space: "DAV:", Local: "current-user-principal"}:
      currentUser := getCurrentUser()
      if currentUser != nil {
        pvalue.Content, pfound = fmt.Sprintf("<D:href>/%s/</D:href>", currentUser.Name), true
      }
    case xml.Name{Space: "urn:ietf:params:xml:ns:caldav", Local: "supported-calendar-component-set"}:
      if resource.IsCollection() {
        for _, component := range supportedComponents {
          compTag := fmt.Sprintf(`<C:comp name="%s"/>`, component)
          pvalue.Contents = append(pvalue.Contents, compTag)
        }
        pfound = true
      }
    }

    if !pfound {
      pvalue.Status = http.StatusNotFound
    }

    result[pvalue.Status] = append(result[pvalue.Status], pvalue)
  }

  return result
}

// Adds a new `msResponse` to the `Responses` array.
func (ms *multistatusResp) AddResponse(href string, found bool, propstats map[int][]msPropValue) {
  ms.Responses = append(ms.Responses, msResponse{
    Href: href,
    Found: found,
    Propstats: propstats,
  })
}

func (ms *multistatusResp) ToXML() string {
  // init multistatus
  var bf lib.StringBuffer
  bf.Write(`<?xml version="1.0" encoding="UTF-8"?>`)
  bf.Write(`<D:multistatus %s>`, ixml.Namespaces())

  // iterate over event hrefs and build multistatus XML on the fly
  for _, response := range ms.Responses {
    bf.Write("<D:response>")
    bf.Write("<D:href>%s</D:href>", response.Href)

    if response.Found {
      for status, props := range response.Propstats {
        bf.Write("<D:propstat>")
        bf.Write("<D:prop>")
        for _, prop := range props {
          bf.Write(ms.propToXML(prop))
        }
        bf.Write("</D:prop>")
        bf.Write(ixml.StatusTag(status))
        bf.Write("</D:propstat>")
      }
    } else {
      // if does not find the resource set 404
      bf.Write(ixml.StatusTag(http.StatusNotFound))
    }
    bf.Write("</D:response>")
  }
  bf.Write("</D:multistatus>")

  return bf.String()
}

func (ms *multistatusResp) propToXML(pv msPropValue) string {
  for _, content := range pv.Contents {
    pv.Content += content
  }
  xmlString := ixml.Tag(pv.Tag, pv.Content)
  return xmlString
}