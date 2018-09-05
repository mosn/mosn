package main

import (
	"fmt"
)

import (
	"github.com/AlexStocks/goext/io/kv/disk"
)

func main() {
	d := gxdiskv.New(gxdiskv.Options{
		BasePath:     "my-gxdiskv-data-directory",
		CacheSizeMax: 1024 * 1024, // 1MB
	})

	key := "alpha"
	if err := d.Write(key, []byte{'1', '2', '3'}); err != nil {
		panic(err)
	}
	return

	value, err := d.Read(key)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", value)

	if err := d.Erase(key); err != nil {
		panic(err)
	}
}
