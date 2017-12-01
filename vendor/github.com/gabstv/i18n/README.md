i18n
======

http://pology.nedohodnik.net/doc/user/en_US/ch-poformat.html

```
msgid ""
msgstr ""
"Project-Id-Version: PACKAGE VERSION\n"
"POT-Creation-Date: 2014-01-26 10:00+0000\n"
"PO-Revision-Date: YEAR-MO-DA HO:MI +ZONE\n"
"Last-Translator: FULL NAME <EMAIL@ADDRESS>\n"
"Language-Team: LANGUAGE <example@example.com>\n"
"MIME-Version: 1.0\n"
"Content-Type: text/plain; charset=utf-8\n"
"Content-Transfer-Encoding: 8bit\n"
"Plural-Forms: nplurals=1; plural=0\n"
"Language-Code: en\n"
"Language-Name: English\n"
"Preferred-Encodings: utf-8 latin1\n"
"Domain: DOMAIN\n"

#. Default: "Comment here"
#: ./etc/a/file.ext:55
msgid "description_content_here"
msgstr "hello, %v"

# msg with context
#: ./skins/archetypesmultilingual/at_babel_edit.cpt:95
msgctxt "context_a"
msgid "some key"
msgstr "some value"
```

```Go
import (
	"github.com/gabstv/i18n/po/poutil"	
)

func main(){
	provider, err := poutil.LoadAll("path/to/root/translations/dir", "en")
	if err := nil {
		panic(err)
	}
	print(provider.L("en").T("description_content_here", "world") + "\n")
	print(provider.L("en").Ctx("context_a")("some key") + "\n")
}
```