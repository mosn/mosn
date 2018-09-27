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

package client

import (
	"github.com/AlexStocks/dubbogo/registry"
)

type rpcRequest struct {
	group       string
	protocol    string
	version     string
	service     string
	method      string
	args        interface{}
	contentType string
	opts        RequestOptions
}

func newRPCRequest(group, protocol, version, service, method string, args interface{},
	contentType string, reqOpts ...RequestOption) Request {

	var opts RequestOptions
	for _, o := range reqOpts {
		o(&opts)
	}

	return &rpcRequest{
		group:       group,
		protocol:    protocol,
		version:     version,
		service:     service,
		method:      method,
		args:        args,
		contentType: contentType,
		opts:        opts,
	}
}

func (r *rpcRequest) Protocol() string {
	return r.protocol
}

func (r *rpcRequest) Version() string {
	return r.version
}

func (r *rpcRequest) ContentType() string {
	return r.contentType
}

func (r *rpcRequest) ServiceConfig() registry.ServiceConfigIf {
	return &registry.ServiceConfig{
		Protocol: r.protocol,
		Service:  r.service,
		Group:    r.group,
		Version:  r.version,
	}
}

func (r *rpcRequest) Method() string {
	return r.method
}

func (r *rpcRequest) Args() interface{} {
	return r.args
}

func (r *rpcRequest) Stream() bool {
	return r.opts.Stream
}

func (r *rpcRequest) Options() RequestOptions {
	return r.opts
}
