package po

type File struct {
	Metadata Meta
	Entries  map[string]string
	Context  map[string]map[string]string
}

type Meta map[string]string

func (m Meta) Get(key string) string {
	return m[key]
}

func (m Meta) Set(key, value string) {
	m[key] = value
}

func (m Meta) Del(key string) {
	delete(m, key)
}
