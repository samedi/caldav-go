package handlers

import (
	"encoding/xml"
)

type propfindHandler struct {
	handlerData
}

func (ph propfindHandler) Handle() *Response {
	// get the target resources based on the request URL
	resources, err := ph.storage.GetResources(ph.requestPath, ph.headers.IsDeep())
	if err != nil {
		return ph.response.SetError(err)
	}

	// read body string to xml struct
	type XMLProp2 struct {
		Tags []xml.Name `xml:",any"`
	}
	type XMLRoot2 struct {
		XMLName xml.Name
		Prop    XMLProp2 `xml:"DAV: prop"`
	}
	var requestXML XMLRoot2
	xml.Unmarshal([]byte(ph.requestBody), &requestXML)

	multistatus := &multistatusResp{
		Minimal: ph.headers.IsMinimal(),
	}
	// for each href, build the multistatus responses
	for _, resource := range resources {
		propstats := multistatus.Propstats(&resource, requestXML.Prop.Tags)
		multistatus.AddResponse(resource.Path, true, propstats)
	}

	if multistatus.Minimal {
		ph.response.SetHeader(HD_PREFERENCE_APPLIED, HD_PREFER_MINIMAL)
	}

	return ph.response.Set(207, multistatus.ToXML())
}
