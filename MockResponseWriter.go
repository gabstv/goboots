package goboots

import (
	"bytes"
	"net/http"
)

type MockResponseWriter struct {
	header http.Header
	buffer bytes.Buffer
	resp   int
}

func (w *MockResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

func (w *MockResponseWriter) Write(b []byte) (int, error) {
	return w.buffer.Write(b)
}

func (w *MockResponseWriter) WriteHeader(status int) {
	w.resp = status
}

func (w *MockResponseWriter) StringBody() string {
	return w.buffer.String()
}

func (w *MockResponseWriter) Body() []byte {
	return w.buffer.Bytes()
}

func (w *MockResponseWriter) StatusCode() int {
	return w.resp
}
