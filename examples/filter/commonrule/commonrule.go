package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/alipay/sofa-mosn/pkg/config"
	_ "github.com/alipay/sofa-mosn/pkg/filter/network/proxy"
	"github.com/alipay/sofa-mosn/pkg/mosn"
	//_ "gitlab.alipay-inc.com/ant-mesh/mosn/pkg/filter/stream/commonrule"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http"
	`github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/metrix`
	_ `github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule`
)

func ServeHTTP1(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[UPSTREAM]receive request %s", r.URL)
	fmt.Println()

	w.Header().Set("Content-Type", "text/plain")

	fmt.Fprintf(w, "Method: %s\n", r.Method)
	fmt.Fprintf(w, "Protocol: %s\n", r.Proto)
	fmt.Fprintf(w, "Host: %s\n", r.Host)
	fmt.Fprintf(w, "RemoteAddr: %s\n", r.RemoteAddr)
	fmt.Fprintf(w, "RequestURI: %q\n", r.RequestURI)
	fmt.Fprintf(w, "URL: %#v\n", r.URL)
	fmt.Fprintf(w, "Body.ContentLength: %d (-1 means unknown)\n", r.ContentLength)
	fmt.Fprintf(w, "Close: %v (relevant for HTTP/1 only)\n", r.Close)
	fmt.Fprintf(w, "TLS: %#v\n", r.TLS)
	fmt.Fprintf(w, "\nHeaders:\n")

	r.Header.Write(w)
}

func main() {
	http.HandleFunc("/", ServeHTTP1)
	go http.ListenAndServe("127.0.0.1:8080", nil)
	//change to your local path
	configPath := "/Users/loull/go/src/gitlab.alipay-inc.com/ant-mesh/mosn/examples/filter/commonrule.json"
	conf := config.Load(configPath)
	go mosn.Start(conf, "", "")
	fmt.Println("wait server and mosn start....")
	time.Sleep(5 * time.Second)


	ticker := metrix.NewTicker(func() {
		client2 := &http.Client{}
		url2 := "http://127.0.0.1:2045/serverlist?aa=a1&aa=a2&bb=b"
		request2, _ := http.NewRequest("GET", url2, nil)
		request2.Header.Add("Cookie", "mycookies")
		request2.Header.Add("User-Agent", "myagent")
		request2.Header.Add("operation-type", "myOp")
		resp2, _ := client2.Do(request2)
		resp2.Body.Close()
		fmt.Println("response status: ", resp2.StatusCode)
	})
	ticker.Start(time.Millisecond * 100)
	time.Sleep(5* time.Second)
	ticker.Stop()

	time.Sleep(30 * time.Second)
	fmt.Println("end ")
}
