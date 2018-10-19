package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
)

func request() {
	client := &http.Client{}
	url2 := "http://127.0.0.1:2045/serverlist?aa=a1&aa=a2&bb=b"
	request, _ := http.NewRequest("GET", url2, nil)
	request.Header.Add("Cookie", "mycookies")
	request.Header.Add("User-Agent", "myagent")
	request.Header.Add("operation-type", "myOp")
	resp2, _ := client.Do(request)
	resp2.Body.Close()
	fmt.Println("response status: ", resp2.StatusCode)
}

func main() {
	log.InitDefaultLogger("", log.DEBUG)
	t := flag.Bool("t", false, "-t")
	flag.Parse()
	for {
		for i := 0; i < 50; i++ {
			request()
			time.Sleep(100 * time.Millisecond)
		}
		time.Sleep(3 * time.Second)
		if !*t {
			return
		}
	}
}
