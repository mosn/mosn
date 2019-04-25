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

	"net"

	"bytes"
	"fmt"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/http"
	"github.com/valyala/fasthttp"
)

func Test_clientStream_AppendHeaders(t *testing.T) {
	streamMocked := stream{
		request: fasthttp.AcquireRequest(),
	}
	remoteAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12200")

	ClientStreamsMocked := []clientStream{
		{
			stream: streamMocked,
			connection: &clientStreamConnection{
				streamConnection: streamConnection{
					conn: network.NewClientConnection(nil, nil, remoteAddr, nil),
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
		"/pic?name=biz&passwd=bar",
	}

	for i := 0; i < len(ClientStreamsMocked); i++ {
		ClientStreamsMocked[i].AppendHeaders(nil, convertHeader(headers[i]), false)
		if len(headers[i]) != 0 && string(ClientStreamsMocked[i].request.Header.RequestURI()) != wantedURI[i] {
			t.Errorf("clientStream AppendHeaders() error, uri:%s", string(ClientStreamsMocked[i].request.Header.RequestURI()))
		}
	}
}

func Test_header_capitalization(t *testing.T) {
	remoteAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12200")

	streamMocked := stream{
		request: fasthttp.AcquireRequest(),
	}
	ClientStreamsMocked := []clientStream{
		{
			stream: streamMocked,
			connection: &clientStreamConnection{
				streamConnection: streamConnection{
					conn: network.NewClientConnection(nil, nil, remoteAddr, nil),
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
		"/pic?name=biz&passwd=bar",
	}

	for i := 0; i < len(ClientStreamsMocked); i++ {
		ClientStreamsMocked[i].AppendHeaders(nil, convertHeader(headers[i]), false)
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
	remoteAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12200")

	streamMocked := stream{
		request: fasthttp.AcquireRequest(),
	}
	ClientStreamsMocked := []clientStream{
		{
			stream: streamMocked,
			connection: &clientStreamConnection{
				streamConnection: streamConnection{
					conn: network.NewClientConnection(nil, nil, remoteAddr, nil),
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
		"/pic?name=biz&passwd=bar",
	}

	for i := 0; i < len(ClientStreamsMocked); i++ {
		ClientStreamsMocked[i].AppendHeaders(nil, convertHeader(headers[i]), false)
		if len(headers[i]) != 0 && string(ClientStreamsMocked[i].request.Header.RequestURI()) != wantedURI[i] {
			t.Errorf("clientStream AppendHeaders() error")
		}

		if len(headers[i]) != 0 && string(ClientStreamsMocked[i].request.Header.Method()) == "com.alipay.test.rpc.sample" {
			t.Errorf("clientStream header key conflicts")
		}
	}
}

func Test_internal_header(t *testing.T) {
	remoteAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12200")
	header := http.RequestHeader{&fasthttp.RequestHeader{}, nil}
	uri := fasthttp.AcquireURI()

	// headers.Get return
	// 1. "", true means it do has corresponding entry with value ""
	// 2. "", false means no entry match the key
	// test if  recycle would change the semantic

	// mock first request arrive, with no query string
	header.SetMethod("GET")
	uri.SetHost("first.test.com")
	uri.SetPath("/first")

	injectInternalHeaders(header, uri)

	// mock request send
	removeInternalHeaders(header, remoteAddr)

	fmt.Println("first request header sent:", header)

	// simulate recycle
	header.Reset()
	uri.Reset()

	// mock second request arrive, with query string
	header.SetMethod("GET")
	uri.SetHost("second.test.com")
	uri.SetPath("/second")
	uri.SetQueryString("meaning=less")

	injectInternalHeaders(header, uri)
	// mock request send
	removeInternalHeaders(header, remoteAddr)

	fmt.Println("second request header sent:", header)

	// simulate recycle
	header.Reset()
	uri.Reset()

	// mock third request arrive, with no query string
	header.SetMethod("GET")
	uri.SetHost("third.test.com")
	uri.SetPath("/third")

	injectInternalHeaders(header, uri)
	// mock request send
	removeInternalHeaders(header, remoteAddr)

	fmt.Println("third request header sent:", header)

	if bytes.Contains(header.RequestURI(), []byte("?")) {
		t.Errorf("internal header processing error")
	}
}

func Test_serverStream_handleRequest(t *testing.T) {
	type fields struct {
		stream           stream
		request          *fasthttp.Request
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
				connection:       tt.fields.connection,
				responseDoneChan: tt.fields.responseDoneChan,
			}
			s.handleRequest()
		})
	}
}

func convertHeader(payload protocol.CommonHeader) http.RequestHeader {
	header := http.RequestHeader{&fasthttp.RequestHeader{}, nil}

	for k, v := range payload {
		header.Set(k, v)
	}

	return header
}
