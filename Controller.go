package goboots

import (
	"bytes"
	"net/http"
	"reflect"
	"text/template"
	//"time"
	"encoding/binary"
)

type IController interface {
	//Setup(app *App)
	Init(app *App)
	Run(w http.ResponseWriter, r *http.Request, params []string) interface{}
	PreFilter(w http.ResponseWriter, r *http.Request, params []string) interface{}
	// rendering
	Render(w http.ResponseWriter, r *http.Request, content interface{})
	_registerMethod(name string, method reflect.Value)
	_getMethod(name string) (reflect.Value, bool)
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
	customMethods map[string]reflect.Value
}

func (c *Controller) Init(app *App) {
	c.App = app
	c.PageTitle = ""
	c.Layout = "default"
	c.ContentType = "text/html; charset=utf-8"
}

func (c *Controller) Run(w http.ResponseWriter, r *http.Request, params []string) interface{} {
	// www.example.com/params[0]/params[1]/(...)params[n]
	content := &PageContent{
		Title:   c.App.Config.Name + " - " + c.PageTitle,
		Content: "content goes here",
	}
	return content
}

func (c *Controller) PreFilter(w http.ResponseWriter, r *http.Request, params []string) interface{} {
	return nil
}

func (c *Controller) Render(w http.ResponseWriter, r *http.Request, content interface{}) {
	c.render(w, r, content, "")
}

func (c *Controller) RenderWithLayout(w http.ResponseWriter, r *http.Request, content interface{}, customLayout string) {
	c.render(w, r, content, customLayout)
}

func (c *Controller) render(w http.ResponseWriter, r *http.Request, content interface{}, customLayout string) {
	if len(c.Layout) && len(customLayout) == 0 {
		// no layout defined!
		var buff bytes.Buffer
		binary.Write(&buff, binary.LittleEndian, content)
		w.Write(buff.Bytes())
		return
	}
	layoutName := c.Layout
	if len(customLayout) > 0 {
		layoutName = customLayout
	}
	var layoutTpl *template.Template
	if len(c.App.Config.DefaultLanguage) > 0 {
		layoutTpl = c.App.GetLocalizedLayout(layoutName, w, r)
	} else {
		layoutTpl = c.App.GetLayout(layoutName)
	}
	layoutTpl.Execute(w, content)
}

// RenderFromCache renders an entire page to the app's bytecache
func (c *Controller) RenderToCache(w http.ResponseWriter, r *http.Request, name string, content interface{}) {
	if len(c.Layout) == 0 {
		// no layout defined!
		var buff bytes.Buffer
		binary.Write(&buff, binary.LittleEndian, content)
		c.App.ByteCaches.SetCache(name, buff.Bytes())
		return
	}
	var layoutTpl *template.Template
	if len(c.App.Config.DefaultLanguage) > 0 {
		layoutTpl = c.App.GetLocalizedLayout(c.Layout, w, r)
	} else {
		layoutTpl = c.App.GetLayout(c.Layout)
	}
	var buffer bytes.Buffer
	err := layoutTpl.Execute(&buffer, content)
	__panic(err)
	c.App.ByteCaches.SetCache(name, buffer.Bytes())
}

// RenderFromCache renders an entire page from the app's bytecache
func (c *Controller) RenderFromCache(w http.ResponseWriter, r *http.Request, name string) interface{} {
	w.Write(c.App.ByteCaches.GetCache(name).Content)
	return nil
}

func (c *Controller) ParseContent(w http.ResponseWriter, r *http.Request, locTemplatePath string, content interface{}) string {
	var tpl *template.Template
	if len(c.App.Config.DefaultLanguage) > 0 {
		tpl = c.App.GetLocalizedViewTemplate(locTemplatePath, w, r)
	} else {
		tpl = c.App.GetViewTemplate(locTemplatePath)
	}

	//TODO: handle error if template is nil
	var buffer bytes.Buffer
	err := tpl.Execute(&buffer, content)
	__panic(err)
	return buffer.String()
}

func (c *Controller) Redirect(w http.ResponseWriter, path string, title string, message string) interface{} {
	tpl := c.App.GetLayout("redirect")
	//TODO: handle error if template is nil
	content := &RedirectPageContent{
		Title:   title,
		Message: message,
		Path:    path,
	}
	tpl.Execute(w, content)
	return nil
}

func (c *Controller) PageError(w http.ResponseWriter, errorTitle string, errorMessage string, extraHTMLContent string) interface{} {
	tpl := c.App.GetLayout("error")
	//TODO: handle error if template is nil
	content := &ErrorPageContent{
		Title:        c.App.Config.Name + " - Error: " + errorTitle,
		ErrorTitle:   errorTitle,
		ErrorMessage: errorMessage,
		Content:      extraHTMLContent,
	}
	tpl.Execute(w, content)
	return nil
}

func (c *Controller) _registerMethod(name string, method reflect.Value) {
	if c.customMethods == nil {
		c.customMethods = make(map[string]reflect.Value, 0)
	}
	c.customMethods[name] = method
}
func (c *Controller) _getMethod(name string) (reflect.Value, bool) {
	if c.customMethods == nil {
		return reflect.Value{}, false
	}
	val, ok := c.customMethods[name]
	return val, ok
}
