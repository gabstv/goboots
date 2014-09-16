package goboots

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestPartialTemplateGet(t *testing.T) {
	wd00, _ := os.Getwd()
	wd0 := path.Join(wd00, ".testpartials")
	err := os.Mkdir(wd0, 0777)
	//if err != nil {
	//	t.Fatal(err)
	//}
	err = os.Mkdir(path.Join(wd0, "view"), 0777)
	//if err != nil {
	//	t.Fatal(err)
	//}
	ioutil.WriteFile(path.Join(wd0, "/view/t1.tpl"), []byte(`{{partial /t2.tpl}} WORLD!`), 0777)
	ioutil.WriteFile(path.Join(wd0, "/view/t2.tpl"), []byte(`HELLO,`), 0777)
	final, err := parseTemplateIncludeDeps(wd0, "view", path.Join(wd0, "/view/"), []byte(`{{partial ../view/t1.tpl}}`))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(final, []byte(`HELLO, WORLD!`)) != 0 {
		t.Fatal(string(final) + " != HELLO, WORLD!")
	}
	os.Remove(path.Join(wd0, "/view/t2.tpl"))
	os.Remove(path.Join(wd0, "/view/t1.tpl"))
	os.Remove(path.Join(wd0, "view/"))
	os.Remove(wd0)
}
