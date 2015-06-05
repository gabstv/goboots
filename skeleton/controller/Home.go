package controller

import (
	"github.com/gabstv/goboots"
)

type Home struct {
	App // super
}

func (c *Home) Index(in *goboots.In) *goboots.Out {
	in.Content.Set("greetings", "Hello, World!")
	return in.OutputTpl("home/index.html")
}
