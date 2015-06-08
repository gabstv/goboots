package goboots

import (
	"reflect"
)

type IController interface {
	GetApp() *App
	SetApp(app *App)
	GetPageTitle() string
	GetLayoutName() string
	Init()
	PreFilter(in *In) *Out
	registerMethod(name string, method reflect.Value)
	getMethod(name string) (controllerMethod, bool)
}

type PageContent struct {
	Title       string
	Content     string
	UserContent string
	HeadInclude string
	FootInclude string
}

type RedirectPageContent struct {
	Title   string
	Message string
	Path    string
}

type ErrorPageContent struct {
	Title        string
	ErrorTitle   string
	ErrorMessage string
	Content      string
}

type Controller struct {
	App           *App
	Path          string
	Name          string
	LayoutName    string
	Action        string
	LayoutAction  string
	Params        []string
	PageTitle     string
	Layout        string
	ContentType   string
	customMethods map[string]controllerMethod
}

const (
	controllerMethodKindLegacy = iota
	controllerMethodKindNew
)

type controllerMethod struct {
	Val reflect.Value
}

func (c *Controller) GetPageTitle() string {
	return c.PageTitle
}

func (c *Controller) GetLayoutName() string {
	return c.Layout
}

func (c *Controller) GetApp() *App {
	return c.App
}

func (c *Controller) SetApp(app *App) {
	c.App = app
}

func (c *Controller) Init() {
	c.PageTitle = ""
	c.Layout = "default"
	c.ContentType = "text/html; charset=utf-8"
}

func (c *Controller) PreFilter(in *In) *Out {
	return in.Continue()
}

func (c *Controller) registerMethod(name string, method reflect.Value) {
	if c.customMethods == nil {
		c.customMethods = make(map[string]controllerMethod, 0)
	}
	c.customMethods[name] = controllerMethod{method}
}
func (c *Controller) getMethod(name string) (controllerMethod, bool) {
	if c.customMethods == nil {
		return controllerMethod{}, false
	}
	val, ok := c.customMethods[name]
	return val, ok
}
