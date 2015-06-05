package controller

import (
	"github.com/gabstv/goboots"
)

type AppController struct {
	goboots.Controller
}

func (c *AppController) Init() {
	c.Controller.Init()
	// initialization logic here
}

func (c *AppController) PreFilter(in *goboots.In) *goboots.Out {
	// runs before the routed function

	// return anything else to stop the execution
	// you could also return something like
	// return in.OutputString("under maintenance")
	return in.Continue()
}
