package gxpprof

import (
	"fmt"
	"time"
)

// go test -v -run Gops
func ExampleGops() {
	// err := Gops("127.0.0.1:60000")
	err := Gops("")
	if err != nil {
		fmt.Printf("error:%#v\n", err)
		return
	}
	time.Sleep(60e9)
	// Output:
}
