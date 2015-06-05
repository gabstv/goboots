package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

// copyDir copies a directory tree over to a new directory.  Any files ending in
// ".skel" are treated as a Go template and rendered using the given data.
// Additionally, the trailing ".skel" is stripped from the file name.
// Also, dot files and dot directories are skipped.
func mustCopyDir(destDir, srcDir string, data map[string]interface{}) error {
	var fullSrcDir string
	// Handle symlinked directories.
	f, err := os.Lstat(srcDir)
	if err == nil && f.Mode()&os.ModeSymlink == os.ModeSymlink {
		fullSrcDir, err = os.Readlink(srcDir)
		if err != nil {
			panic(err)
		}
	} else {
		fullSrcDir = srcDir
	}

	return filepath.Walk(fullSrcDir, func(srcPath string, info os.FileInfo, err error) error {
		// Get the relative path from the source base, and the corresponding path in
		// the dest directory.
		relSrcPath := strings.TrimLeft(srcPath[len(fullSrcDir):], string(os.PathSeparator))
		destPath := path.Join(destDir, relSrcPath)

		// Skip dot files and dot directories.
		if strings.HasPrefix(relSrcPath, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create a subdirectory if necessary.
		if info.IsDir() {
			err := os.MkdirAll(path.Join(destDir, relSrcPath), 0777)
			if !os.IsExist(err) {
				panicOnError(err, "Failed to create directory")
			}
			return nil
		}

		// If this file ends in ".skel", render it as a template.
		if strings.HasSuffix(relSrcPath, ".skel") {
			mustRenderTemplate(destPath[:len(destPath)-len(".skel")], srcPath, data)
			return nil
		}

		// Else, just copy it over.
		mustCopyFile(destPath, srcPath)
		return nil
	})
}

func mustCopyFile(destFilename, srcFilename string) {
	destFile, err := os.Create(destFilename)
	panicOnError(err, "Failed to create file "+destFilename)

	srcFile, err := os.Open(srcFilename)
	panicOnError(err, "Failed to open file "+srcFilename)

	_, err = io.Copy(destFile, srcFile)
	panicOnError(err,
		fmt.Sprintf("Failed to copy data from %s to %s", srcFile.Name(), destFile.Name()))

	err = destFile.Close()
	panicOnError(err, "Failed to close file "+destFile.Name())

	err = srcFile.Close()
	panicOnError(err, "Failed to close file "+srcFile.Name())
}

func mustRenderTemplate(destPath, srcPath string, data map[string]interface{}) {
	tmpl, err := template.ParseFiles(srcPath)
	panicOnError(err, "Failed to parse template "+srcPath)

	f, err := os.Create(destPath)
	panicOnError(err, "Failed to create "+destPath)

	err = tmpl.Execute(f, data)
	panicOnError(err, "Failed to render template "+srcPath)

	err = f.Close()
	panicOnError(err, "Failed to close "+f.Name())
}

const clist = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789[]|@!+=-()%$"

func newkey32() string {
	bs := make([]byte, 32)
	rand.Read(bs)
	buf := new(bytes.Buffer)
	for _, v := range bs {
		for int(v) >= len(clist) {
			v -= byte(len(clist))
		}
		buf.WriteByte(clist[int(v)])
	}
	return buf.String()
}

func panicOnError(err error, msg string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Abort: %s: %s\n", msg, err)
		panic(LoggedError{err})
	}
}
