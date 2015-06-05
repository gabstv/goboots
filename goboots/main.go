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

const usageTpl = `usage: goboots [command] [arguments]

Commands:
{{range .}}
    {{.Name | printf "%-9s"}} {{.Short}}{{end}}

Use "goboots help [command]" for more information.
`

var commands = []*Command{cmdNew}

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
	fmt.Fprintln(os.Stdout, header)
	flag.Usage = func() { usage(1) }
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 || args[0] == "help" {
		if len(args) == 1 {
			usage(0)
		}
		if len(args) > 1 {
			for _, cmd := range commands {
				if cmd.Name() == args[1] {
					tmpl(os.Stdout, helpTemplate, cmd)
					return
				}
			}
		}
		usage(2)
	}

	// Commands use panic to abort execution when something goes wrong.
	// Panics are logged at the point of error.  Ignore those.
	defer func() {
		if err := recover(); err != nil {
			if _, ok := err.(LoggedError); !ok {
				// This panic was not expected / logged.
				panic(err)
			}
			os.Exit(1)
		}
	}()

	for _, cmd := range commands {
		if cmd.Name() == args[0] {
			cmd.Run(args[1:])
			return
		}
	}

	errorf("unknown command %q\nRun 'goboots help' for usage.\n", args[0])
}

var helpTemplate = `usage: goboots {{.UsageLine}}
{{.Long}}
`

func usage(exitCode int) {
	tmpl(os.Stderr, usageTpl, commands)
	os.Exit(exitCode)
}

func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	template.Must(t.Parse(text))
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}

func errorf(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	panic(LoggedError{}) // Panic instead of os.Exit so that deferred will run.
}

// Use a wrapper to differentiate logged panics from unexpected ones.
type LoggedError struct{ error }
