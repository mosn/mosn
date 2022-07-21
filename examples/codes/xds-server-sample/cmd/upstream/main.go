package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	var (
		upstreamPort uint
		hello        string
	)

	flag.UintVar(&upstreamPort, "upstream", 18080, "Upstream HTTP/1.1 port")
	flag.StringVar(&hello, "message", "Default message", "Message to send in response")
	flag.Parse()

	log.Printf("upstream listening HTTP/1.1 on %d\n", upstreamPort)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(hello)); err != nil {
			log.Println(err)
		}
	})
	if err := http.ListenAndServe(fmt.Sprintf(":%d", upstreamPort), nil); err != nil {
		log.Println(err)
	}
}
