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
}

func (in *In) Session() *Session {
	if in.session == nil {
		in.session = GetSession(in.W, in.R)
	}
	return in.session
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
