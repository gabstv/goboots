package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
)

const header = `
######################################
##         .-'\                     ##
##      .-'  '/\                    ##
##    -'      '/\                   ##
##             '/\                  ##
##    \         '/\                 ##
##     \    _-   '/\       _.--.    ##
##      \    _-   '/'-..--\     )   ##
##       \    _-   ',','  /    ,')  ##
##        '-_   -   ' -- ~   ,','   ##
##         '-              ,','     ##
##          \,--.    ____==-~       ##
##           \   \_-~\              ##
##            '_-~_.-'              ##
##             \-~                  ##
##          g o b o o t s           ##
## http://gabstv.github.com/goboots ##
##                                  ##
######################################
`

const usageTpl = `usage: goboots commang [arguments]

Commands:
[[]]

Use "goboots help [command]" for more information.
`

var commands = []*Command{}

type Command struct {
	Run                    func(args []string)
	UsageLine, Short, Long string
}

func (cmd *Command) Name() string {
	name := cmd.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func main() {
	fmt.Println(os.Stdout, header)
	flag.Usage = usage
}

func usage() {
	tmpl(os.Stderr, usageTpl, commands)
	os.Exit(2)
}

func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	template.Must(t.Parse(text))
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}
