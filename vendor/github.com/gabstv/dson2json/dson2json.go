package dson2json

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
)

var (
	keywords = map[string]string{
		"such":  "{",
		"wow":   "}",
		"is":    ":",
		"so":    "[",
		"and":   ",",
		"also":  ",",
		"many":  "]",
		"yes":   "true",
		"no":    "false",
		"empty": "null",
		".":     ",",
		",":     ",",
		"!":     ",",
		"?":     ",",
	}
)

func convert(input io.Reader, output io.Writer) error {
	tokens := make(chan string, 16)
	returner := make(chan error)
	inString := false
	stringQuote := '"'
	currentToken := new(bytes.Buffer)
	dogeIsEscaping := false
	rdr := bufio.NewReader(input)

	doingDoge := true

	go func() {
		for {
			var cdoge string
			if !doingDoge {
				select {
				case cdoge = <-tokens:
				case <-time.After(time.Millisecond):
					returner <- nil
					return
				}
			} else {
				cdoge = <-tokens
			}
			if len(keywords[cdoge]) > 0 {
				_, err := output.Write([]byte(keywords[cdoge]))
				if err != nil {
					returner <- err
					return
				}
			} else if cdoge[0] != '"' {
				n0 := strings.Replace(strings.ToLower(cdoge), "very", "e", -1)
				_, err := strconv.ParseFloat(n0, 64)
				if err == nil {
					_, err = output.Write([]byte(n0))
					if err != nil {
						returner <- err
						return
					}
				} else {
					returner <- errors.New("Unable to parse number `" + cdoge + "`. " + err.Error())
					return
				}
			} else {
				_, err := output.Write([]byte(cdoge))
				if err != nil {
					returner <- err
					return
				}
			}
		}
	}()

	for {
		currentChar, _, err := rdr.ReadRune()
		if err != nil {
			doingDoge = false
			break
		}
		if !inString && currentChar == '"' {
			inString = true
			stringQuote = currentChar
			currentToken.WriteRune(currentChar)
		} else if inString {
			if !dogeIsEscaping && currentChar == stringQuote {
				currentToken.WriteRune(currentChar)
				tokens <- currentToken.String()
				inString = false
				currentToken.Reset()
			} else if currentChar == '\\' {
				dogeIsEscaping = true
				currentToken.WriteRune(currentChar)
			} else {
				currentToken.WriteRune(currentChar)
				dogeIsEscaping = false
			}
		} else if currentChar == ' ' || currentChar == '\n' || currentChar == '\r' || currentChar == '\t' {
			if currentToken.Len() > 0 {
				tokens <- currentToken.String()
				currentToken.Reset()
			}
		} else if currentChar == ',' || currentChar == '.' || currentChar == '!' || currentChar == '?' {
			if currentToken.Len() > 0 {
				tokens <- currentToken.String()
			}
			tokens <- string(currentChar)
			currentToken.Reset()
		} else {
			currentToken.WriteRune(currentChar)
		}
	}
	if currentToken.Len() > 0 {
		tokens <- currentToken.String()
	}
	return <-returner
}

func Convert(input io.Reader, output io.Writer) error {
	return convert(input, output)
}

//TODO: add more function shortcuts
