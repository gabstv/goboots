package goboots

import (
	"bytes"

	"github.com/gabstv/i18n"
)

const (
	runePct = rune('%')
)

func LocalizeTemplate(templateStr string, langcode string, provider i18n.Provider) string {
	if provider == nil {
		return templateStr
	}
	runes := []rune(templateStr)
	var current rune
	var curStr bytes.Buffer
	var buf bytes.Buffer
	//insideLoc := false
	max := len(runes)
	for i := 0; i < max; i++ {
		current = runes[i]
		if current == runePct {
			if i+1 >= max {
				// it's the last character
				buf.WriteRune(current)
				continue
			}
			if i+2 >= max {
				// it's the last character
				buf.WriteRune(runes[i+1])
				continue
			}
			// check if next 2 runes are a magic character too
			if runes[i+1] == runePct && runes[i+2] == runePct {
				if i+3 >= max {
					//EOF!
					buf.WriteString("%%%")
				}
				// capture str
				i += 3
				curStr.Reset()
				for {
					current = runes[i]
					if i+1 >= max {
						//EOF while trying to fetch a localized string
						buf.WriteRune(current)
						break
					}
					if current == runePct {
						// try to close!
						if runes[i+1] == runePct {
							if i+2 >= max {
								//EOF while trying to fetch a localized string
								buf.WriteRune(runes[i+1])
								break
							}
							if runes[i+2] == runePct {
								// finalize string!
								i += 2
								if provider == nil {
									buf.WriteString(curStr.String())
								} else {
									if ll := provider.L(langcode); ll != nil {
										buf.WriteString(ll.T(curStr.String()))
									} else {
										buf.WriteString(curStr.String())
									}
								}
								break
							}
						}
					}
					curStr.WriteRune(current)
					i++
				}
			} else {
				// just write the rune normally
				buf.WriteRune(current)
			}
		} else {
			// just write the rune normally
			buf.WriteRune(current)
		}
	}
	return buf.String()
}
