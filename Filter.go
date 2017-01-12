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

// based on https://gist.github.com/the42/1956518
func CompressFilter(in *In) bool {
	if in.hijacked {
		return true
	}
	if !strings.Contains(in.R.Header.Get("Accept-Encoding"), "gzip") || !in.App.Config.GZipDynamic {
		return true
	}
	in.W.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(in.W)
	gzr := &gzipRespWriter{gz, in.W}
	in.W = gzr
	in.closers = append(in.closers, gz)
	return true
}

func ServedByProxyFilter(in *In) bool {
	if v := in.R.Header.Get("X-Real-IP"); v != "" {
		in.R.RemoteAddr = v
	} else if v := in.R.Header.Get("X-Forwarded-For"); v != "" {
		in.R.RemoteAddr = v
	}
	return true
}

func NoCacheFilter(in *In) bool {
	hh := in.W.Header()
	hh.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	hh.Set("Pragma", "no-cache")
	hh.Set("Expires", "0")
	return true
}
