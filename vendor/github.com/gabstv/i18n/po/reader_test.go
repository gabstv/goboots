package po

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidLines(t *testing.T) {
	assert.Equal(t, true, lineIsValid(`"valid 1"`))
}

func TestBasic(t *testing.T) {
	pofile := `
msgid ""
msgstr ""
"Language-Code: en\n"
"Language-Name: English\n"
"Last-Translator: FULL NAME <EMAIL@ADDRESS>\n"

msgid "marco"
msgstr "polo"

# next id must compile to "abc"
msgid ""
"a"
"b"
"c"
msgstr "cba"
	`
	var rdr Reader
	rdr.Strict = true
	_, err := rdr.Read([]byte(pofile))
	assert.NoError(t, err)
	var f File
	err = rdr.Decode(&f)
	assert.NoError(t, err)
	assert.Equal(t, "polo", f.Entries["marco"])
	assert.Equal(t, "cba", f.Entries["abc"])
	assert.Equal(t, "English", f.Metadata.Get("Language-Name"))
	assert.Equal(t, "en", f.Metadata.Get("Language-Code"))
	f.Metadata.Del("Language-Code")
	assert.Equal(t, "", f.Metadata.Get("Language-Code"))
}

func TestParseErrors(t *testing.T) {
	var rdr Reader
	rdr.Strict = true
	v := `
msgid +
msgstr ""
`
	_, err := rdr.Read([]byte(v))
	require.Error(t, err)
	v = `
msgid ""
"a"
msgstr ""
"""
`
	_, err = rdr.Read([]byte(v))
	require.Error(t, err)
	_, err = rdr.Read(nil)
	require.NoError(t, err)
	err = rdr.Decode(nil)
	require.Error(t, err)
	require.Equal(t, "nil", err.Error())
}

func TestContext(t *testing.T) {
	var rdr Reader
	rdr.Strict = true
	v := `
msgctxt "a"
msgid "key"
msgstr "val ctx a"

msgctxt "b"
msgid "key"
msgstr "val ctx b"`
	_, err := rdr.Read([]byte(v))
	assert.NoError(t, err)
	var f File
	assert.NoError(t, rdr.Decode(&f))
	assert.Equal(t, "val ctx a", f.Context["a"]["key"])
	assert.Equal(t, "val ctx b", f.Context["b"]["key"])
	assert.Equal(t, "", rdr.last.context.String())
	assert.Equal(t, "", rdr.last.key.String())
	assert.Equal(t, "", rdr.last.value.String())
	assert.Equal(t, "", rdr.last.line.String())
	v = `msgctxt "a"
msgid "key"
msgstr "must trigger error"
msgid "dd"
msgstr "b1"
msgid "dd"
msgstr "b2"
`
	_, err = rdr.Read([]byte(v))
	t.Log(rdr.contextEntries, rdr.entries)
	t.Log(rdr.last.context.String())
	t.Log(rdr.last.key.String())
	t.Log(rdr.last.value.String())
	require.Error(t, err)
	assert.Equal(t, "duplicated key 'key' near line 12", err.Error())
}

func TestDupe(t *testing.T) {
	var rdr Reader
	rdr.Strict = true
	rdr.last.line.WriteString(`msgid "a"`)
	assert.NoError(t, rdr.readLastLine())
	rdr.last.line.WriteString(`msgstr "1"`)
	assert.NoError(t, rdr.readLastLine())
	rdr.last.line.WriteString(`msgid "a"`)
	assert.NoError(t, rdr.readLastLine())
	assert.Equal(t, "", rdr.last.value.String())
	rdr.last.line.WriteString(`msgstr "2"`)
	assert.NoError(t, rdr.readLastLine())
	rdr.last.line.WriteString(`msgid "a"`)
	assert.Error(t, rdr.readLastLine())
}

func TestDupeCtx(t *testing.T) {
	var rdr Reader
	rdr.Strict = true
	rdr.last.line.WriteString(`msgctxt "a"`)
	assert.NoError(t, rdr.readLastLine())
	rdr.last.line.WriteString(`msgid "a"`)
	assert.NoError(t, rdr.readLastLine())
	rdr.last.line.WriteString(`msgstr "1"`)
	assert.NoError(t, rdr.readLastLine())
	rdr.last.line.WriteString(`msgctxt "a"`)
	assert.NoError(t, rdr.readLastLine())
	assert.Equal(t, rdr.contextEntries["a"]["a"], "1")
	rdr.last.line.WriteString(`msgid "a"`)
	assert.NoError(t, rdr.readLastLine())
	assert.Equal(t, "", rdr.last.value.String())
	rdr.last.line.WriteString(`msgstr "2"`)
	assert.NoError(t, rdr.readLastLine())
	rdr.last.line.WriteString(`msgid "a"`)
	assert.Error(t, rdr.readLastLine())
}

func TestMultiline(t *testing.T) {
	var rdr Reader
	rdr.Strict = true
	n, err := rdr.Read([]byte(`msgctxt ""
"a"
"1"
msgid "b"
"1"
""
"0"
msgstr "5"
""
"7"

`))
	assert.Equal(t, 59, n)
	assert.NoError(t, err)
	var f File
	assert.NoError(t, rdr.Decode(&f))
	assert.Equal(t, f.Context["a1"]["b10"], "57")
}

func TestInitError(t *testing.T) {
	var rdr Reader
	rdr.Strict = true
	_, err := rdr.Read([]byte("\"not ok\"\n"))
	assert.Error(t, err)
	_, err = rdr.Read([]byte("invalid\n"))
	assert.Error(t, err)
}
