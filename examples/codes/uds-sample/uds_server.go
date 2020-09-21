package main

import (
	"fmt"
	"net"
	"sync"
	"syscall"
)

const (
	MaxSize = 1024
)

var bufPool = sync.Pool{New: func() interface{} { return make([]byte, MaxSize) }}

func main() {

	path := "/tmp/server-proxy.sock"
	syscall.Unlink(path)
	ls, err := net.Listen("unix", path)
	if err != nil {
		fmt.Printf("Failed to listen, %s", err)
		return
	}

	fmt.Println("Listening on uds model  ...")

	for {
		c, err := ls.Accept()
		if err != nil {
			fmt.Println("Could not accept connection , err:", err)
			continue
		}
		go func(conn net.Conn) {
			for {
				buffer := bufPool.Get().([]byte)
				n, err := conn.Read(buffer)
				if err != nil {
					fmt.Println("Read from upstream err:", err)
					conn.Close()
					return
				}
				echo := "echo: " + string(buffer[:n])
				_, err = conn.Write([]byte(echo))
				bufPool.Put(buffer)
				if err != nil {
					fmt.Println("Write to upstream err:", err)
					conn.Close()
					return
				}
			}

		}(c)

	}
}
