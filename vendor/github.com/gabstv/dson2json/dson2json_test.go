package dson2json

import (
	"bytes"
	"testing"
)

func TestSyntax(t *testing.T) {
	dson0 := `such "foo" is empty wow`
	json0 := `{"foo":null}`

	var in0, out0 bytes.Buffer

	in0.WriteString(dson0)

	err := Convert(&in0, &out0)
	if err != nil {
		t.Fatal()
		return
	}
	if out0.String() != json0 {
		t.Fatalf("Expected `%v` got `%v`", json0, out0.String())
	}
}

func TestManyWow(t *testing.T) {
	dsons := []string{
		`such "foo" is "bar" wow`,
		`such "foo" is so "bar" and "baz" also "fizzbuzz" many wow`,
		`such "foo" is 42very3 wow`,
		`such " \"such " is "bar" wow`,
		`such
            " such " is "very bar" wow`,
		`such "foo" is such "shiba" is "inu". "doge" is such "good" is yes! "a" is empty ? "b" is no wow wow wow`,
	}
	jsons := []string{
		`{"foo":"bar"}`,
		`{"foo":["bar","baz","fizzbuzz"]}`,
		`{"foo":42e3}`,
		`{" \"such ":"bar"}`,
		`{" such ":"very bar"}`,
		`{"foo":{"shiba":"inu","doge":{"good":true,"a":null,"b":false}}}`,
	}

	var in0, out0 bytes.Buffer

	for k, v := range jsons {
		in0.Reset()
		out0.Reset()
		in0.WriteString(dsons[k])
		err := Convert(&in0, &out0)
		if err != nil {
			t.Fatal(err)
		}
		if out0.String() != v {
			t.Fatalf("Expected `%v` got `%v`", v, out0.String())
		} else {
			t.Logf("In: `%v` Out: `%v`", dsons[k], out0.String())
		}
	}
}
