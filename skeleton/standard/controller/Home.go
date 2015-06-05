package controller

import (
	"fmt"
	"github.com/gabstv/goboots"
)

type Home struct {
	App // parent/super controller
}

func (c *Home) Init() {
	c.App.Init()
	// add initialization code below
}

func (c *Home) PreFilter(in *goboots.In) *goboots.Out {
	// Abort execution if the parent controller didn't send in.Continue()
	if r := c.App.PreFilter(in); r == nil || !r.IsContinue() {
		return r
	}
	return in.Continue()
}

func (c *Home) Index(in *goboots.In) *goboots.Out {
	in.Content.Set("greetings", "Hello, World!")
	return in.OutputTpl("home/index.tpl")
}

func (c *Home) Count(in *goboots.In) *goboots.Out {

	count := in.Session().GetInt32D("count", 0)
	count++
	in.Session().Data["count"] = count
	in.Session().Flush()

	return in.OutputString(fmt.Sprintf("counter: %v", count))
}

func init() {
	//Globally register this controller
	goboots.RegisterControllerGlobal(&Home{})
}
