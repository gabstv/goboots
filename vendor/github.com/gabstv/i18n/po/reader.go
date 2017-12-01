package po

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type state int

const (
	stateInit        state = 0
	stateReadId      state = 1
	stateReadValue   state = 2
	stateReadContext state = 3
)

type Reader struct {
	m sync.Mutex
	//
	entries        map[string]string
	contextEntries map[string]map[string]string
	rawMetadata    string
	//
	last struct {
		key         bytes.Buffer
		value       bytes.Buffer
		line        bytes.Buffer
		withContext bool
		context     bytes.Buffer
		n           int
		state       state
	}
	//
	Strict bool
}

func (r *Reader) Read(p []byte) (n int, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	if p == nil || len(p) < 1 {
		return 0, nil
	}
	for pos, char := range string(p) {
		n = pos
		r.last.line.WriteRune(char)
		if char == '\n' {
			if e := r.readLastLine(); e != nil {
				err = e
				return
			}
		}
	}
	n = len(p)
	return
}

func (r *Reader) Decode(f *File) error {
	r.m.Lock()
	defer r.m.Unlock()
	if f == nil {
		return fmt.Errorf("nil")
	}
	if r.last.line.Len() > 0 {
		if err := r.readLastLine(); err != nil {
			return err
		}
	}
	if r.last.value.Len() > 0 {
		if err := r.flushLastEntry(); err != nil {
			return err
		}
	}
	if len(r.rawMetadata) > 0 {
		metad := Meta{}
		ml := strings.Split(r.rawMetadata, "\n")
		for _, v := range ml {
			if i := strings.Index(v, ":"); i != -1 {
				metad.Set(v[:i], strings.TrimSpace(v[i+1:]))
			}
		}
		f.Metadata = metad
	}
	f.Entries = r.entries
	f.Context = r.contextEntries
	return nil
}

// reads and cleans line
func (r *Reader) readLastLine() error {
	defer func() {
		r.last.line.Reset()
		r.last.n++
	}()
	l := strings.TrimSpace(r.last.line.String())
	if len(l) == 0 {
		return nil
	}
	if l[0] == '#' {
		// comment
		return nil
	}
	if l[0] == '"' {
		if r.last.state == stateInit {
			if r.Strict {
				return syntaxError(r.last.n)
			}
		} else {
			parsed, err := strconv.Unquote(l)
			if err != nil {
				if r.Strict {
					return syntaxError(r.last.n)
				}
			} else {
				switch r.last.state {
				case stateReadContext:
					r.last.context.WriteString(parsed)
				case stateReadId:
					r.last.key.WriteString(parsed)
				case stateReadValue:
					r.last.value.WriteString(parsed)
				}
			}
		}
	} else if strings.Index(l, "msgctxt ") == 0 {
		if r.last.state != stateInit {
			if err := r.flushLastEntry(); err != nil {
				return err
			}
		}
		r.last.state = stateReadContext
		// next message will have a context
		r.last.withContext = true
		l = strings.TrimSpace(l[8:])
		if !lineIsValid(l) {
			if r.Strict {
				return syntaxError(r.last.n)
			}
		} else {
			parsed, err := strconv.Unquote(l)
			if err != nil {
				if r.Strict {
					return syntaxError(r.last.n)
				}
			} else {
				r.last.context.WriteString(parsed)
			}
		}
	} else if strings.Index(l, "msgid ") == 0 {
		if r.last.state != stateReadContext && r.last.state != stateInit {
			if err := r.flushLastEntry(); err != nil {
				return err
			}
		}
		r.last.state = stateReadId
		l = strings.TrimSpace(l[6:])
		if !lineIsValid(l) {
			if r.Strict {
				return syntaxError(r.last.n)
			}
		} else {
			parsed, err := strconv.Unquote(l)
			if err != nil {
				if r.Strict {
					return syntaxError(r.last.n)
				}
			} else {
				r.last.key.WriteString(parsed)
			}
		}
	} else if strings.Index(l, "msgstr ") == 0 {
		r.last.state = stateReadValue
		l = strings.TrimSpace(l[7:])
		if !lineIsValid(l) {
			if r.Strict {
				return syntaxError(r.last.n)
			}
		} else {
			parsed, err := strconv.Unquote(l)
			if err != nil {
				if r.Strict {
					return syntaxError(r.last.n)
				}
			} else {
				r.last.value.WriteString(parsed)
			}
		}
	} else {
		if r.Strict {
			return syntaxError(r.last.n)
		}
	}
	return nil
}

func syntaxError(l0 int) error {
	return fmt.Errorf("syntax error on line %v", l0+1)
}

func duplicatedKeyError(key string, l0 int) error {
	return fmt.Errorf("duplicated key '%v' near line %v", key, l0+1)
}

func (r *Reader) flushLastEntry() error {
	ctx := r.last.context.String()
	hasctx := r.last.withContext
	key := r.last.key.String()
	val := r.last.value.String()
	r.last.context.Reset()
	r.last.withContext = false
	r.last.key.Reset()
	r.last.value.Reset()
	r.last.state = stateInit
	if hasctx {
		if r.contextEntries == nil {
			r.contextEntries = make(map[string]map[string]string)
		}
		if r.contextEntries[ctx] == nil {
			r.contextEntries[ctx] = make(map[string]string)
		}
		m := r.contextEntries[ctx]
		if _, ok := m[key]; ok && r.Strict {
			return duplicatedKeyError(key, r.last.n)
		}
		m[key] = val
		r.contextEntries[ctx] = m
	} else {
		if r.entries == nil {
			r.entries = make(map[string]string)
		}
		if _, ok := r.entries[key]; ok && r.Strict {
			return duplicatedKeyError(key, r.last.n)
		}
		r.entries[key] = val
		if key == "" {
			r.rawMetadata = val
		}
	}
	return nil
}

// lineIsValid tests if the first and last characters are double quotes.
// It doesn't fully check if the quoted string is valid.
// For this, use strconv.Unquote(str)
func lineIsValid(l string) bool {
	return l[0] == '"' && l[len(l)-1] == '"'
}
