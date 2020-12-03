package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

var clientBufPool = sync.Pool{New: func() interface{} { return make([]byte, 1024) }}

func main() {

	conn, err := net.Dial("unix", "/tmp/client-proxy.sock")
	if err != nil {
		fmt.Printf("Failed to connect, %s", err)
		return
	}

	req := "hello, MOSN"
	for {
		_, err := conn.Write([]byte(req))
		if err != nil {
			fmt.Println("Write to upstream err:", err)
			conn.Close()
			return
		}
		buffer := clientBufPool.Get().([]byte)
		n, err := conn.Read(buffer)
		clientBufPool.Put(buffer)
		if err != nil {
			fmt.Println("Write to upstream err:", err)
			conn.Close()
			continue
		}
		fmt.Println(string(buffer[:n]))
		time.Sleep(time.Second)
	}
}
