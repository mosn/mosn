package main

import (
	"fmt"
	"os"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/plugin/proto"
)

type filter struct{}

func (s *filter) Call(request *proto.Request) (*proto.Response, error) {
	header := request.GetHeader()
	body := request.GetBody()
	trailer := request.GetTrailer()
	log.DefaultLogger.Infof("filter header:%d, body:%d, trailer:%d", len(header), len(body), len(trailer))

	response := new(proto.Response)
	response.Status = 1
	return response, nil
}

func main() {
	err := log.InitDefaultLogger(plugin.GetLogPath()+"pluginfilter.log", log.INFO)
	if err != nil {
		fmt.Fprintln(os.Stderr, "filter log error:", err)
	}

	plugin.Serve(&filter{})
}
