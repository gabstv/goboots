package poutil

import (
	"fmt"
	"sync"

	"github.com/gabstv/i18n"
	"github.com/gabstv/i18n/po"
)

type provider struct {
	sync.Mutex
	langs           map[string]*lnguage
	defaultLangCode string
}

func (p *provider) L(code string) i18n.Language {
	p.Lock()
	defer p.Unlock()
	if l := p.langs[code]; l != nil {
		return l
	}
	if l := p.langs[p.defaultLangCode]; l != nil {
		return l
	}
	return nil
}

func (p *provider) LanguageCodes() []string {
	lc := make([]string, len(p.langs))
	i := -1
	for k, _ := range p.langs {
		i++
		lc[i] = k
	}
	return lc
}

type lnguage struct {
	metadata po.Meta
	entries  map[string]string
	context  map[string]map[string]string
}

func (l *lnguage) Meta(key string) string {
	return l.metadata.Get(key)
}

func (l *lnguage) Ctx(ctx string) func(id string, v ...interface{}) string {
	return func(id string, v ...interface{}) string {
		if c := l.context[ctx]; c != nil {
			if v2, ok := c[id]; ok {
				return fmt.Sprintf(v2, v...)
			}
		}
		return fmt.Sprintf(id, v...)
	}
}

func (l *lnguage) T(id string, v ...interface{}) string {
	if v2, ok := l.entries[id]; ok {
		return fmt.Sprintf(v2, v...)
	}
	return fmt.Sprintf(id, v...)
}
