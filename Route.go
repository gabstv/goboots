package goboots

import (
	"regexp"
	"strings"
)

const (
	routeMethodExact       = byte(0)
	routeMethodRemainder   = byte(1)
	routeMethodRegExp      = byte(2)
	routeMethodIgnoreTrail = byte(3)
)

type OldRoute struct {
	Path        string
	Controller  string
	Method      string
	RedirectTLS bool
	_t          byte
}

func (route *OldRoute) IsMatch(url string) bool {
	switch route._t {
	case routeMethodRegExp:
		return route.isMatchRegExp(url)
	case routeMethodRemainder:
		return route.isMatchRemainder(url)
	case routeMethodExact:
		return route.isMatchExact(url)
	case routeMethodIgnoreTrail:
		return route.isMatchIgnoreTrail(url)
	}
	return false
}

func (route *OldRoute) isMatchRegExp(url string) bool {
	match, _ := regexp.MatchString(route.Path, url)
	return match
}

func (route *OldRoute) isMatchExact(url string) bool {
	return url == route.Path
}

func (route *OldRoute) isMatchRemainder(url string) bool {
	return strings.HasPrefix(url, route.Path)
}

func (route *OldRoute) isMatchIgnoreTrail(url string) bool {
	return strings.TrimRight(url, "/") == route.Path[:len(route.Path)-2]
}
