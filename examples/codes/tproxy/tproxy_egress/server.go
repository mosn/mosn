package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
)

func param(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	addr, _ := ctx.Value("addr").(string)
	fmt.Fprintln(res, addr)
}

func connContext(ctx context.Context, conn net.Conn) context.Context {
	local := conn.LocalAddr()
	romote := conn.RemoteAddr()

	addr := fmt.Sprintf("[server] server ip: %s; client ip: %s", local.String(), romote.String())
	fmt.Println(addr)

	c := context.WithValue(ctx, "addr", addr)
	return c
}

func main() {

	args := os.Args

	addr := "0.0.0.0:8080"
	if len(args) > 1 {
		addr = args[1]
	}

	server := http.Server{
		Addr:        addr,
		ConnContext: connContext,
	}

	http.HandleFunc("/", param)
	server.ListenAndServe()
}
