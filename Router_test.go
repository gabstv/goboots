package goboots

import (
	"regexp"
	"testing"
)

func TestRouteLineReader(t *testing.T) {
	paths := [][]string{
		[]string{
			"GET  /login       App.Login         # A simple path",
			"true",
			"GET", "/login", "App.Login", "", ""},
		[]string{
			"GET  /login       App.Login",
			"true",
			"GET", "/login", "App.Login", "", ""},
		[]string{
			"WS   /ws/chat     Chat.WSChat  TLS  # Websocket chat",
			"true",
			"WS", "/ws/chat", "Chat.WSChat", "", "TLS"},
		[]string{
			"WS   /ws/chat     Chat.WSChat  TLS",
			"true",
			"WS", "/ws/chat", "Chat.WSChat", "", "TLS"},
		[]string{
			"POST /action/:id  Home.Action(\"okay\", \"(())\") # Action",
			"true",
			"POST", "/action/:id", "Home.Action", "\"okay\",\"(())\"", ""},
		[]string{
			"POST /action/:id  Home.Action(\"okay\", \"(())\")",
			"true",
			"POST", "/action/:id", "Home.Action", "\"okay\",\"(())\"", ""},
		[]string{
			"POST /action/:id  Home.Action(\"okay\", \"(())\") TLS",
			"true",
			"POST", "/action/:id", "Home.Action", "\"okay\",\"(())\"", "TLS"},
		[]string{
			"POST /action/:id  Home.Action(\"okay\", \"(())\") TLS #Comment",
			"true",
			"POST", "/action/:id", "Home.Action", "\"okay\",\"(())\"", "TLS"},
		[]string{
			"POST /action/:id  Home.Action(\"okay\", \"it is \\\"really\\\" corn\") TLS #Comment",
			"true",
			"POST", "/action/:id", "Home.Action", "\"okay\",\"it is \\\"really\\\" corn\"", "TLS"},
		// invalid ones
		[]string{
			"GET  /login      # A simple path",
			"false",
			"GET", "", "", "", ""},
		[]string{
			"GET  login      # A simple path",
			"false",
			"GET", "", "", "", ""},
		[]string{
			"NO  /login  Login(\"a\")    # A simple path",
			"false",
			"", "", "", "", ""},
		[]string{
			"GET  /login App.Login( ",
			"false",
			"", "", "", "", ""},
		[]string{
			"GET  /login App.Login( notquoted ) ",
			"false",
			"", "", "", "", ""},
		[]string{
			"GET  /login App.Login( notquoted ) ",
			"false",
			"", "", "", "", ""},
		[]string{
			"GET  /login App.Login TLz ",
			"false",
			"", "", "", "", ""},
	}
	for n, v := range paths {
		method, path, action, fixedArgs, tls, found, errormessage, _ := routeParseLine(v[0])
		if found && v[1] != "true" {
			t.Fatalf("Route \n%v\n  Should be invalid!\n", v[0])
		} else if !found && v[1] == "true" {
			t.Fatalf("Route \n%v\n (%s)  Should be valid!\n", v[0], errormessage)
		}
		if found {
			if method != v[2] {
				t.Fatalf("Method of route %v should be '%v' but it is '%v'\n", n, v[2], method)
			}
			if path != v[3] {
				t.Fatalf("Path of route %v should be '%v' but it is '%v'\n", n, v[3], path)
			}
			if action != v[4] {
				t.Fatalf("Action of route %v should be '%v' but it is '%v'\n", n, v[4], action)
			}
			if fixedArgs != v[5] {
				t.Fatalf("FixedArgs of route %v should be '%v' but it is '%v'\n", n, v[5], fixedArgs)
			}
			if !tls && v[6] == "TLS" {
				t.Fatalf("TLS must be true on route %v\n", n)
			}
			if tls && v[6] == "" {
				t.Fatalf("TLS must be false on route %v\n", n)
			}
		}
	}
}

func BenchmarkLineReaderOld(b *testing.B) {
	var routePattern *regexp.Regexp = regexp.MustCompile(
		"(?i)^(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD|WS|\\*)" +
			"[(]?([^)]*)(\\))?[ \t]+" +
			"(.*/[^ \t]*)[ \t]+([^ \t(]+)" +
			`\(?([^)]*)\)?[ \t]*$`)
	paths := []string{
		"GET  /login       App.Login         # A simple path",
		`POST /action/:id  Home.Action("one","two","three") # Action`,
		`WS /ws/sync       WSS.Sync # Comment`,
	}
	pn := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		routePattern.FindStringSubmatch(paths[pn])
		pn++
		if pn >= len(paths) {
			pn = 0
		}
	}
}

func BenchmarkLineReaderNew(b *testing.B) {
	paths := []string{
		"GET  /login       App.Login         # A simple path",
		`POST /action/:id  Home.Action("one","two","three") # Action`,
		`WS /ws/sync       WSS.Sync # Comment`,
	}
	pn := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		routeParseLine(paths[pn])
		pn++
		if pn >= len(paths) {
			pn = 0
		}
	}
}
