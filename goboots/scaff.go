package main

import (
	"fmt"
	//"go/build"
	"os"
	//"os/exec"
	"path/filepath"
)

var cmdScaff = &Command{
	UsageLine: "scaff [kind] [name]",
	Short:     "Create a scaffold of the selected type.",
	Long: `
Creates the essential starter code of the desired type.

Types Available:
	controller [c]
        'scaff controller Hello' | Creates the controller Hello
                                 | at controller/Hello.go
`,
}

func init() {
	cmdScaff.Run = newScaff
}

func newScaff(args []string) {
	initSysPaths()
	if len(args) < 2 {
		errorf("No kind and name given.\nRun 'goboots help scaff' for usage.\n")
	}
	if args[0] != "controller" && args[0] != "c" {
		errorf("Invalid kind.\nRun 'goboots help scaff' for usage.\n")
	}
	ppp := filepath.Join(srcRoot, filepath.FromSlash("github.com/gabstv/goboots/skeleton/helpers/ctrltemplate.tpl"))
	wd, _ := os.Getwd()
	pj := filepath.Join(wd, "controller")
	if _, err := os.Stat("controller"); os.IsNotExist(err) {
		pj = wd
	}
	mustRenderTemplate(filepath.Join(pj, args[1]+".go"), ppp, map[string]interface{}{
		"Name": args[1],
	})
	fmt.Println("Controller " + args[1] + " created.")
}
