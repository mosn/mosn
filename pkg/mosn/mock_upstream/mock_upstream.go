package main

import (
	"fmt"
	"golang.org/x/net/http2"
	"log"
	"net"
	"net/http"
	"time"
)

const RealServerAddr = "127.0.0.1:23456"


func main() {

	go func() {
		// upstream
		server := &http.Server{
			Addr:         RealServerAddr,
			Handler:      &serverHandler{},
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		}
		s2 := &http2.Server{
			IdleTimeout: 1 * time.Minute,
		}

		http2.ConfigureServer(server, s2)
		l, err := net.Listen("tcp", RealServerAddr)
		if err != nil {
			log.Fatalln("listen error:", err)
		}
		defer l.Close()

		for {
			rwc, err := l.Accept()
			if err != nil {
				fmt.Println("accept err:", err)
				continue
			}
			fmt.Println("Accept:",rwc.RemoteAddr().String())
			go s2.ServeConn(rwc, &http2.ServeConnOpts{BaseConfig: server})
		}
	}()

	select {
	case <-time.After(time.Second * 120):
		fmt.Println("[UPSTREAM]closing..")
	}
}

type serverHandler struct{}

func (sh *serverHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ShowRequestInfoHandler(w, req)
}

func ShowRequestInfoHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[UPSTREAM]receive request %s", r.URL)
	fmt.Println()

	w.Header().Set("Content-Type", "text/plain")

	for k, _ := range r.Header {
		w.Header().Set(k, r.Header.Get(k))
	}

	fmt.Fprintf(w, "Method: %s\n", r.Method)
	fmt.Fprintf(w, "Protocol: %s\n", r.Proto)
	fmt.Fprintf(w, "Host: %s\n", r.Host)
	fmt.Fprintf(w, "RemoteAddr: %s\n", r.RemoteAddr)
	fmt.Fprintf(w, "RequestURI: %q\n", r.RequestURI)
	fmt.Fprintf(w, "URL: %#v\n", r.URL)
	fmt.Fprintf(w, "Body.ContentLength: %d (-1 means unknown)\n", r.ContentLength)
	fmt.Fprintf(w, "Close: %v (relevant for HTTP/1 only)\n", r.Close)
	fmt.Fprintf(w, "TLS: %#v\n", r.TLS)
	fmt.Fprintf(w, "\nHeaders:\n")

	r.Header.Write(w)
}
