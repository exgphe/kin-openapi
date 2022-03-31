// Package legacy implements a router.
//
// It differs from the gorilla/mux router:
// * it provides granular errors: "path not found", "method not allowed", "variable missing from path"
// * it does not handle matching routes with extensions (e.g. /books/{id}.json)
// * it handles path patterns with a different syntax (e.g. /params/{x}/{y}/{z.*})
package legacy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/exgphe/kin-openapi/openapi3"
	"github.com/exgphe/kin-openapi/routers"
	"github.com/exgphe/kin-openapi/routers/legacy/pathpattern"
)

// Routers maps a HTTP request to a Router.
type Routers []*Router

// FindRoute extracts the route and parameters of an http.Request
func (rs Routers) FindRoute(req *http.Request) (routers.Router, *routers.Route, map[string]string, error) {
	for _, router := range rs {
		// Skip routers that have DO NOT have servers
		if len(router.Doc.Servers) == 0 {
			continue
		}
		route, pathParams, err := router.FindRoute(req)
		if err == nil {
			return router, route, pathParams, nil
		}
	}
	for _, router := range rs {
		// Skip routers that DO have servers
		if len(router.Doc.Servers) > 0 {
			continue
		}
		route, pathParams, err := router.FindRoute(req)
		if err == nil {
			return router, route, pathParams, nil
		}
	}
	return nil, nil, nil, &routers.RouteError{
		Reason: "none of the routers match",
	}
}

// Router maps a HTTP request to an OpenAPI operation.
type Router struct {
	Doc      *openapi3.T
	PathNode *pathpattern.Node
}

// NewRouter creates a new router.
//
// If the given OpenAPIv3 document has servers, router will use them.
// All operations of the document will be added to the router.
func NewRouter(doc *openapi3.T) (routers.Router, error) {
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validating OpenAPI failed: %v", err)
	}
	router := &Router{Doc: doc}
	root := router.Node()
	for path, pathItem := range doc.Paths {
		for method, operation := range pathItem.Operations() {
			method = strings.ToUpper(method)
			if err := root.Add(method+" "+path, &routers.Route{
				Spec:      doc,
				Path:      path,
				PathItem:  pathItem,
				Method:    method,
				Operation: operation,
			}, nil); err != nil {
				return nil, err
			}
		}
	}
	return router, nil
}

// AddRoute adds a route in the router.
func (router *Router) AddRoute(route *routers.Route) error {
	method := route.Method
	if method == "" {
		return errors.New("route is missing method")
	}
	method = strings.ToUpper(method)
	path := route.Path
	if path == "" {
		return errors.New("route is missing path")
	}
	return router.Node().Add(method+" "+path, router, nil)
}

func (router *Router) Node() *pathpattern.Node {
	root := router.PathNode
	if root == nil {
		root = &pathpattern.Node{}
		router.PathNode = root
	}
	return root
}

// FindRoute extracts the route and parameters of an http.Request
func (router *Router) FindRoute(req *http.Request) (*routers.Route, map[string]string, error) {
	method, url := req.Method, req.URL
	doc := router.Doc

	// Get server
	servers := doc.Servers
	var server *openapi3.Server
	var remainingPath string
	var pathParams map[string]string
	if len(servers) == 0 {
		remainingPath = url.Path
	} else {
		var paramValues []string
		server, paramValues, remainingPath = servers.MatchURL(url)
		if server == nil {
			return nil, nil, &routers.RouteError{
				Reason: routers.ErrPathNotFound.Error(),
			}
		}
		pathParams = make(map[string]string, 8)
		paramNames, _ := server.ParameterNames()
		for i, value := range paramValues {
			name := paramNames[i]
			pathParams[name] = value
		}
	}

	// Get PathItem
	root := router.Node()
	var route *routers.Route
	node, paramValues := root.Match(method + " " + remainingPath)
	if node != nil {
		route, _ = node.Value.(*routers.Route)
	}
	if route == nil {
		pathItem := doc.Paths[remainingPath]
		if pathItem == nil {
			return nil, nil, &routers.RouteError{Reason: routers.ErrPathNotFound.Error()}
		}
		if pathItem.GetOperation(method) == nil {
			return nil, nil, &routers.RouteError{Reason: routers.ErrMethodNotAllowed.Error()}
		}
	}

	if pathParams == nil {
		pathParams = make(map[string]string, len(paramValues))
	}
	paramKeys := node.VariableNames
	for i, value := range paramValues {
		key := paramKeys[i]
		if strings.HasSuffix(key, "*") {
			key = key[:len(key)-1]
		}
		pathParams[key] = value
	}
	return route, pathParams, nil
}
