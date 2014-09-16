package goboots

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	//"log"
	"path"
	"strings"
)

//TODO: !IMPORTANT check for a way to avoid infinite loop cycles!

// {{partial "NAME" /path/to/template}}

//
func (a *App) parseTemplateIncludeDeps(lwd string, template []byte) ([]byte, error) {
	return parseTemplateIncludeDeps(a.basePath, a.Config.ViewsFolderPath, lwd, template)
}

func parseTemplateIncludeDeps(basePath, viewsFolderPath, lwd string, template []byte) ([]byte, error) {
	fb := new(bytes.Buffer)
	wb := new(bytes.Buffer)
	stackb := new(bytes.Buffer)
	tlen := len(template)
	partialb := []byte("{{partial ")
	skipb := 0
	for k, v := range template {
		if skipb > 0 {
			skipb--
			continue
		}
		//log.Println(string(v), k, tlen-13)
		if v == '{' && k < tlen-13 {
			//log.Println("possible match!")
			//log.Println("`"+string(template[k:k+10])+"`", "`"+string(partialb)+"`")
			if bytes.Compare(template[k:k+10], partialb) == 0 {
				// we got a match
				k2 := k + 10
				//log.Println("match at", k2, template[k2])
				for {
					if k2 >= tlen {
						// EOF!
						return nil, errors.New("EOF error while trying to get partial template `" + wb.String() + "`")
					}
					if template[k2] == '}' {
						// try to end it
						if k2+1 >= tlen {
							return nil, errors.New("EOF error while trying to get partial template `" + wb.String() + "`")
						}
						if template[k2+1] != '}' {
							return nil, errors.New("parse error while trying to get partial template `" + wb.String() + "`")
						}
						// finalize
						sptr := strings.TrimSpace(wb.String())
						//log.Println("sptr", sptr)
						name := ""
						lpath := ""
						if sptr[0] == '"' {
							// not inline
							name = sptr[:strings.LastIndex(sptr, "\"")]
							lpath = strings.TrimSpace(sptr[strings.LastIndex(sptr, "\"")+1:])
						} else {
							// inline
							lpath = sptr
						}
						//log.Println("name", name)
						//log.Println("lpath", lpath)
						if strings.HasPrefix(lpath, "./") {
							lpath = path.Join(lwd, lpath[1:])
						} else if strings.HasPrefix(lpath, "../") {
							lpath = path.Join(lwd, lpath)
						} else if strings.HasPrefix(lpath, "/") {
							lpath = path.Join(basePath, viewsFolderPath, lpath)
						} else {
							lpath = path.Join(lwd, lpath)
						}
						//TODO: test if this really prevents off
						if !strings.HasPrefix(lpath, basePath) && !strings.HasPrefix(lpath, viewsFolderPath) {
							return nil, errors.New("partial template path `" + lpath + "` outside of app path `" + basePath + "`!")
						}
						// get raw template
						childbits, err := ioutil.ReadFile(lpath)
						if err != nil {
							return nil, errors.New("partial template error (io): " + err.Error())
						}
						childp, _ := path.Split(lpath)
						childbits, err = parseTemplateIncludeDeps(basePath, viewsFolderPath, childp, childbits)
						if err != nil {
							return nil, errors.New("partial template error: " + err.Error())
						}
						if len(name) > 0 {
							// not inline
							stackb.WriteString("{{define " + name + "}}")
							fb.Write(childbits)
							stackb.WriteString("{{end}}")
						} else {
							stackb.Write(childbits)
						}
						skipb = k2 + 1 - k
						wb.Reset()
						break
					} else {
						wb.WriteByte(template[k2])
					}
					k2++
				}
			} else {
				fb.WriteByte(v)
			}
		} else {
			fb.WriteByte(v)
		}
	}
	_, ferr := io.Copy(stackb, fb)
	if ferr != nil {
		return nil, errors.New("io.Copy error while parsing template partials on path `" + lwd + "`: " + ferr.Error())
	}
	return stackb.Bytes(), nil
}
