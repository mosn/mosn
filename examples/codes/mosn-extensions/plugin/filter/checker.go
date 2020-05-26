package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/plugin/proto"
)

type checker struct {
	config map[string]string
}

func (c *checker) Call(request *proto.Request) (*proto.Response, error) {
	header := request.Header
	for k, v := range c.config {
		value, ok := header[k]
		if !ok || value != v {
			return &proto.Response{
				Status: -1,
			}, nil
		}
	}
	return &proto.Response{}, nil
}

func main() {
	conf := flag.String("c", "", "-c config.json")
	flag.Parse()
	m := make(map[string]string)
	b, err := ioutil.ReadFile(*conf)
	if err == nil {
		json.Unmarshal(b, &m)
	}
	log.DefaultLogger.Infof("configfile: %s get config: %v", *conf, m)
	plugin.Serve(&checker{m})
}
