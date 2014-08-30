package goboots

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"github.com/gabstv/i18ngo"
	"io"
	"net/http"
	"text/template"
)

const (
	outJSON         = 1
	outXML          = 2
	outTemplate     = 3
	outTemplateSolo = 4
	outString       = 5
	outBytes        = 6
)

type In struct {
	R             *http.Request
	W             http.ResponseWriter
	URLParts      []string
	session       *Session
	Content       InContent
	LayoutContent InContent
	App           *App
	Controller    IController
	LangCode      string
	GlobalTitle   string
	closers       []io.Closer
}

func (in *In) closeall() {
	if in.closers == nil {
		return
	}
	for _, v := range in.closers {
		v.Close()
	}
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

func (c *InContent) Get2(key string) (val interface{}, ok bool) {
	c.init()
	val, ok = c.vals[key]
	return
}

func (c *InContent) GetString2(key string) (val string, ok bool) {
	c.init()
	var rval interface{}
	rval, ok = c.vals[key]
	if ok {
		val = rval.(string)
	}
	return
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

// Translates text to the user language (if available)
func (in *In) T(format string, v ...interface{}) string {
	return i18ngo.TL(in.LangCode, format, v...)
}

func (in *In) Session() *Session {
	if in.session == nil {
		in.session = GetSession(in.W, in.R)
	}
	return in.session
}

func (in *In) OutputTpl(tplPath string) *Out {
	return in.outputTpl(tplPath, "")
}

func (in *In) outputTpl(tplPath, customLayout string) *Out {
	o := &Out{}
	in.LayoutContent.Set("Content", in.OutputSoloTpl(tplPath).String())
	if v, ok := in.LayoutContent.GetString2("Title"); ok {
		in.LayoutContent.Set("Title", in.GlobalTitle+v)
	} else {
		in.LayoutContent.Set("Title", in.GlobalTitle+in.T(in.Controller.GetPageTitle()))
	}
	o.kind = outTemplate
	o.contentObj = in.LayoutContent.Set("Flash", in.Session().Flash.All()).All()
	ln := in.Controller.GetLayoutName()
	if len(ln) == 0 && len(customLayout) == 0 {
		// no layout defined!
		o.kind = outBytes
		var buff bytes.Buffer
		binary.Write(&buff, binary.BigEndian, o.contentObj)
		o.contentBytes = buff.Bytes()
		o.contentObj = nil
		return o
	}
	if len(customLayout) > 0 {
		ln = customLayout
	}
	var layoutTpl *template.Template
	if len(in.App.Config.DefaultLanguage) > 0 {
		layoutTpl = in.App.GetLocalizedLayout(ln, in.W, in.R)
	} else {
		layoutTpl = in.App.GetLayout(ln)
	}
	o.tpl = layoutTpl
	return o
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

func (in *In) OutputJSON(jobj interface{}) *Out {
	o := &Out{}
	o.kind = outJSON
	o.contentObj = jobj
	return o
}

func (in *In) OutputXML(xobj interface{}) *Out {
	o := &Out{}
	o.kind = outXML
	o.contentObj = xobj
	return o
}

func (in *In) OutputString(str string) *Out {
	o := &Out{}
	o.kind = outString
	o.contentStr = str
	return o
}

type Out struct {
	kind         int
	contentObj   interface{}
	contentStr   string
	contentBytes []byte
	tpl          *template.Template
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
		w.Header().Set("Content-Type", "application/json; charset=utf8")
		w.Write(o.mustb(json.Marshal(o.contentObj)))
	case outXML:
		w.Write(o.mustb(xml.Marshal(o.contentObj)))
	case outTemplateSolo, outTemplate:
		o.tpl.Execute(w, o.contentObj)
	case outString:
		w.Write([]byte(o.contentStr))
	case outBytes:
		w.Write(o.contentBytes)
	}
}

func (o *Out) String() string {
	switch o.kind {
	case outJSON:
		return string(o.mustb(json.Marshal(o.contentObj)))
	case outXML:
		return string(o.mustb(xml.Marshal(o.contentObj)))
	case outTemplateSolo, outTemplate:
		var buffer bytes.Buffer
		o.tpl.Execute(&buffer, o.contentObj)
		return buffer.String()
	case outString:
		return o.contentStr
	case outBytes:
		return string(o.contentBytes)
	}
	return ""
}
