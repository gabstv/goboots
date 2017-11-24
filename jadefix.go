package goboots

import (
	"bytes"
)

// TEMPORARY workaround for
// https://github.com/Joker/jade/issues/16
func tempJadeFix(parsed, leftdelim, rightdelim string) string {
	inside := 0

	lrunes := make([]rune, 0)
	rrunes := make([]rune, 0)

	for _, r := range leftdelim {
		lrunes = append(lrunes, r)
	}
	for _, r := range rightdelim {
		rrunes = append(rrunes, r)
	}

	prev0 := make([]rune, len(lrunes))
	prev1 := make([]rune, len(rrunes))

	push := func(b []rune, v rune) {
		for k := 0; k < len(b)-1; k++ {
			b[k] = b[k+1]
		}
		b[len(b)-1] = v
	}
	equal := func(a, b []rune) bool {
		if len(a) != len(b) {
			return false
		}
		for k := range a {
			if a[k] != b[k] {
				return false
			}
		}
		return true
	}

	var buf bytes.Buffer

	for _, r := range parsed {
		if inside > 0 {
			if r == '\'' {
				buf.WriteRune('"')
			} else {
				buf.WriteRune(r)
			}
		} else {
			buf.WriteRune(r)
		}
		push(prev0, r)
		if equal(prev0, lrunes) {
			inside++
		}
		push(prev1, r)
		if equal(prev1, rrunes) {
			inside--
		}
	}
	return buf.String()
}
