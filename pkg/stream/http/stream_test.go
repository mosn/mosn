/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http

import (
	"testing"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/valyala/fasthttp"
)

func Test_clientStream_AppendHeaders(t *testing.T) {

	addr := "www.antfin.com"
	ClientStreamsMocked := []clientStream{
		{
			request: fasthttp.AcquireRequest(),
			wrapper: &clientStreamWrapper{
				client: &fasthttp.HostClient{
					Addr: addr,
				},
			},
		},
	}

	queryString := "name=biz&passwd=bar"

	path := "/pic"

	headers := []protocol.CommonHeader{
		{
			protocol.MosnHeaderQueryStringKey: queryString,
			protocol.MosnHeaderPathKey:        path,
		},
	}

	wantedURI := []string{
		"http://www.antfin.com/pic?name=biz&passwd=bar",
	}

	for i := 0; i < len(ClientStreamsMocked); i++ {
		ClientStreamsMocked[i].AppendHeaders(nil, headers[i], false)
		if len(headers[i]) != 0 && string(ClientStreamsMocked[i].request.Header.RequestURI()) != wantedURI[i] {
			t.Errorf("clientStream AppendHeaders() error")
		}
	}
}

func Test_header_capitalization(t *testing.T) {

	addr := "www.antfin.com"
	ClientStreamsMocked := []clientStream{
		{
			request: fasthttp.AcquireRequest(),
			wrapper: &clientStreamWrapper{
				client: &fasthttp.HostClient{
					Addr: addr,
				},
			},
		},
	}

	queryString := "name=biz&passwd=bar"

	path := "/pic"

	headers := []protocol.CommonHeader{
		{
			protocol.MosnHeaderQueryStringKey: queryString,
			protocol.MosnHeaderPathKey:        path,
			"Args": "Hello, world!",
		},
	}

	wantedURI := []string{
		"http://www.antfin.com/pic?name=biz&passwd=bar",
	}

	for i := 0; i < len(ClientStreamsMocked); i++ {
		ClientStreamsMocked[i].AppendHeaders(nil, headers[i], false)
		if len(headers[i]) != 0 && string(ClientStreamsMocked[i].request.Header.RequestURI()) != wantedURI[i] {
			t.Errorf("clientStream AppendHeaders() error")
		}

		if len(headers[i]) != 0 && ClientStreamsMocked[i].request.Header.Peek("args") != nil &&
			ClientStreamsMocked[i].request.Header.Peek("Args") == nil {
			t.Errorf("clientStream header capitalization error")
		}
	}
}

func Test_header_conflict(t *testing.T) {

	addr := "www.antfin.com"
	ClientStreamsMocked := []clientStream{
		{
			request: fasthttp.AcquireRequest(),
			wrapper: &clientStreamWrapper{
				client: &fasthttp.HostClient{
					Addr: addr,
				},
			},
		},
	}

	queryString := "name=biz&passwd=bar"

	path := "/pic"

	headers := []protocol.CommonHeader{
		{
			protocol.MosnHeaderQueryStringKey: queryString,
			protocol.MosnHeaderPathKey:        path,
			"Method":                          "com.alipay.test.rpc.sample",
		},
	}

	wantedURI := []string{
		"http://www.antfin.com/pic?name=biz&passwd=bar",
	}

	for i := 0; i < len(ClientStreamsMocked); i++ {
		ClientStreamsMocked[i].AppendHeaders(nil, headers[i], false)
		if len(headers[i]) != 0 && string(ClientStreamsMocked[i].request.Header.RequestURI()) != wantedURI[i] {
			t.Errorf("clientStream AppendHeaders() error")
		}

		if len(headers[i]) != 0 && string(ClientStreamsMocked[i].request.Header.Method()) == "com.alipay.test.rpc.sample" {
			t.Errorf("clientStream header key conflicts")
		}
	}
}

func Test_serverStream_handleRequest(t *testing.T) {
	type fields struct {
		stream           stream
		ctx              *fasthttp.RequestCtx
		connection       *serverStreamConnection
		responseDoneChan chan bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &serverStream{
				stream:           tt.fields.stream,
				ctx:              tt.fields.ctx,
				connection:       tt.fields.connection,
				responseDoneChan: tt.fields.responseDoneChan,
			}
			s.handleRequest()
		})
	}
}
