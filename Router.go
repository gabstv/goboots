package goboots

// derived from: https://raw.githubusercontent.com/revel/revel/master/router.go

//  Copyright (C) 2012 Rob Figueiredo
//  All Rights Reserved.
//
//  MIT LICENSE
//
//  Permission is hereby granted, free of charge, to any person obtaining a copy of
//  this software and associated documentation files (the "Software"), to deal in
//  the Software without restriction, including without limitation the rights to
//  use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
//  the Software, and to permit persons to whom the Software is furnished to do so,
//  subject to the following conditions:
//
//  The above copyright notice and this permission notice shall be included in all
//  copies or substantial portions of the Software.
//
//  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
//  FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
//  COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
//  IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
//  CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/robfig/pathtree"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Route struct {
	Method         string   // e.g. GET
	Path           string   // e.g. /app/:id
	Action         string   // e.g. "Application.ShowApp", "404"
	ControllerName string   // e.g. "Application", ""
	MethodName     string   // e.g. "ShowApp", ""
	FixedParams    []string // e.g. "arg1","arg2","arg3" (CSV formatting)
	TreePath       string   // e.g. "/GET/app/:id"
	TLSOnly        bool

	routesPath string // e.g. /Users/robfig/gocode/src/myapp/conf/routes
	line       int    // e.g. 3
	app        *App
}

type RouteMatch struct {
	Action         string // e.g. 404
	ControllerName string // e.g. Application
	MethodName     string // e.g. ShowApp
	FixedParams    []string
	Params         Params // e.g. {id: 123}
	TLSOnly        bool
}

var routeMatchNotFound = &RouteMatch{Action: "404"}

type arg struct {
	name       string
	index      int
	constraint *regexp.Regexp
}

// Prepares the route to be used in matching.
func NewRoute(method, path, action, fixedArgs, routesPath string, line int, tlsonly bool, app *App) (r *Route) {
	// Handle fixed arguments
	argsReader := strings.NewReader(fixedArgs)
	csv := csv.NewReader(argsReader)
	csv.TrimLeadingSpace = true
	fargs, err := csv.Read()
	if err != nil && err != io.EOF {
		app.Logger.Printf("Invalid fixed parameters (%v): for string '%v'\n", err.Error(), fixedArgs)
	}

	r = &Route{
		Method:      strings.ToUpper(method),
		Path:        path,
		Action:      action,
		FixedParams: fargs,
		TreePath:    treePath(strings.ToUpper(method), path),
		TLSOnly:     tlsonly,
		routesPath:  routesPath,
		line:        line,
		app:         app,
	}

	// URL pattern
	if !strings.HasPrefix(r.Path, "/") {
		app.Logger.Println("Absolute URL required.")
		return
	}

	actionSplit := strings.Split(action, ".")
	if len(actionSplit) == 2 {
		r.ControllerName = actionSplit[0]
		r.MethodName = actionSplit[1]
	}

	return
}

func treePath(method, path string) string {
	if method == "*" {
		method = ":METHOD"
	}
	return "/" + method + path
}

type Router struct {
	Routes []*Route
	Tree   *pathtree.Node
	path   string // path to the routes file
	app    *App
}

func (router *Router) Route(req *http.Request) *RouteMatch {
	// Override method if set in header
	if method := req.Header.Get("X-HTTP-Method-Override"); method != "" && req.Method == "POST" {
		req.Method = method
	}

	leaf, expansions := router.Tree.Find(treePath(req.Method, req.URL.Path))
	if leaf == nil {
		return nil
	}
	route := leaf.Value.(*Route)

	// Create a map of the route parameters.
	var params Params
	if len(expansions) > 0 {
		params = make(Params)
		for i, v := range expansions {
			params[leaf.Wildcards[i]] = v
		}
	}

	// Special handling for explicit 404's.
	if route.Action == "404" {
		return routeMatchNotFound
	}

	// If the action is variablized, replace into it with the captured args.
	controllerName, methodName := route.ControllerName, route.MethodName
	if controllerName[0] == ':' {
		controllerName = params[controllerName[1:]]
	}
	if methodName[0] == ':' {
		methodName = params[methodName[1:]]
	}

	return &RouteMatch{
		ControllerName: controllerName,
		MethodName:     methodName,
		Params:         params,
		FixedParams:    route.FixedParams,
		TLSOnly:        route.TLSOnly,
	}
}

// Refresh re-reads the routes file and re-calculates the routing table.
// Returns an error if a specified action could not be found.
func (router *Router) Refresh() (err error) {
	router.Routes, err = parseRoutesFile(router.path, "", true, router.app)
	if err != nil {
		return
	}
	err = router.updateTree()
	return
}

func (router *Router) updateTree() error {
	router.Tree = pathtree.New()
	for _, route := range router.Routes {
		err := router.Tree.Add(route.TreePath, route)

		// Allow GETs to respond to HEAD requests.
		if err == nil && route.Method == "GET" {
			err = router.Tree.Add(treePath("HEAD", route.Path), route)
		}

		// Error adding a route to the pathtree.
		if err != nil {
			return routeError(err, route.routesPath, "", route.line)
		}
	}
	return nil
}

