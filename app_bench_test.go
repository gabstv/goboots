package goboots

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type BenchmarkRouterBuilder func(namespaces []string, resources []string) http.Handler

// https://github.com/agrison/golang-mux-benchmark/blob/master/mux_bench_test.go
func testRequest(method, path string) (respRec *httptest.ResponseRecorder, r *http.Request) {
	r, _ = http.NewRequest(method, path, nil)
	respRec = httptest.NewRecorder()
	return
}

// Returns a routeset with N *resources per namespace*. so N=1 gives about 15 routes
func resourceSetup(N int) (namespaces []string, resources []string, requests []*http.Request) {
	namespaces = []string{"admin", "api", "site"}
	resources = []string{}

	for i := 0; i < N; i += 1 {
		sha1 := sha1.New()
		io.WriteString(sha1, fmt.Sprintf("%d", i))
		strResource := fmt.Sprintf("%x", sha1.Sum(nil))
		resources = append(resources, strResource)
	}

	for _, ns := range namespaces {
		for _, res := range resources {
			req, _ := http.NewRequest("GET", "/"+ns+"/"+res, nil)
			requests = append(requests, req)
			req, _ = http.NewRequest("POST", "/"+ns+"/"+res, nil)
			requests = append(requests, req)
			req, _ = http.NewRequest("GET", "/"+ns+"/"+res+"/3937", nil)
			requests = append(requests, req)
			req, _ = http.NewRequest("PUT", "/"+ns+"/"+res+"/3937", nil)
			requests = append(requests, req)
			req, _ = http.NewRequest("DELETE", "/"+ns+"/"+res+"/3937", nil)
			requests = append(requests, req)
		}
	}

	return
}

func benchmarkRoutes(b *testing.B, handler http.Handler, requests []*http.Request) {
	recorder := httptest.NewRecorder()
	reqId := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if reqId >= len(requests) {
			reqId = 0
		}
		req := requests[reqId]
		handler.ServeHTTP(recorder, req)

		if recorder.Code != 200 {
			panic("wat")
		}

		reqId += 1
	}
}

func benchmarkRoutesN(b *testing.B, N int, builder BenchmarkRouterBuilder) {
	namespaces, resources, requests := resourceSetup(N)
	router := builder(namespaces, resources)
	benchmarkRoutes(b, router, requests)
}

func gobootsRouterFor(namespaces []string, resources []string) http.Handler {
	app := NewApp()
	app.Config = &AppConfig{
		Name:            "Benchmark",
		HostAddr:        ":19019",
		GlobalPageTitle: "Benchmark - ",
	}
	app.RegisterController(&BenchmarkController{})
	for _, ns := range namespaces {
		for _, res := range resources {
			app.AddRouteLine("GET /" + ns + "/" + res + " BenchmarkController.Bench")
			app.AddRouteLine("POST /" + ns + "/" + res + " BenchmarkController.Bench")
			app.AddRouteLine("GET /" + ns + "/" + res + "/:id BenchmarkController.Bench")
			app.AddRouteLine("PUT /" + ns + "/" + res + "/:id BenchmarkController.Bench")
			app.AddRouteLine("DELETE /" + ns + "/" + res + "/:id BenchmarkController.Bench")
		}
	}
	app.BenchLoadAll()
	return app
}

type BenchmarkController struct {
	Controller
}

func (c *BenchmarkController) Bench(in *In) *Out {
	return in.OutputString("hello")
}

func (c *BenchmarkController) Composite(in *In) *Out {
	return in.OutputString(in.Params["MyField"])
}

func BenchmarkGoboots_Simple(b *testing.B) {
	//
	app := NewApp()
	app.Config = &AppConfig{
		Name:            "Benchmark",
		HostAddr:        ":19019",
		GlobalPageTitle: "Benchmark - ",
	}
	app.RegisterController(&BenchmarkController{})
	app.AddRouteLine("GET / BenchmarkController.Bench")
	app.BenchLoadAll()

	w, r := testRequest("GET", "/")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.ServeHTTP(w, r)
	}
}

func BenchmarkGoboots_Route15(b *testing.B) {
	benchmarkRoutesN(b, 1, gobootsRouterFor)
}

func BenchmarkGoboots_Route75(b *testing.B) {
	benchmarkRoutesN(b, 5, gobootsRouterFor)
}

func BenchmarkGoboots_Route150(b *testing.B) {
	benchmarkRoutesN(b, 10, gobootsRouterFor)
}

func BenchmarkGoboots_Route300(b *testing.B) {
	benchmarkRoutesN(b, 20, gobootsRouterFor)
}

func BenchmarkGoboots_Route3000(b *testing.B) {
	benchmarkRoutesN(b, 200, gobootsRouterFor)
}

func BenchmarkGoboots_Middleware(b *testing.B) {
	myFilter := func(in *In) bool {
		return true
	}

	app := NewApp()
	app.Config = &AppConfig{
		Name:            "Benchmark",
		HostAddr:        ":19019",
		GlobalPageTitle: "Benchmark - ",
	}

	app.Filters = []Filter{myFilter, myFilter, myFilter, myFilter, myFilter, myFilter}

	app.RegisterController(&BenchmarkController{})
	app.AddRouteLine("GET / BenchmarkController.Bench")
	app.BenchLoadAll()

	w, r := testRequest("GET", "/")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.ServeHTTP(w, r)
		if w.Code != 200 {
			panic("no good")
		}
	}
}

func BenchmarkGoboots_Composite(b *testing.B) {
	namespaces, resources, requests := resourceSetup(10)

	myFilter := func(in *In) bool {
		return true
	}

	myFilterDoes := func(in *In) bool {
		in.Params["MyField"] = in.R.URL.Path
		return true
	}

	app := NewApp()
	app.Config = &AppConfig{
		Name:            "Benchmark",
		HostAddr:        ":19019",
		GlobalPageTitle: "Benchmark - ",
	}

	app.Filters = []Filter{myFilterDoes, myFilter, myFilter, myFilter, myFilter, myFilter}

	app.RegisterController(&BenchmarkController{})

	for _, ns := range namespaces {
		for _, res := range resources {
			app.AddRouteLine("GET /" + ns + "/" + res + " BenchmarkController.Composite")
			app.AddRouteLine("POST /" + ns + "/" + res + " BenchmarkController.Composite")
			app.AddRouteLine("GET /" + ns + "/" + res + "/:id BenchmarkController.Composite")
			app.AddRouteLine("PUT /" + ns + "/" + res + "/:id BenchmarkController.Composite")
			app.AddRouteLine("DELETE /" + ns + "/" + res + "/:id BenchmarkController.Composite")
		}
	}
	app.BenchLoadAll()
	benchmarkRoutes(b, app, requests)
}
