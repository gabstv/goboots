package goboots

import (
	"net/url"
	"testing"
)

func TestTedirectURLs(t *testing.T) {

	uris := [][]string{
		[]string{"http://john:doe@www.google.com/randomurl?active=true", "https://john:doe@www.google.com/randomurl?active=true", "true", ":443"},
		[]string{"ws://john:doe@fibbo.com/randomurl?active=true", "wss://john:doe@fibbo.com:8083/randomurl?active=true", "true", ":8083"},
		[]string{"ws://john:doe@fibbo.com/randomurl?active=true", "wss://john:doe@fibbo.com/randomurl?active=true", "true", ":443"},
		[]string{"ws://localhost/randomurl?active=true", "wss://localhost:449/randomurl?active=true", "true", ":449"},
		[]string{"ws://localhost/randomurl?active=true", "", "false", "hostwithoutport"},
	}

	for _, v := range uris {
		uri, _ := url.Parse(v[0])
		rr, err := getTLSRedirectURL(v[3], uri)
		if v[2] == "true" && err != nil {
			t.Fatalf("Error [1] on url %v\n", v[0])
		} else if v[2] == "false" && err == nil {
			t.Fatalf("Error [2] on url %v\n", v[0])
		}
		if err == nil {
			if v[1] != rr {
				t.Fatalf("Error [3] URL should be %v but it is %v\n", v[1], rr)
			}
		}
	}
}
