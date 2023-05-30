package main

import (
	"fmt"
	"net/http"
	"time"
)

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[UPSTREAM]receive request %s", r.URL)
	fmt.Println()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Transfer-Encoding", "chunked")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	for i := 1; i <= 1000; i++ {
		fmt.Fprintf(w, "data: {\"id\":\"test_id\",\"object\":\"test.chunk\",\"created\":1685430109,\"index\":%d,\"finish_reason\":null}]} \n\n", i) // 发送消息
		flusher.Flush()
		time.Sleep(10 * time.Millisecond)
	}
}

func main() {
	http.HandleFunc("/", ServeHTTP)
	http.ListenAndServe("127.0.0.1:8888", nil)
}
