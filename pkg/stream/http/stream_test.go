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
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
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
					conn: network.NewClientConnection(nil, 0, nil, remoteAddr, nil),
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
					conn: network.NewClientConnection(nil, 0, nil, remoteAddr, nil),
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
			"Args":                            "Hello, world!",
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
					conn: network.NewClientConnection(nil, 0, nil, remoteAddr, nil),
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

func Test_clientStream_CheckReasonError(t *testing.T) {
	remoteAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12200")

	csc := &clientStreamConnection{
		streamConnection: streamConnection{
			conn: network.NewClientConnection(nil, 0, nil, remoteAddr, nil),
		},
	}

	res, ok := csc.CheckReasonError(true, api.RemoteClose)
	if res != types.UpstreamReset || ok {
		t.Errorf("csc.CheckReasonError(true, types.RemoteClose) got %v , want %v", res, types.UpstreamReset)
	}

	res, ok = csc.CheckReasonError(true, api.OnConnect)
	if res != types.StreamConnectionSuccessed || !ok {
		t.Errorf("csc.CheckReasonError(true, types.OnConnect) got %v , want %v", res, types.StreamConnectionSuccessed)
	}

}

func TestHeaderSize(t *testing.T) {
	// Only request line, do not add the end of request '\r\n\r\n' identification.
	requestSmall := []byte("HEAD / HTTP/1.1\r\nHost: test.com\r\nCookie: key=1234")
	requestLarge := []byte("HEAD / HTTP/1.1\r\nHost: test.com\r\nCookie: key=12345")
	testAddr := "127.0.0.1:11345"
	l, err := net.Listen("tcp", testAddr)
	if err != nil {
		t.Logf("listen error %v", err)
		return
	}
	defer l.Close()

	rawc, err := net.Dial("tcp", testAddr)
	if err != nil {
		t.Errorf("net.Dial error %v", err)
		return
	}

	connection := network.NewServerConnection(context.Background(), rawc, nil)
	proxyGeneralExtendConfig := v2.ProxyGeneralExtendConfig{
		MaxHeaderSize: len(requestSmall),
	}

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyProxyGeneralConfig, proxyGeneralExtendConfig)
	ssc := newServerStreamConnection(ctx, connection, nil)
	if ssc == nil {
		t.Errorf("newServerStreamConnection failed!")
	}

	// test the header size is within the limit
	buf := buffer.GetIoBuffer(len(requestSmall))
	buf.Write(requestSmall)
	ssc.Dispatch(buf)
	// if it exceeds the limit size of the header, the connection is closed immediately
	if connection.State() == api.ConnClosed {
		t.Errorf("requestSmall header size does not exceed limit!")
	}

	rawc, err = net.Dial("tcp", testAddr)
	if err != nil {
		t.Errorf("net.Dial error %v", err)
		return
	}
	connection = network.NewServerConnection(context.Background(), rawc, nil)
	ssc = newServerStreamConnection(ctx, connection, nil)
	if ssc == nil {
		t.Errorf("newServerStreamConnection failed!")
	}

	// test the header size exceeds the limit
	buf = buffer.GetIoBuffer(len(requestLarge))
	buf.Write(requestLarge)
	ssc.Dispatch(buf)
	if connection.State() != api.ConnClosed {
		t.Errorf("requestLarge header size should exceed limit!")
	}
}

func TestAppendData(t *testing.T) {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	responseBytes := randomBytes(t, rand.Intn(1024)+1024*1024*4, rand)
	if responseBytes == nil || len(responseBytes) == 0 {
		t.Fatal("randomBytes failed!")
	}

	buf := buffer.NewIoBufferBytes(responseBytes)

	var s serverStream
	var hb httpBuffers
	s.stream = stream{
		request:  &hb.serverRequest,
		response: &hb.serverResponse,
	}

	s.AppendData(context.Background(), buf, false)

	if bytes.Compare(s.response.Body(), responseBytes) != 0 {
		t.Errorf("server AppendData failed")
	}
}

func convertHeader(payload protocol.CommonHeader) http.RequestHeader {
	header := http.RequestHeader{&fasthttp.RequestHeader{}, nil}

	for k, v := range payload {
		header.Set(k, v)
	}

	return header
}

func randomBytes(t *testing.T, n int, rand *rand.Rand) []byte {
	r := make([]byte, n)
	if _, err := rand.Read(r); err != nil {
		t.Fatal("randomBytes rand.Read failed: " + err.Error())
		return nil
	}
	return r
}
