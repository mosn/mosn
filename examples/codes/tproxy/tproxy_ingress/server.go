package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	listen, _ := net.Listen("tcp", "0.0.0.0:8080")

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	go func() {
		response := make([]byte, 2048)
		conn.Read(response)
		//response, _ := ioutil.ReadAll(conn)
		fmt.Println(string(response))
	}()

	local := conn.LocalAddr()
	romote := conn.RemoteAddr()

	stream := fmt.Sprintf("[server] server ip: %s; client ip: %s", local.String(), romote.String())
	fmt.Println(string(stream))

	write := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: 10\r\n\r\n")
	conn.Write([]byte(write))

	time.Sleep(time.Millisecond * 100)
}
