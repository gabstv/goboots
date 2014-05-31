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
		return route.isMatchRegExp(url)
	case byte(1):
		return route.isMatchRemainder(url)
	case byte(0):
		return route.isMatchExact(url)
	}
	return false
}

func (route *Route) isMatchRegExp(url string) bool {
	match, _ := regexp.MatchString(route.Path, url)
	return match
}

func (route *Route) isMatchExact(url string) bool {
	return url == route.Path
}

func (route *Route) isMatchRemainder(url string) bool {
	return strings.HasPrefix(url, route.Path)
}
