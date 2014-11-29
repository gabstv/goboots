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
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"errors"
	"github.com/robfig/pathtree"
	"log"
)

var AppRoot = "" //TODO: move/remove this

type Route struct {
	Method         string   // e.g. GET
	Path           string   // e.g. /app/:id
	Action         string   // e.g. "Application.ShowApp", "404"
	ControllerName string   // e.g. "Application", ""
	MethodName     string   // e.g. "ShowApp", ""
	FixedParams    []string // e.g. "arg1","arg2","arg3" (CSV formatting)
	TreePath       string   // e.g. "/GET/app/:id"

	routesPath string // e.g. /Users/robfig/gocode/src/myapp/conf/routes
	line       int    // e.g. 3
	app        *App
}

type RouteMatch struct {
	Action         string // e.g. 404
	ControllerName string // e.g. Application
	MethodName     string // e.g. ShowApp
	FixedParams    []string
	Params         map[string][]string // e.g. {id: 123}
}

type arg struct {
	name       string
	index      int
	constraint *regexp.Regexp
}

// Prepares the route to be used in matching.
func NewRoute(method, path, action, fixedArgs, routesPath string, line int, app *App) (r *Route) {
	// Handle fixed arguments
	argsReader := strings.NewReader(fixedArgs)
	csv := csv.NewReader(argsReader)
	csv.TrimLeadingSpace = true
	fargs, err := csv.Read()
	if err != nil && err != io.EOF {
		log.Printf("Invalid fixed parameters (%v): for string '%v'\n", err.Error(), fixedArgs)
	}

	r = &Route{
		Method:      strings.ToUpper(method),
		Path:        path,
		Action:      action,
		FixedParams: fargs,
		TreePath:    treePath(strings.ToUpper(method), path),
		routesPath:  routesPath,
		line:        line,
		app:         app,
	}

	// URL pattern
	if !strings.HasPrefix(r.Path, "/") {
		log.Println("Absolute URL required.")
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

var notFound = &RouteMatch{Action: "404"}

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
	var params url.Values
	if len(expansions) > 0 {
		params = make(url.Values)
		for i, v := range expansions {
			params[leaf.Wildcards[i]] = []string{v}
		}
	}

	// Special handling for explicit 404's.
	if route.Action == "404" {
		return notFound
	}

	// If the action is variablized, replace into it with the captured args.
	controllerName, methodName := route.ControllerName, route.MethodName
	if controllerName[0] == ':' {
		controllerName = params[controllerName[1:]][0]
	}
	if methodName[0] == ':' {
		methodName = params[methodName[1:]][0]
	}

	return &RouteMatch{
		ControllerName: controllerName,
		MethodName:     methodName,
		Params:         params,
		FixedParams:    route.FixedParams,
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
		method, path, action, fixedArgs, found := parseRouteLine(line)
		if !found {
			continue
		}

		// this will avoid accidental double forward slashes in a route.
		// this also avoids pathtree freaking out and causing a runtime panic
		// because of the double slashes
		if strings.HasSuffix(joinedPath, "/") && strings.HasPrefix(path, "/") {
			joinedPath = joinedPath[0 : len(joinedPath)-1]
		}
		path = strings.Join([]string{AppRoot, joinedPath, path}, "")

		route := NewRoute(method, path, action, fixedArgs, routesPath, n, app)
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

// routeError adds context to a simple error message.
func routeError(err error, routesPath, content string, n int) error {
	// Load the route file content if necessary
	if content == "" {
		contentBytes, err := ioutil.ReadFile(routesPath)
		if err != nil {
			log.Printf("Failed to read route file %s: %s\n", routesPath, err)
		} else {
			content = string(contentBytes)
		}
	}
	return errors.New("Route validation error; " + err.Error() + "; " + routesPath + "; line " + fmt.Sprintf("%v", n+1))
}

// Groups:
// 1: method
// 4: path
// 5: action
// 6: fixedargs
var routePattern *regexp.Regexp = regexp.MustCompile(
	"(?i)^(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD|WS|\\*)" +
		"[(]?([^)]*)(\\))?[ \t]+" +
		"(.*/[^ \t]*)[ \t]+([^ \t(]+)" +
		`\(?([^)]*)\)?[ \t]*$`)

func parseRouteLine(line string) (method, path, action, fixedArgs string, found bool) {
	var matches []string = routePattern.FindStringSubmatch(line)
	if matches == nil {
		return
	}
	method, path, action, fixedArgs = matches[1], matches[4], matches[5], matches[6]
	found = true
	return
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
		log.Println("revel/router: reverse router got invalid action ", action)
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
				log.Println("revel/router: reverse route missing route arg ", el[1:])
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
	log.Println("Failed to find reverse route:", action, argValues)
	return nil
}
