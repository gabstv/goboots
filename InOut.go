package goboots

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"text/template"
)

const (
	outJSON         = 1
	outXML          = 2
	outTemplate     = 3
	outTemplateSolo = 4
	outString       = 5
)

type In struct {
	R        *http.Request
	W        http.ResponseWriter
	URLParts []string
	session  *Session
	Content  InContent
	App      *App
}

type InContent struct {
	vals map[string]interface{}
}

func (c *InContent) init() {
	if c.vals == nil {
		c.vals = make(map[string]interface{})
	}
}

func (c *InContent) Merge(v map[string]interface{}) *InContent {
	c.init()
	for kk, vv := range v {
		c.vals[kk] = vv
	}
	return c
}

func (c *InContent) Set(key string, val interface{}) *InContent {
	c.init()
	c.vals[key] = val
	return c
}

func (c *InContent) All() map[string]interface{} {
	c.init()
	return c.vals
}

func (in *In) Session() *Session {
	if in.session == nil {
		in.session = GetSession(in.W, in.R)
	}
	return in.session
}

func (in *In) OutputSoloTpl(tplPath string) *Out {
	o := &Out{}
	o.kind = outTemplateSolo
	o.contentObj = in.Content.Set("Flash", in.Session().Flash.All()).All()
	var tpl *template.Template
	if len(in.App.Config.DefaultLanguage) > 0 {
		tpl = in.App.GetLocalizedViewTemplate(tplPath, in.W, in.R)
	} else {
		tpl = in.App.GetViewTemplate(tplPath)
	}
	o.tpl = tpl
	return o
}

type Out struct {
	kind       int
	contentObj interface{}
	contentStr string
	tpl        *template.Template
	//in         *In
	//ctrlr      *Controller
}

func (o *Out) mustb(b []byte, err error) []byte {
	if err != nil {
		panic(err)
		return b
	}
	return b
}

func (o *Out) render(w http.ResponseWriter) {
	switch o.kind {
	case outJSON:
		w.Write(o.mustb(json.Marshal(o.contentObj)))
	case outXML:
		w.Write(o.mustb(xml.Marshal(o.contentObj)))
	case outTemplateSolo:
		o.tpl.Execute(w, o.contentObj)
	case outString:
		w.Write([]byte(o.contentStr))
	}
}

func (o *Out) String() string {
	switch o.kind {
	case outJSON:
		return string(o.mustb(json.Marshal(o.contentObj)))
	case outXML:
		return string(o.mustb(xml.Marshal(o.contentObj)))
	case outTemplateSolo:
		var buffer bytes.Buffer
		o.tpl.Execute(&buffer, o.contentObj)
		return buffer.String()
	case outString:
		return o.contentStr
	}
	return ""
}
