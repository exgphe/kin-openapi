package routers

import (
	"github.com/exgphe/kin-openapi/openapi3"
)

// Router helps link http.Request.s and an OpenAPIv3 spec
//type Router = *legacy.Router

// Route describes the operation an http.Request can match
type Route struct {
	Spec      *openapi3.T
	Server    *openapi3.Server
	Path      string
	PathItem  *openapi3.PathItem
	Method    string
	Operation *openapi3.Operation
}

// ErrPathNotFound is returned when no route match is found
var ErrPathNotFound error = &RouteError{"no matching operation was found"}

// ErrMethodNotAllowed is returned when no method of the matched route matches
var ErrMethodNotAllowed error = &RouteError{"method not allowed"}

// RouteError describes Router errors
type RouteError struct {
	Reason string
}

func (e *RouteError) Error() string { return e.Reason }
