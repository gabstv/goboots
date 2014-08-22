package goboots

import (
	"net/http"
)

type In struct {
	R        *http.Request
	W        http.ResponseWriter
	URLParts []string
}
