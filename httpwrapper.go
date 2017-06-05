// +build !windows

package goboots

import (
	"crypto/tls"
	"net/http"

	"github.com/gabstv/endless"
	"golang.org/x/crypto/acme/autocert"
)

func listenAndServe(addr string, handler http.Handler) error {
	return http.ListenAndServe(addr, handler)
}

func listenAndServeTLS(addr, certFile, keyFile string, handler http.Handler) error {
	return http.ListenAndServeTLS(addr, certFile, keyFile, handler)
}

func listenAndServeGracefully(addr string, handler http.Handler) error {
	return endless.ListenAndServe(addr, handler)
}

func listenAndServeTLSGracefully(addr, certFile, keyFile string, handler http.Handler) error {
	return endless.ListenAndServeTLS(addr, certFile, keyFile, handler)
}

func autocertServerTLS(addrtls string, whitelist []string, app *App) error {
	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(whitelist...),
	}
	s := &http.Server{
		Addr:      addrtls,
		TLSConfig: &tls.Config{GetCertificate: m.GetCertificate},
	}
	return s.ListenAndServeTLS("", "")
}

func autocertServerTLSGracefully(addrtls string, whitelist []string, app *App) error {
	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(whitelist...),
	}
	s := endless.NewServer(addrtls, app.entryHTTPS)
	s.Addr = addrtls
	s.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}
	return s.ListenAndServeTLS("", "")
}
