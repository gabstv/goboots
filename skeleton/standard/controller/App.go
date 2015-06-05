package controller

import (
	"github.com/gabstv/goboots"
)

// All controllers are a subset of the App Controller
// [goboots:ignore]
type App struct {
	goboots.Controller
}

func (c *App) Init() {
	c.Controller.Init()
	// add initialization code here
}

func (c *App) PreFilter(in *goboots.In) *goboots.Out {
	return in.Continue()
}
