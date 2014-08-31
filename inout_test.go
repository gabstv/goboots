package goboots

import (
	"testing"
)

func TestInContentMerge(t *testing.T) {
	c0 := &InContent{}
	c0.init()
	c0.Set("Pie", 1)
	map2 := make(map[string]interface{})
	map2["Apple"] = 5
	c0.Merge(map2)
	_, ok := c0.Get2("Apple")
	if !ok {
		t.Fatal("InContent Merge FAILED")
	}
	struct0 := struct {
		Oranges int
	}{10}
	c0.Merge(struct0)
	ocount, _ := c0.GetInt2("Oranges")
	if ocount != 10 {
		t.Fatal("InContent Merge FAILED [2]")
	}
}
