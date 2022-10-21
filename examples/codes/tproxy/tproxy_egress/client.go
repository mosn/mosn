package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
)

func main() {

	avgs := os.Args

	localPort := 12345
	remoteAddr := "172.18.0.2:8080"

	for i, arg := range avgs {
		switch i {
		case 0:
			continue
		case 1:
			localPort, _ = strconv.Atoi(arg)
		case 2:
			remoteAddr = arg
		}
	}

	netAddr := &net.TCPAddr{Port: localPort}
	dialer := &net.Dialer{
		LocalAddr: netAddr,
	}

	tr := &http.Transport{
		Proxy:       http.ProxyFromEnvironment,
		DialContext: dialer.DialContext,
	}

	client := &http.Client{
		Transport: tr,
	}

	url := fmt.Sprintf("http://%s", remoteAddr)
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}

	read := make([]byte, 1024)
	resp.Body.Read(read)
	defer resp.Body.Close()

	fmt.Printf("%s", read)
}
