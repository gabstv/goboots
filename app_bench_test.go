package goboots

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func testRequest(method, path string) (respRec *httptest.ResponseRecorder, r *http.Request) {
	r, _ = http.NewRequest(method, path, nil)
	respRec = httptest.NewRecorder()
	return
}

type BenchmarkController struct {
	Controller
}

func (c *BenchmarkController) Bench(in *In) *Out {
	return in.OutputString("hello")
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
	app.loadAll()
	//app.runRoutines()

	w, r := testRequest("GET", "/")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.ServeHTTP(w, r)
	}
}
