package controller

import (
	"github.com/gabstv/goboots"
)

type {{.Name}} struct {
	App // parent/super controller
}


// Init is ran once when the App starts
func (c *{{.Name}}) Init() {
	c.App.Init()
	// add initialization code below
}

// This is ran before any controller method that wll be routed
func (c *{{.Name}}) PreFilter(in *goboots.In) *goboots.Out {
	// Abort execution if the parent controller didn't send in.Continue()
	if r := c.App.PreFilter(in); r == nil || !r.IsContinue() {
		return r
	}
	// Add own logic below
	return in.Continue()
}

// {{.Name}} methods

// {{.Name}}.Index
func (c *{{.Name}}) Index(in *goboots.In) *goboots.Out {
	return in.OutputString("Hello!")
}

// REGISTER

func init() {
	//Globally register this controller
	goboots.RegisterControllerGlobal(&{{.Name}}{})
}