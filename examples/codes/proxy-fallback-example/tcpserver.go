package main

import (
	"fmt"
	"net"
)

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:9090")
	if err != nil {
		fmt.Println("fail to start tcp server")
		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("fail to accept tcp connection")
		}

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	fmt.Println("Accepted new connection.")
	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		size, err := conn.Read(buf)
		if err != nil {
			return
		}
		data := buf[:size]
		fmt.Println("Read new data from connection:", string(data))
		conn.Write(data)
	}
}
