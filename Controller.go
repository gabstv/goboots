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
	GetPageTitle() string
	GetLayoutName() string
	Init(app *App)
	Run(w http.ResponseWriter, r *http.Request, params []string) interface{}
	PreFilter(w http.ResponseWriter, r *http.Request, params []string) interface{}
	// rendering
	Render(w http.ResponseWriter, r *http.Request, content interface{})
	RenderNew(w http.ResponseWriter, out *Out)
	registerMethod(name string, method reflect.Value, inKind, outKind int)
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
	Val           reflect.Value
	MethodKindIn  int
	MethodKindOut int
}

func (c *Controller) GetPageTitle() string {
	return c.PageTitle
}

func (c *Controller) GetLayoutName() string {
	return c.Layout
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

func (c *Controller) RenderNew(w http.ResponseWriter, out *Out) {
	if out != nil {
		out.render(w)
	}
}

//TODO: remove Output functions (they are inside the In struct)
func (c *Controller) OutputSoloTpl(in *In, tplPath string, content interface{}) *Out {
	o := &Out{}
	o.kind = outTemplateSolo
	o.contentObj = content
	var tpl *template.Template
	if len(c.App.Config.DefaultLanguage) > 0 {
		tpl = c.App.GetLocalizedViewTemplate(tplPath, in.W, in.R)
	} else {
		tpl = c.App.GetViewTemplate(tplPath)
	}
	o.tpl = tpl
	return o
}

func (c *Controller) OutputJSON(jobj interface{}) *Out {
	o := &Out{}
	o.kind = outJSON
	o.contentObj = jobj
	return o
}

func (c *Controller) OutputXML(xobj interface{}) *Out {
	o := &Out{}
	o.kind = outXML
	o.contentObj = xobj
	return o
}

func (c *Controller) OutputString(str string) *Out {
	o := &Out{}
	o.kind = outString
	o.contentStr = str
	return o
}

func (c *Controller) render(w http.ResponseWriter, r *http.Request, content interface{}, customLayout string) {
	if len(c.Layout) == 0 && len(customLayout) == 0 {
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
	if len(c.App.Config.DefaultLanguage) > 0 && w != nil && r != nil {
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

func (c *Controller) registerMethod(name string, method reflect.Value, inKind, outKind int) {
	if c.customMethods == nil {
		c.customMethods = make(map[string]controllerMethod, 0)
	}
	c.customMethods[name] = controllerMethod{method, inKind, outKind}
}
func (c *Controller) getMethod(name string) (controllerMethod, bool) {
	if c.customMethods == nil {
		return controllerMethod{}, false
	}
	val, ok := c.customMethods[name]
	return val, ok
}
