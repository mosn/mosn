package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println("[Rpc Server]: Listen error", err)
		return
	}
	fmt.Println("[RPC Server] Start Server...")
Serve:
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("[RPC Server] get connection error:", err)
			continue Serve
		}
		fmt.Println("[RPC Server] get connection :", conn.RemoteAddr())
		//read connection
		buf := make([]byte, 4*1024)
	Read:
		for {
			t := time.Now()
			conn.SetReadDeadline(t.Add(3 * time.Second))
			bytesRead, err := conn.Read(buf)
			if err != nil {
				if e, ok := err.(net.Error); ok && e.Timeout() {
					fmt.Println("[RPC Server] read error:", err)
					continue Read
				}
				fmt.Println("[RPC Server] read connection buffer failed")
				continue Serve
			}
			if bytesRead > 0 {
				fmt.Println("[RPC Server] read connection data :", string(buf[:bytesRead]))
				break Read
			}
		}
		fmt.Println("[RPC Server] write data")
		if _, err := conn.Write(boltV1ResBytes); err != nil {
			fmt.Println("[RPC Server] write response error", err)
		}
	}
}
