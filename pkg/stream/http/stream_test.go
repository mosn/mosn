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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"mosn.io/api"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

func TestBuildUrlFromCtxVar(t *testing.T) {
	testcases := []struct {
		path    string
		pathOri string
		want    string
	}{
		{
			"/home/sa%6dple",
			"/home/sa%25%36%64ple",
			"/home/sa%25%36%64ple",
		},
		{
			"/home/sa%6dple",
			"/home/sample%20",
			"/home/sa%256dple",
		},
		{
			"/home/sample ",
			"/home/sample%20",
			"/home/sample%20",
		},
		{
			"/home/sample ",
			"/home/sample ",
			"/home/sample ",
		},
		{
			"/home/sam ple",
			"/home/sam ple",
			"/home/sam ple",
		},
		{
			"/home/sample",
			"/home/%2Fsample",
			"/home/%2Fsample",
		},
	}
	for _, tc := range testcases {
		ctx := variable.NewVariableContext(context.Background())
		variable.SetString(ctx, types.VarPath, tc.path)
		variable.SetString(ctx, types.VarPathOriginal, tc.pathOri)
		assert.Equal(t, buildUrlFromCtxVar(ctx), tc.want)
	}
}

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
					conn: network.NewClientConnection(0, nil, remoteAddr, nil),
				},
			},
		},
	}

	queryString := "name=biz&passwd=bar"

	path := "/pic"

	headers := http.RequestHeader{&fasthttp.RequestHeader{}}

	wantedURI := []string{
		"/pic?name=biz&passwd=bar",
	}

	ctx := variable.NewVariableContext(context.Background())
	url := &fasthttp.URI{}
	url.SetPath(path)
	url.SetQueryString(queryString)
	for i := 0; i < len(ClientStreamsMocked); i++ {
		injectCtxVarFromProtocolHeaders(ctx, headers, url)
		ClientStreamsMocked[i].AppendHeaders(ctx, headers, false)
		if string(ClientStreamsMocked[i].request.Header.RequestURI()) != wantedURI[i] {
			t.Errorf("clientStream AppendHeaders() error, uri:%s", string(ClientStreamsMocked[i].request.Header.RequestURI()))
		}
	}
}

func TestStreamConnectionDispatch(t *testing.T) {
	streamConnectionMocked := &streamConnection{
		bufChan:    make(chan buffer.IoBuffer),
		endRead:    make(chan struct{}),
		connClosed: make(chan bool, 1),
	}
	streamConnectionMocked.br = bufio.NewReaderSize(streamConnectionMocked, defaultMaxHeaderSize)
	httpTestResponseHeader := "HTTP/1.1 200 OK\r\nDate: Fri, 13 Nov 2020 09:27:39 GMT\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: 12\r\n\r\n"
	httpTestResponseBody := `hello`
	httpTestResponseBody2 := ` world!`
	go streamConnectionMocked.Dispatch(buffer.NewIoBufferString(httpTestResponseHeader))
	// wait Dispatch ready
	time.Sleep(time.Second)
	go streamConnectionMocked.Dispatch(buffer.NewIoBufferString(httpTestResponseBody))

	time.Sleep(time.Second)
	go streamConnectionMocked.Dispatch(buffer.NewIoBufferString(httpTestResponseBody2))

	response := fasthttp.AcquireResponse()
	// wait Dispatch ready
	time.Sleep(time.Second)
	err := response.Read(streamConnectionMocked.br)
	if err != nil {
		t.Fatalf("http reponse read error: %v", err)
	}

	//t.Logf("Header: %v body: %v", response.Header.String(), string(response.Body()))
	if response.Header.ContentLength() != 12 {
		t.Errorf("want length: %v get: %v", 12, response.Header.ContentLength())
	}

	if string(response.Body()) != httpTestResponseBody+httpTestResponseBody2 {
		t.Errorf("want body: %v. get: %v", httpTestResponseBody+httpTestResponseBody2, string(response.Body()))
	}
}

