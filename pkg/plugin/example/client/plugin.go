package main

import (
	"fmt"
	"time"

	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/plugin/proto"
)

func main() {
	client, err := plugin.Register("plugin-server", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	response, err := client.Call(&proto.Request{
		Body: []byte("hello"),
	}, time.Second)

	if err != nil {
		fmt.Println(err)
		return
	}
	if string(response.Body) != "world" {
		fmt.Printf("faild! response body :%s\n", string(response.Body))
		return
	}
	fmt.Printf("success! response body: %s\n", string(response.Body))
}
