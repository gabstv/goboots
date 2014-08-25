package goboots

import (
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
)

type In struct {
	R        *http.Request
	W        http.ResponseWriter
	URLParts []string
}

type Out struct {
	kind       int
	jsonObj    interface{}
	xmlObj     interface{}
	contentObj interface{}
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
		w.Write(o.mustb(json.Marshal(o.jsonObj)))
	case outXML:
		w.Write(o.mustb(xml.Marshal(o.xmlObj)))
	case outTemplateSolo:
		o.tpl.Execute(w, o.contentObj)
	}
}
