package main

import (
	"flag"
	"fmt"
	"github.com/gabstv/goboots"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var cmdStatic = &Command{
	UsageLine: "static [path] [[args]]",
	Short:     "Starts a static server using [path] as root.",
	Long: `
Starts a static server using [path] as root.

Flags:
	listen
        -listen=":8080" | Listen at 0.0.0.0:8080
                        | ip:port
                        | default: ":80"
    listen-tls
        -listen-tls=":8443" | Listen at 0.0.0.0:8443
                            | ip:port
                            | default: ":443"
    tls-cert
        -tls-cert=cert.pem
    tls-key
        -tls-key=key.pem
`,
}

func init() {
	cmdStatic.Run = staticApp
}

type staticServer struct {
	path    string
	indexes []string
}

//TODO: caching, CORS

func (s *staticServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	addr := r.Header.Get("X-Real-IP")
	if addr == "" {
		addr = r.Header.Get("X-Forwarded-For")
		if addr == "" {
			addr = r.RemoteAddr
		}
	}

	niceurl := r.URL.Path
	fdir := goboots.FormatPath(s.path + "/" + niceurl)
	//
	info, err := os.Stat(fdir)
	if os.IsNotExist(err) {
		http.Error(w, "not found", 404)
		log.Println(addr, "[R]", r.URL.String(), 404, time.Since(start))
		return
	}
	if info.IsDir() {
		// resolve static index files
		for _, v := range s.indexes {
			j := filepath.Join(fdir, v)
			if _, err2 := os.Stat(j); err2 == nil {
				http.ServeFile(w, r, j)
				log.Println(addr, "[R]", r.URL.String(), "OK", time.Since(start))
				return
			}
		}
		http.Error(w, "forbidden", 403)
		log.Println(addr, "[R]", r.URL.String(), 403, time.Since(start))
		return
	}
	//
	http.ServeFile(w, r, fdir)
	log.Println(addr, "[R]", r.URL.String(), "OK", time.Since(start))
}

func staticApp(args []string) {
	if len(args) < 1 {
		errorf("No path specified.\nRun 'goboots help static' for usage.\n")
	}
	path := args[0]
	if path == "." {
		path = cwd
	}
	fs := flag.NewFlagSet("StaticFlags", flag.ContinueOnError)
	var listen, listen_tls, tls_cert, tls_key, index_files string

	fs.StringVar(&listen, "listen", ":80", `-listen="ip:port"`)
	fs.StringVar(&listen_tls, "listen-tls", ":443", `-listen-tls="ip:port"`)
	fs.StringVar(&tls_cert, "tls-cert", "", `-tls-cert="file.pem"`)
	fs.StringVar(&tls_key, "tls-key", "", `-tls-key="file.pem"`)
	fs.StringVar(&index_files, "index-files", "index.html,index.json", `-index-files="index.html,index.md"`)
	fs.Parse(args[1:])
	log.Println("Static server started!")

	indexSplit := strings.Split(index_files, ",")

	server := &staticServer{}
	server.path = path
	server.indexes = indexSplit

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		errr := http.ListenAndServe(listen, server)
		fmt.Println("LISTEN ERROR", errr)
	}()
	if len(tls_key) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errr := http.ListenAndServeTLS(listen_tls, goboots.FormatPath(tls_cert), goboots.FormatPath(tls_key), server)
			fmt.Println("LISTEN (TLS) ERROR", errr)
		}()
	}
	wg.Wait()
}
