package main

import (
	"log"
	"os"
)

var logger *log.Logger

func init() {
	f, err := os.OpenFile("thrift-codec.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}

	logger = log.New(f, "", log.LstdFlags)
}
