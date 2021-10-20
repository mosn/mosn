package main

import (
	"net/http"
)

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Header.Write(w)

}

func main() {
	http.HandleFunc("/", ServeHTTP)
	http.ListenAndServe("127.0.0.1:8080", nil)
}
