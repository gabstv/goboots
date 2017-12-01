# DSON to JSON
## A DSON/DogeON converter made in Go.

Based on http://jsfiddle.net/Ha65v/3/

Usage example:
```Go
dson := `such "foo" is so no and yes many wow`
in0 := bytes.NewBufferString(dson)
out0 := new(bytes.Buffer)
dson2json.Convert(in0, out0)
fmt.Println(out0.String()) // prints {"foo":[false,true]}
```

[DogeON Syntax Reference](http://dogeon.org)