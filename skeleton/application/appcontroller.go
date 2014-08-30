package application

import (
	"github.com/gabstv/goboots"
	"net/http"
)

type AppController struct {
	goboots.Controller
}

//TODO: remove the need to have this function (and change it to index!)
func (p *AppController) Run(w http.ResponseWriter, r *http.Request, params []string) interface{} {
	return nil
}

func (c *AppController) PreFilter(w http.ResponseWriter, r *http.Request, params []string) interface{} {
	return nil
}
