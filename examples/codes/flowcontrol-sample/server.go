package main

import (
	"net/http"
)

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success"))
}

func main() {
	http.HandleFunc("/test", ServeHTTP)
	http.ListenAndServe("127.0.0.1:8080", nil)
}