// parseRoutesFile reads the given routes file and returns the contained routes.
func parseRoutesFile(routesPath, joinedPath string, validate bool, app *App) ([]*Route, error) {
	contentBytes, err := ioutil.ReadFile(routesPath)
	if err != nil {
		return nil, errors.New("Failed to load routes file: " + err.Error())
	}
	return parseRoutes(routesPath, joinedPath, string(contentBytes), validate, app)
}

// parseRoutes reads the content of a routes file into the routing table.
func parseRoutes(routesPath, joinedPath, content string, validate bool, app *App) ([]*Route, error) {
	var routes []*Route

	for n, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// A single route
		method, path, action, fixedArgs, tls, found, errmsg, errbyte := routeParseLine(line)
		if !found {
			app.Logger.Printf("ROUTER ERROR on line %d:\n%s <<[%d] %s\n", n, line[:errbyte], errbyte, errmsg)
			continue
		}

		// this will avoid accidental double forward slashes in a route.
		// this also avoids pathtree freaking out and causing a runtime panic
		// because of the double slashes
		if strings.HasSuffix(joinedPath, "/") && strings.HasPrefix(path, "/") {
			joinedPath = joinedPath[0 : len(joinedPath)-1]
		}
		path = strings.Join([]string{joinedPath, path}, "")

		route := NewRoute(method, path, action, fixedArgs, routesPath, n, tls, app)
		routes = append(routes, route)

		if validate {
			if err := validateRoute(route); err != nil {
				return nil, routeError(err, routesPath, content, n)
			}
		}
	}

	return routes, nil
}

// validateRoute checks that every specified action exists.
func validateRoute(route *Route) error {
	// Skip 404s
	if route.Action == "404" {
		return nil
	}

	// We should be able to load the action.
	parts := strings.Split(route.Action, ".")
	if len(parts) != 2 {
		return fmt.Errorf("Expected two parts (Controller.Action), but got %d: %s",
			len(parts), route.Action)
	}

	// Skip variable routes.
	if parts[0][0] == ':' || parts[1][0] == ':' {
		return nil
	}

	c := route.app.controllerMap[parts[0]]
	if c == nil {
		return errors.New("Controller " + parts[0] + " not found!")
	}

	if _, ok := c.getMethod(parts[1]); !ok {
		return errors.New("Controller " + parts[0] + " has no method " + parts[1])
	}

	return nil
}

func routeParseLine(line string) (method, path, action, fixedArgs string, tls, found bool, errormessage string, errorbyte int) {
	stage := 0
	begin := false
	quoted := false
	bbackslashes := 0
	buf := new(bytes.Buffer)
	for i, r := range line {
		errorbyte = i
		switch stage {
		case 0:
			// METHOD
			if !begin {
				if r == ' ' || r == '\t' {
					continue
				} else {
					begin = true
					buf.WriteRune(r)
				}
			} else {
				if r == ' ' || r == '\t' {
					method = buf.String()
					switch method {
					case "GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD", "WS", "*":
						stage = 1
						begin = false
						buf.Reset()
					default:
						errormessage = fmt.Sprintf("Method %s is not valid!", method)
						found = false
						return
					}
				} else {
					buf.WriteRune(r)
				}
			}
		case 1:
			// PATH
			if !begin {
				if r == ' ' || r == '\t' {
					continue
				} else {
					if r != '/' {
						// Paths should always brgin with a trailing slash
						//TODO: error notify
						errormessage = fmt.Sprintf("Paths should always brgin with a trailing slash")
						found = false
						return
					}
					begin = true
					buf.WriteRune(r)
				}
			} else {
				if r == ' ' || r == '\t' {
					path = buf.String()
					stage = 2
					buf.Reset()
					begin = false
				} else {
					buf.WriteRune(r)
				}
			}
		case 2:
			// Controller.Action
			if !begin {
				if r == ' ' || r == '\t' {
					continue
				} else {
					begin = true
					buf.WriteRune(r)
				}
			} else {
				if r == ' ' || r == '\t' || r == '(' {
					if r == '(' {
						// jump to fixedArgs
						stage = 3
					} else {
						stage = 4
						// jump to TLS checker
					}
					action = buf.String()
					buf.Reset()
					begin = false
				} else {
					buf.WriteRune(r)
				}
			}
		case 3:
			// fixedArgs (this is an optional parameter)
			if !begin {
				if r == '#' {
					// it's a comment without closing the parenthesis! (wtf)
					found = false
					return
				}
				if r == ' ' || r == '\t' {
					continue
				} else {
					begin = true
					buf.WriteRune(r)
					if r == '"' {
						quoted = true
					}
				}
			} else {
				if quoted {
					if r == '\\' {
						bbackslashes++
					}
					if r == '"' {
						if bbackslashes%2 == 0 {
							quoted = false
							bbackslashes = 0
						}
					}
					if r != '\\' {
						bbackslashes = 0
					}
					buf.WriteRune(r)
				} else {
					if r == ')' {
						begin = false
						quoted = false
						bbackslashes = 0
						fixedArgs = buf.String()
						buf.Reset()
						found = true
						stage = 4 // go to TLS check
						continue
					}
					if r != ' ' && r != '\t' && r != ',' && r != '"' {
						// bad character between records
						errormessage = fmt.Sprintf("bad character (%v) between fixedArgs", string(r))
						found = false
						return
					}
					if r != ' ' && r != '\t' {
						buf.WriteRune(r)
					}
					if r == '"' {
						quoted = true
					}
				}
			}
		case 4:
			// mandatory TLS check (this is an optional parameter)
			if !begin {
				if r == '#' {
					// it's a comment; end this early
					found = true
					return
				}
				if r == ' ' || r == '\t' {
					continue
				} else {
					begin = true
					buf.WriteRune(r)
				}
			} else {
				if r == '#' {
					// since we began the step, this transforms the route
					// into an invalid one
					errormessage = "found a comment while parsing TLS check"
					found = false
					return
				} else if r == ' ' || r == '\t' {
					bs := buf.String()
					if bs == "TLS" {
						tls = true
						found = true
						return
					} else {
						// at this moment, there is no other valid
						// parameter besides TLS
						errormessage = "TLS parameter `" + bs + "` invalid"
						found = false
						return
					}
				} else {
					buf.WriteRune(r)
				}
			}
		}
	}
	if stage < 2 {
		// EOL before reaching mandatory stage 2
		errormessage = fmt.Sprintf("EOL before reaching action parameter; method: '%s' path: '%s' stage: %v", method, path, stage)
		found = false
		return
	}
	if stage == 2 {
		// EOL while still in stage 2
		action = buf.String()
		buf.Reset()
		found = true
		return
	}
	if stage == 3 {
		// EOL while still in stage 3
		// this shouldn't happen since the parenthesis should be closed
		buf.Reset()
		found = false
		return
	}
	if stage == 4 {
		if !begin {
			found = true
			return
		}
		if buf.String() == "TLS" {
			buf.Reset()
			tls = true
			found = true
			return
		}
	}
	return
}

