package main

import (
	"golang.org/x/net/http2"
	"crypto/tls"
	"net"
	"net/http"
	"fmt"
	"io/ioutil"
)

const  MeshServerAddr = "127.0.0.1:2045"

func main(){
	tr := &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(netw, addr)
		},
	}
	
	httpClient := http.Client{Transport: tr}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/", MeshServerAddr), nil)
	req.Header.Add("service", "tst")
	resp, err := httpClient.Do(req)
	
	if err != nil {
		fmt.Printf("[CLIENT]receive err %s", err)
		fmt.Println()
		return
	}
	
	fmt.Println(resp.Header)
	fmt.Println("resp.StatusCode:",resp.StatusCode)
	
	
	
	defer resp.Body.Close()
	
	
	
	body, err := ioutil.ReadAll(resp.Body)
	
	if err != nil {
		fmt.Printf("[CLIENT]receive err %s", err)
		fmt.Println()
		return
	}
	
	fmt.Printf("[CLIENT]receive data %s", body)
	fmt.Println()
}