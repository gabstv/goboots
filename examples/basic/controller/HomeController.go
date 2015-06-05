package controller

import (
	"github.com/gabstv/goboots"
)

type HomeController struct {
	AppController
}

func (c *HomeController) Init(app *goboots.App) {
	c.AppController.Init(app)
}

func (c *HomeController) PreFilter(in *goboots.In) *goboots.Out {
	// runs before the routed function
	if pfResult := c.AppController.PreFilter(in); pfResult == nil || !pfResult.IsContinue() {
		return pfResult
	}

	// return anything else to stop the execution
	// you could also return something like
	// return in.OutputString("under maintenance")
	return in.Continue()
}

func (c *HomeController) Index(in *goboots.In) *goboots.Out {
	return in.OutputTpl("home/index.tpl")
}
