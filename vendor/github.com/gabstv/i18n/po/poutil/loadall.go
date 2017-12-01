package poutil

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabstv/i18n"
	"github.com/gabstv/i18n/po"

	"golang.org/x/text/language"
)

func LoadAll(root, defaultLangCode string) (i18n.Provider, error) {
	return LoadAllFs(root, defaultLangCode, i18n.OsFS())
}

func LoadAllFs(root, defaultLangCode string, filesystem i18n.Fs) (i18n.Provider, error) {
	provd := provider{
		langs:           make(map[string]*lnguage),
		defaultLangCode: defaultLangCode,
	}
	if er2 := i18n.Walk(filesystem, root, func(path string, info os.FileInfo, err error) error {
		//
		if ext := filepath.Ext(path); ext != ".po" && ext != ".pot" && ext != ".txt" {
			return nil
		}
		var bf0 bytes.Buffer
		f0, err := filesystem.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(&bf0, f0)
		if err != nil {
			return err
		}
		f0.Close()
		b := bf0.Bytes()
		bf0.Reset()
		var f po.File
		if po.Unmarshal(b, &f) == nil {
			lc := getLanguageCode(path, &f)
			l := provd.langs[lc]
			if l == nil {
				l = &lnguage{
					metadata: po.Meta{},
					entries:  make(map[string]string),
					context:  make(map[string]map[string]string),
				}
			}
			// merge
			for k, v := range f.Entries {
				l.entries[k] = v
			}
			for ctxn, ctxv := range f.Context {
				if l.context[ctxn] == nil {
					l.context[ctxn] = ctxv
				} else {
					mm := l.context[ctxn]
					for kkk, vvv := range ctxv {
						mm[kkk] = vvv
					}
					l.context[ctxn] = mm
				}
			}
			for k, v := range f.Metadata {
				l.metadata[k] = v
			}
			provd.langs[lc] = l
		}
		return nil
		//
	}); er2 != nil {
		return nil, er2
	}

	return &provd, nil
}

func getLanguageCode(path string, file *po.File) string {
	if file != nil {
		if c := file.Metadata.Get("Language-Code"); c != "" {
			return c
		}
	}
	paths := strings.Split(path, string(os.PathSeparator))
	for i := len(paths) - 2; i >= 0; i-- {
		if _, err := language.Parse(paths[i]); err == nil {
			return paths[i]
		}
	}
	return ""
}
