package main

import (
	"fmt"
	"net"
)

func main() {

	addr := "172.18.0.2:80"
	d := net.Dialer{}
	conn, err := d.Dial("tcp", addr)

	if err != nil {
		return
	}

	defer conn.Close()

	local := conn.LocalAddr()
	romote := conn.RemoteAddr()

	stream := fmt.Sprintf("[client] client ip: %s; server ip: %s", local.String(), romote.String())
	fmt.Println(string(stream))

	write := fmt.Sprintf("GET / HTTP/1.1\r\n\r\n")

	conn.Write([]byte(write))

	response := make([]byte, 1024)
	conn.Read(response)
	//response, _ := ioutil.ReadAll(conn)
	fmt.Print(string(response))

}
