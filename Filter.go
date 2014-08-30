package goboots

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type Filter func(in *In) bool

// gzip compression filter

type gzipRespWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipRespWriter) Write(b []byte) (int, error) {
	if w.Header().Get("Content-Type") == "" {
		// If no content type, apply sniffing algorithm to un-gzipped body.
		w.Header().Set("Content-Type", http.DetectContentType(b))
	}
	return w.Writer.Write(b)
}

func CompressFilter(in *In) bool {
	if !strings.Contains(in.R.Header.Get("Accept-Encoding"), "gzip") {
		return true
	}
	in.W.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(in.W)
	gzr := &gzipRespWriter{gz, in.W}
	in.W = gzr
	in.closers = append(in.closers, gz)
	return true
}