package goboots

import (
	"regexp"
	"strings"
)

type Route struct {
	Path        string
	Controller  string
	Method      string
	RedirectTLS bool
	_t          byte
}

func (route *Route) IsMatch(url string) bool {
	switch route._t {
	case byte(2):
		return route._isMatchRegExp(url)
	case byte(1):
		return route._isMatchRemainder(url)
	case byte(0):
		return route._isMatchExact(url)
	}
	return false
}

func (route *Route) _isMatchRegExp(url string) bool {
	match, _ := regexp.MatchString(route.Path, url)
	return match
}

func (route *Route) _isMatchExact(url string) bool {
	return url == route.Path
}

func (route *Route) _isMatchRemainder(url string) bool {
	return strings.Index(url, route.Path) == 0
}
