package goboots

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJadeFix(t *testing.T) {
	tpls := []string{
		`<!DOCTYPE html>
<html lang="{{.T 'lc'}}">
    <head>
        <meta charset="utf-8">
        <title>{{ .Title }}</title>
    </head>
    <body>
        <p>hello</p>
    </body>
</html>`,
	}
	expected := []string{
		`<!DOCTYPE html>
<html lang="{{.T "lc"}}">
    <head>
        <meta charset="utf-8">
        <title>{{ .Title }}</title>
    </head>
    <body>
        <p>hello</p>
    </body>
</html>`,
	}
	for k := range tpls {
		assert.Equal(t, expected[k], tempJadeFix(tpls[k], "{{", "}}"))
	}
}
