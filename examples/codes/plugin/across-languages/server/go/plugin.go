package main

import (
	"errors"

	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/plugin/proto"
)

type filter struct{}

func (s *filter) Call(request *proto.Request) (*proto.Response, error) {
	if string(request.GetBody()) != "hello" {
		return nil, errors.New("request body error")
	}

	response := new(proto.Response)
	response.Body = []byte("hello go")
	response.Status = 1
	return response, nil
}

func main() {
	plugin.Serve(&filter{})
}
