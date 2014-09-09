package goboots

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"github.com/gabstv/i18ngo"
	"io"
	"log"
	"net/http"
	"reflect"
	"text/template"
)

const (
	outPre          = 0
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
	Content       *InContent
	LayoutContent *InContent
	App           *App
	Controller    IController
	LangCode      string
	GlobalTitle   string
	closers       []io.Closer
}

// New clones a new In but without the content.
// Useful to render separate parts
func (in *In) New() *In {
	in2 := &In{}
	in2.R = in.R
	in2.W = in.W
	in2.URLParts = in.URLParts
	in2.session = in.session
	in2.Content = &InContent{}
	in2.LayoutContent = &InContent{}
	in2.App = in.App
	in2.Controller = in.Controller
	in2.LangCode = in.LangCode
	in2.GlobalTitle = in.GlobalTitle
	return in2
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

func (c *InContent) Merge(v interface{}) *InContent {
	if v == nil {
		return c
	}
	c.init()
	vtype := reflect.TypeOf(v)
	switch vtype.Kind() {
	case reflect.Map, reflect.Struct:
		// PASS
	case reflect.Ptr:
		if vtype.Elem().Kind() != reflect.Struct {
			return c
		}
	default:
		// tried to merge an invalid type
		return c
	}
	vl := reflect.ValueOf(v)
	if vtype.Kind() == reflect.Ptr {
		vl = vl.Elem()
		vtype = vtype.Elem()
		log.Println(vl.Kind().String())
	}
	if vtype.Kind() == reflect.Map {
		// merge mappy things
		if vtype.Key().Kind() != reflect.String {
			log.Println("MAP KEY IS NOT A STRING!")
			return c
		}
		keys := vl.MapKeys()
		for _, key := range keys {
			val := vl.MapIndex(key)
			c.vals[key.String()] = val.Interface()
		}
	} else {
		// merge structy things
		len0 := vl.NumField()
		for i := 0; i < len0; i++ {
			field := vl.Field(i)
			c.vals[vtype.Field(i).Name] = field.Interface()
		}
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

func (c *InContent) GetInt2(key string) (val int, ok bool) {
	c.init()
	var rval interface{}
	rval, ok = c.vals[key]
	if ok {
		val, _ = rval.(int)
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

func (in *In) OutputLayTpl(layout, tplPath string) *Out {
	return in.outputTpl(tplPath, layout)
}

func (in *In) OutputLay(layout string) *Out {
	return in.outputTpl("", layout)
}

func (in *In) outputTpl(tplPath, customLayout string) *Out {
	o := &Out{}
	if len(tplPath) > 0 {
		in.LayoutContent.Set("Content", in.OutputSoloTpl(tplPath).String())
	}
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
	if len(in.App.Config.DefaultLanguage) > 0 && in.R != nil && in.W != nil {
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
	if in.R != nil && in.W != nil {
		o.contentObj = in.Content.Set("Flash", in.Session().Flash.All()).All()
	} else {
		o.contentObj = in.Content.All()
	}
	var tpl *template.Template
	if len(in.App.Config.DefaultLanguage) > 0 && in.R != nil && in.W != nil {
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

func (in *In) OutputBytes(b []byte) *Out {
	o := &Out{}
	o.kind = outBytes
	o.contentBytes = b
	return o
}

func (in *In) Continue() *Out {
	o := &Out{}
	o.kind = outPre
	return o
}

func (in *In) SetNoCache() *In {
	in.W.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	in.W.Header().Set("Pragma", "no-cache")
	in.W.Header().Set("Expires", "0")
	return in
}

type Out struct {
	kind         int
	contentObj   interface{}
	contentStr   string
	contentBytes []byte
	tpl          *template.Template
}

func (o *Out) IsContinue() bool {
	return o.kind == outPre
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
		if len(w.Header().Get("Content-Type")) < 1 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		w.Write(o.mustb(json.Marshal(o.contentObj)))
	case outXML:
		if len(w.Header().Get("Content-Type")) < 1 {
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		}
		w.Write(o.mustb(xml.Marshal(o.contentObj)))
	case outTemplateSolo, outTemplate:
		if len(w.Header().Get("Content-Type")) < 1 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		o.tpl.Execute(w, o.contentObj)
	case outString:
		if len(w.Header().Get("Content-Type")) < 1 {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
		w.Write([]byte(o.contentStr))
	case outBytes:
		if len(w.Header().Get("Content-Type")) < 1 {
			w.Header().Set("Content-Type", "application/octet-stream; charset=utf-8")
		}
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
