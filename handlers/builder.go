package handlers

import (
	"net/http"
)

type handlerInterface interface {
	Handle() *Response
}

func NewHandler(request *http.Request) handlerInterface {
	response := NewResponse()

	switch request.Method {
	case "GET":
		return getHandler{request, response, false}
	case "HEAD":
		return getHandler{request, response, true}
	case "PUT":
		return putHandler{request, response}
	case "DELETE":
		return deleteHandler{request, response}
	case "PROPFIND":
		return propfindHandler{request, response}
	case "OPTIONS":
		return optionsHandler{response}
	case "REPORT":
		return reportHandler{request, response}
	default:
		return notImplementedHandler{response}
	}
}
