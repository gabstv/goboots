package controller

import (
	"github.com/gabstv/goboots"
)

func init() {
	////REGISTER CONTROLLERS INIT
	goboots.RegisterControllerGlobal(&Home{})
	////REGISTER CONTROLLERS END
}