func BenchmarkStreamConnection_Dispatch(b *testing.B) {
	streamConnectionMocked := &streamConnection{
		bufChan:    make(chan buffer.IoBuffer),
		endRead:    make(chan struct{}),
		connClosed: make(chan bool, 1),
	}
	streamConnectionMocked.br = bufio.NewReaderSize(streamConnectionMocked, defaultMaxHeaderSize)
	httpTestResponse := "HTTP/1.1 200 OK\r\nDate: Fri, 13 Nov 2020 09:27:39 GMT\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: 12\r\n\r\nhello world!"

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		go streamConnectionMocked.Dispatch(buffer.NewIoBufferString(httpTestResponse))

		response := fasthttp.AcquireResponse()
		// wait Dispatch ready
		time.Sleep(time.Second)
		err := response.Read(streamConnectionMocked.br)
		if err != nil {
			b.Fatalf("http reponse read error: %v", err)
		}
	}
	b.StopTimer()

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
					conn: network.NewClientConnection(0, nil, remoteAddr, nil),
				},
			},
		},
	}

	queryString := "name=biz&passwd=bar"

	path := "/pic"

	headers := []protocol.CommonHeader{
		{
			"Args": "Hello, world!",
		},
	}

	wantedURI := []string{
		"/pic?name=biz&passwd=bar",
	}
	ctx := variable.NewVariableContext(context.Background())
	url := &fasthttp.URI{}
	url.SetPath(path)
	url.SetQueryString(queryString)
	for i := 0; i < len(ClientStreamsMocked); i++ {
		injectCtxVarFromProtocolHeaders(ctx, convertHeader(headers[i]), url)
		ClientStreamsMocked[i].AppendHeaders(ctx, convertHeader(headers[i]), false)
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
					conn: network.NewClientConnection(0, nil, remoteAddr, nil),
				},
			},
		},
	}

	queryString := "name=biz&passwd=bar"

	path := "/pic"

	headers := []protocol.CommonHeader{
		{
			"Method": "com.alipay.test.rpc.sample",
		},
	}

	wantedURI := []string{
		"/pic?name=biz&passwd=bar",
	}

	ctx := variable.NewVariableContext(context.Background())
	url := &fasthttp.URI{}
	url.SetPath(path)
	url.SetQueryString(queryString)
	for i := 0; i < len(ClientStreamsMocked); i++ {
		injectCtxVarFromProtocolHeaders(ctx, convertHeader(headers[i]), url)
		ClientStreamsMocked[i].AppendHeaders(ctx, convertHeader(headers[i]), false)
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
	header := http.RequestHeader{&fasthttp.RequestHeader{}}
	uri := fasthttp.AcquireURI()

	// headers.Get return
	// 1. "", true means it do has corresponding entry with value ""
	// 2. "", false means no entry match the key
	// test if  recycle would change the semantic

	// mock first request arrive, with no query string
	header.SetMethod("GET")
	uri.SetHost("first.test.com")
	uri.SetPath("/first")

	ctx := variable.NewVariableContext(context.Background())
	injectCtxVarFromProtocolHeaders(ctx, header, uri)

	// mock request send
	FillRequestHeadersFromCtxVar(ctx, header, remoteAddr)

	fmt.Println("first request header sent:", header)

	// simulate recycle
	header.Reset()
	uri.Reset()

	// mock second request arrive, with query string
	ctx = variable.NewVariableContext(context.Background())

	header.SetMethod("GET")
	uri.SetHost("second.test.com")
	uri.SetPath("/second")
	uri.SetQueryString("meaning=less")

	injectCtxVarFromProtocolHeaders(ctx, header, uri)
	// mock request send
	FillRequestHeadersFromCtxVar(ctx, header, remoteAddr)

	fmt.Println("second request header sent:", header)

	// simulate recycle
	header.Reset()
	uri.Reset()

	ctx = variable.NewVariableContext(context.Background())
	// mock third request arrive, with no query string
	header.SetMethod("GET")
	uri.SetHost("third.test.com")
	uri.SetPath("/third")

	injectCtxVarFromProtocolHeaders(ctx, header, uri)
	// mock request send
	FillRequestHeadersFromCtxVar(ctx, header, remoteAddr)

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
			s.handleRequest(nil)
		})
	}
}

func Test_clientStream_CheckReasonError(t *testing.T) {
	remoteAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12200")

	csc := &clientStreamConnection{
		streamConnection: streamConnection{
			conn: network.NewClientConnection(0, nil, remoteAddr, nil),
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

func TestStreamConfigHandler(t *testing.T) {
	t.Run("test config", func(t *testing.T) {
		v := map[string]interface{}{
			"max_header_size":       1024,
			"max_request_body_size": 1024,
		}
		rv := streamConfigHandler(v)
		cfg, ok := rv.(StreamConfig)
		if !ok {
			t.Fatalf("config handler should returns an StreamConfig")
		}
		if !(cfg.MaxHeaderSize == 1024 &&
			cfg.MaxRequestBodySize == 1024) {
			t.Fatalf("unexpected config: %v", cfg)
		}
	})
	t.Run("test body size", func(t *testing.T) {
		v := map[string]interface{}{
			"max_request_body_size": 8192,
		}
		rv := streamConfigHandler(v)
		cfg, ok := rv.(StreamConfig)
		if !ok {
			t.Fatalf("config handler should returns an StreamConfig")
		}
		if cfg.MaxHeaderSize != defaultMaxHeaderSize {
			t.Fatalf("no header size configured, should use default header size but not: %d", cfg.MaxHeaderSize)
		}
	})
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

	httpConfig := map[string]interface{}{
		"max_header_size": len(requestSmall),
	}
	proxyGeneralExtendConfig := make(map[api.ProtocolName]interface{})
	proxyGeneralExtendConfig[protocol.HTTP1] = streamConfigHandler(httpConfig)

	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableProxyGeneralConfig, proxyGeneralExtendConfig)

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
	header := http.RequestHeader{&fasthttp.RequestHeader{}}

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
