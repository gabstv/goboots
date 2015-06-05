package main

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
)

var cmdNew = &Command{
	UsageLine: "new [path] [skeleton]",
	Short:     "Create a skeleton Goboots App.",
	Long: `
Creates some basic files to get a new Goboots application running.

All the files will be created in the given import path. The final
element will also be the application name.
`,
}

func init() {
	cmdNew.Run = newApp
}

var (
	gopath  string
	gocmd   string
	srcRoot string

	importPath string
	appPath    string
	appName    string
	basePath   string
	skelln     string
)

func newApp(args []string) {
	print("RAN NEWAPP COMMAND!\n")
	//
	if len(args) < 1 {
		errorf("No import path given.\nRun 'goboots help new' for usage.\n")
	}
	if len(args) > 2 {
		errorf("Too many arguments provided.\nRun 'goboots help new' for usage.\n")
	}
	//
	p := "github.com/gabstv/goboots/skeleton/standard"
	if len(args) > 1 {
		p = args[1]
	}
	//
	initSysPaths()
	//
	initAppPaths(args[0])
	//
	initSkelPath(p)
	//
	copyFiles()
}

func initSysPaths() {
	// find go path
	gopath = build.Default.GOPATH
	if len(gopath) < 1 {
		errorf("Abort: GOPATH environment variable is not set. " +
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.")
	}
	// set go src path
	srcRoot = filepath.Join(filepath.SplitList(gopath)[0], "src")

	// find go executable
	var err error
	gocmd, err = exec.LookPath("go")
	if err != nil {
		errorf("Go executable not found in PATH.")
	}
}

func initAppPaths(p string) {
	var err error
	importPath = p
	if filepath.IsAbs(importPath) {
		errorf("Abort: '%s' looks like a directory.  Please provide a Go import path instead.", importPath)
	}
	_, err = build.Import(importPath, "", build.FindOnly)
	if err == nil {
		errorf("Abort: Import path %s already exists.\n", importPath)
	}
	appPath = filepath.Join(srcRoot, filepath.FromSlash(importPath))
	appName = filepath.Base(appPath)
	basePath = filepath.ToSlash(filepath.Dir(importPath))
	if basePath == "." {
		// we need to remove the a single '.' when
		// the app is in the $GOROOT/src directory
		basePath = ""
	} else {
		// we need to append a '/' when the app is
		// is a subdirectory such as $GOROOT/src/path/to/revelapp
		basePath += "/"
	}
}

func initSkelPath(path string) {
	skelln = path
	var err error
	_, err = build.Import(skelln, "", build.FindOnly)
	if err != nil {
		// Execute "go get <pkg>"
		getCmd := exec.Command(gocmd, "get", "-d", skelln)
		fmt.Println("Exec:", getCmd.Args)
		//getOutput, err := getCmd.CombinedOutput()
	}
	skelln = filepath.Join(srcRoot, skelln)
}

func copyFiles() {
	err := os.MkdirAll(appPath, 0744)
	if err != nil {
		panicOnError(err, "Could not create directories.")
	}
	mustCopyDir(appPath, skelln, map[string]interface{}{
		"AppName": appName,
		"Salt":    newkey32(),
	})
	// Dotfiles are skipped by mustCopyDir, so we have to explicitly copy the .gitignore.
	gitignore := ".gitignore"
	mustCopyFile(filepath.Join(appPath, gitignore), filepath.Join(skelln, gitignore))
}
