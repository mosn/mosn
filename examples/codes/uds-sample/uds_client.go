package main

import (
	"fmt"
	"net"
	"sync"
)

const (
	MaxSize = 1024
)

var bufPool = sync.Pool{New: func() interface{} { return make([]byte, MaxSize) }}


func main() {

	ls, err := net.Listen("unix", "/tmp/unix.sock")
	if err != nil {
		fmt.Printf("Failed to listen, %s", err)
		return
	}

	fmt.Println("Listening on uds model  ...")

	for {
		conn, err := ls.Accept()
		if err != nil {
			fmt.Println("Could not accept connection , err:", err)
			continue
		}
		buffer := bufPool.Get().([]byte)
		_, err = conn.Read(buffer)
		if err != nil {
			fmt.Println("Read from upstream err:", err)
			conn.Close()
			continue
		}
		echo := "echo: " + string(buffer)
		_, err = conn.Write([]byte(echo))
		bufPool.Put(buffer)
		if err != nil {
			fmt.Println("Write to upstream err:", err)
			conn.Close()
			continue
		}
	}
}
