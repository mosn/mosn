// Copyright (c) 2015 Asim Aslam.
// Copyright (c) 2016 ~ 2018, Alex Stocks.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
	"github.com/AlexStocks/dubbogo/common"
	"github.com/AlexStocks/dubbogo/registry"
	"github.com/AlexStocks/dubbogo/transport"
)

type Options struct {
	Codecs    map[string]codec.NewCodec
	Registry  registry.Registry
	Transport transport.Transport

	ConfList        []registry.ServerConfig
	ServiceConfList []registry.ServiceConfig
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

func newOptions(opt ...Option) Options {
	opts := Options{
		Codecs: make(map[string]codec.NewCodec),
	}

	for _, o := range opt {
		o(&opts)
	}

	if opts.Registry == nil {
		panic("server.Options.Registry is nil")
	}

	if opts.Transport == nil {
		panic("server.Options.Transport is nil")
	}

	return opts
}

// Codec to use to encode/decode requests for a given content type
func Codec(codecs map[string]codec.NewCodec) Option {
	return func(o *Options) {
		o.Codecs = codecs
	}
}

// Registry used for discovery
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// Transport mechanism for communication e.g http, rabbitmq, etc
func Transport(t transport.Transport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

func ConfList(confList []registry.ServerConfig) Option {
	return func(o *Options) {
		o.ConfList = confList
		for i := 0; i < len(o.ConfList); i++ {
			o.ConfList[i].IP, _ = common.GetLocalIP(o.ConfList[i].IP)
		}
	}
}

func ServiceConfList(confList []registry.ServiceConfig) Option {
	return func(o *Options) {
		o.ServiceConfList = confList
	}
}