// routeError adds context to a simple error message.
func routeError(err error, routesPath, content string, n int) error {
	return errors.New("Route validation error; " + err.Error() + "; " + routesPath + "; line " + fmt.Sprintf("%v", n+1))
}

func NewRouter(app *App, routesPath string) *Router {
	return &Router{
		Tree: pathtree.New(),
		path: routesPath,
		app:  app,
	}
}

type ActionDefinition struct {
	Host, Method, Url, Action string
	Star                      bool
	Args                      map[string]string
}

func (a *ActionDefinition) String() string {
	return a.Url
}

func (router *Router) Reverse(action string, argValues map[string]string) *ActionDefinition {
	actionSplit := strings.Split(action, ".")
	if len(actionSplit) != 2 {
		router.app.Logger.Println("router: reverse router got invalid action ", action)
		return nil
	}
	controllerName, methodName := actionSplit[0], actionSplit[1]

	for _, route := range router.Routes {
		// Skip routes without either a ControllerName or MethodName
		if route.ControllerName == "" || route.MethodName == "" {
			continue
		}

		// Check that the action matches or is a wildcard.
		controllerWildcard := route.ControllerName[0] == ':'
		methodWildcard := route.MethodName[0] == ':'
		if (!controllerWildcard && route.ControllerName != controllerName) ||
			(!methodWildcard && route.MethodName != methodName) {
			continue
		}
		if controllerWildcard {
			argValues[route.ControllerName[1:]] = controllerName
		}
		if methodWildcard {
			argValues[route.MethodName[1:]] = methodName
		}

		// Build up the URL.
		var (
			queryValues  = make(url.Values)
			pathElements = strings.Split(route.Path, "/")
		)
		for i, el := range pathElements {
			if el == "" || el[0] != ':' {
				continue
			}

			val, ok := argValues[el[1:]]
			if !ok {
				val = "<nil>"
				router.app.Logger.Println("router: reverse route missing route arg ", el[1:])
			}
			pathElements[i] = val
			delete(argValues, el[1:])
			continue
		}

		// Add any args that were not inserted into the path into the query string.
		for k, v := range argValues {
			queryValues.Set(k, v)
		}

		// Calculate the final URL and Method
		url := strings.Join(pathElements, "/")
		if len(queryValues) > 0 {
			url += "?" + queryValues.Encode()
		}

		method := route.Method
		star := false
		if route.Method == "*" {
			method = "GET"
			star = true
		}

		return &ActionDefinition{
			Url:    url,
			Method: method,
			Star:   star,
			Action: action,
			Args:   argValues,
			Host:   "TODO",
		}
	}
	router.app.Logger.Println("Failed to find reverse route:", action, argValues)
	return nil
}
