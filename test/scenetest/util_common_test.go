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

package tests

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/network/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/codec"
	"github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	"golang.org/x/net/http2"
)

//Common UpstreamServer
type ServeConn func(t *testing.T, conn net.Conn)
type UpstreamServer struct {
	Listener net.Listener
	Serve    ServeConn
	conns    []net.Conn
	mu       sync.Mutex
	t        *testing.T
	closed   bool
}

func NewUpstreamServer(t *testing.T, addr string, serve ServeConn) *UpstreamServer {
	//wait resource release
	time.Sleep(2 * time.Second)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen %s failed, error: %v\n", addr, err)
		return nil
	}
	return &UpstreamServer{
		Listener: l,
		conns:    []net.Conn{},
		mu:       sync.Mutex{},
		Serve:    serve,
		t:        t,
		closed:   false,
	}

}
func (s *UpstreamServer) GoServe() {
	go s.serve()
}
func (s *UpstreamServer) serve() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				s.t.Logf("Accept error: %v\n", err)
				continue
			}
			return
		}
		s.t.Logf("server %s Accept connection: %s\n", s.Listener.Addr().String(), conn.RemoteAddr().String())
		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()
		go s.Serve(s.t, conn)
	}
}

func (s *UpstreamServer) Close() {
	s.mu.Lock()
	if s.closed {
		return
	}
	s.t.Logf("server %s closed\n", s.Listener.Addr().String())
	s.Listener.Close()
	for _, conn := range s.conns {
		conn.Close()
	}
	s.closed = true
	s.mu.Unlock()
}

//Server Implement
type HTTP2Server struct {
	t      *testing.T
	Server *http2.Server
}

func (s *HTTP2Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.t.Logf("[server] Receive request\n")
	w.Header().Set("Content-Type", "text/plain")

	for k := range r.Header {
		w.Header().Set(k, r.Header.Get(k))
	}

	fmt.Fprintf(w, "\nRequestId:%s\n", r.Header.Get("Requestid"))

}
func (s *HTTP2Server) ServeConn(t *testing.T, conn net.Conn) {
	s.Server.ServeConn(conn, &http2.ServeConnOpts{Handler: s})
}

func NewUpstreamHTTP2(t *testing.T, addr string) *UpstreamServer {
	s := &HTTP2Server{
		t:      t,
		Server: &http2.Server{IdleTimeout: 1 * time.Minute},
	}
	return NewUpstreamServer(t, addr, s.ServeConn)
}

type HTTP2Response struct {
	re *regexp.Regexp
}

func (resp *HTTP2Response) Filter(data string, records sync.Map) {
	if resp.re == nil {
		resp.re = regexp.MustCompile("\nRequestId:[0-9]+\n")
	}
	bodys := strings.Split(
		strings.Trim(resp.re.FindString(data), "\n"), ":",
	)
	if len(bodys) == 2 {
		requestID := bodys[1]
		if _, ok := records.Load(requestID); ok {
			records.Delete(requestID)
		}
	}
}

//Http Server
//use in net/http/httptest
type HTTPServer struct {
	t    *testing.T
	name string
}

func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.t.Logf("server %s Receive request\n", s.name)
	w.Header().Set("Content-Type", "text/plain")
	for k := range r.Header {
		w.Header().Set(k, r.Header.Get(k))
	}
	fmt.Fprintf(w, "\nServerName:%s\n", s.name)
	fmt.Fprintf(w, "\nRequestId:%s\n", r.Header.Get("Requestid"))
}

type ReponseFilter interface {
	Filter(data string, records sync.Map)
}

//Rpc client
//Send Request Byte
type RPCClient struct {
	t              *testing.T
	conn           types.ClientConnection
	addr           string
	responseFilter ReponseFilter
	waitReponse    sync.Map
}

//types.ReadFilter
func (c *RPCClient) OnNewConnection() types.FilterStatus {
	return types.Continue
}
func (c *RPCClient) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}
func (c *RPCClient) OnData(buffer types.IoBuffer) types.FilterStatus {
	c.t.Logf("[client] receive data: \n")
	//c.t.Logf("%s\n", buffer.String())
	resp := buffer.String()
	c.responseFilter.Filter(resp, c.waitReponse)
	buffer.Reset()
	return types.Continue
}

//types.ConnectionEventListener
func (c *RPCClient) OnEvent(event types.ConnectionEvent) {}

func (c *RPCClient) Connect() error {
	stopChan := make(chan struct{})
	remoteAddr, _ := net.ResolveTCPAddr("tcp", c.addr)
	cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
	c.conn = cc
	cc.AddConnectionEventListener(c)
	cc.FilterManager().AddReadFilter(c)
	if err := cc.Connect(true); err != nil {
		//c.t.Logf("[client] connection failed\n")
		return err
	}
	return nil
}

var streamIDCounter uint32

func GetStreamID() uint32 {
	return atomic.AddUint32(&streamIDCounter, 1)
}

func (c *RPCClient) SendRequest(streamID uint32, req []byte) {
	c.conn.Write(buffer.NewIoBufferBytes(req))
	c.waitReponse.Store(fmt.Sprintf("%d", streamID), streamID)
}

//BoltV1 Client
//types.StreamReceiver
type BoltV1Client struct {
	t            *testing.T
	ClientID     string
	Codec        stream.CodecClient
	Waits        sync.Map
	conn         types.ClientConnection
	respCount    uint32
	requestCount uint32
}

func (c *BoltV1Client) Connect(addr string) error {
	stopChan := make(chan struct{})
	remoteAddr, _ := net.ResolveTCPAddr("tcp", addr)
	cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
	c.conn = cc
	if err := cc.Connect(true); err != nil {
		c.t.Logf("client[%s] connect to server error: %v\n", c.ClientID, err)
		return err
	}
	c.Codec = stream.NewCodecClient(context.Background(), protocol.SofaRPC, cc, nil)
	return nil
}
func (c *BoltV1Client) SendRequest() {
	id := GetStreamID()
	streamID := sofarpc.StreamIDConvert(id)
	requestEncoder := c.Codec.NewStream(streamID, c)
	headers := buildBoltV1Request(id)
	requestEncoder.AppendHeaders(headers, true)
	atomic.AddUint32(&c.requestCount, 1)
	c.Waits.Store(streamID, streamID)
}
func (c *BoltV1Client) Stats() {
	c.t.Logf("client %s send request:%d, get response:%d \n", c.ClientID, c.requestCount, c.respCount)
}

//
func (c *BoltV1Client) OnReceiveData(data types.IoBuffer, endStream bool) {
}
func (c *BoltV1Client) OnReceiveTrailers(trailers map[string]string) {
}
func (c *BoltV1Client) OnDecodeError(err error, headers map[string]string) {
}
func (c *BoltV1Client) OnReceiveHeaders(headers map[string]string, endStream bool) {
	streamID, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)]
	if ok {
		if _, ok := c.Waits.Load(streamID); ok {
			//c.t.Logf("Get Stream Response: %s ,headers: %v\n", streamID, headers)
			atomic.AddUint32(&c.respCount, 1)
			c.Waits.Delete(streamID)
		}
	}
}

//Protocols
func buildBoltV1Request(requestID uint32) *sofarpc.BoltRequestCommand {
	request := &sofarpc.BoltRequestCommand{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.RPC_REQUEST,
		Version:  1,
		ReqID:    requestID,
		CodecPro: sofarpc.HESSIAN_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}

	headers := map[string]string{"service": "testSofa"} // used for sofa routing

	if headerBytes, err := serialize.Instance.Serialize(headers); err != nil {
		panic("serialize headers error")
	} else {
		request.HeaderMap = headerBytes
		request.HeaderLen = int16(len(headerBytes))
	}

	return request
}
func buildBoltV1Resposne(req *sofarpc.BoltRequestCommand) *sofarpc.BoltResponseCommand {
	return &sofarpc.BoltResponseCommand{
		Protocol:       req.Protocol,
		CmdType:        sofarpc.RESPONSE,
		CmdCode:        sofarpc.RPC_RESPONSE,
		Version:        req.Version,
		ReqID:          req.ReqID,
		CodecPro:       req.CodecPro, //todo: read default codec from config
		ResponseStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
		HeaderLen:      req.HeaderLen,
		HeaderMap:      req.HeaderMap,
	}

}

//SofaRpc Serve
func ServeBoltV1(t *testing.T, conn net.Conn) {
	iobuf := buffer.NewIoBuffer(102400)
	for {
		now := time.Now()
		conn.SetReadDeadline(now.Add(30 * time.Second))
		buf := make([]byte, 10*1024)
		bytesRead, err := conn.Read(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				t.Logf("Connect read error: %v\n", err)
				continue
			}
			return
		}
		if bytesRead > 0 {
			iobuf.Write(buf[:bytesRead])
			for iobuf.Len() > 1 {
				cmd, _ := codec.BoltV1.GetDecoder().Decode(nil, iobuf)
				if cmd == nil {
					break
				}
				if req, ok := cmd.(*sofarpc.BoltRequestCommand); ok {
					resp := buildBoltV1Resposne(req)
					iobufresp, err := codec.BoltV1.GetEncoder().EncodeHeaders(nil, resp)
					if err != nil {
						t.Errorf("Build response error: %v\n", err)
					} else {
						//t.Logf("server %s write to remote: %d\n", conn.LocalAddr().String(), resp.GetReqID)
						respdata := iobufresp.Bytes()
						conn.Write(respdata)
					}
				}
			}
		}
	}
}

//tools
func RandomDuration(min, max time.Duration) time.Duration {
	if min > max {
		return min
	}
	d := rand.Int63n(int64(max - min))
	return time.Duration(d) * time.Nanosecond
}
func IsMapEmpty(m sync.Map) bool {
	empty := true
	//If there is a key in the map, return not empty
	m.Range(func(key, value interface{}) bool {
		empty = false
		return false
	})
	return empty
}
func WaitMapEmpty(m sync.Map, timeout time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		for {
			select {
			case <-ch:
				return
			default:
				if IsMapEmpty(m) {
					close(ch)
					return
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
	}() //check goroutine
	select {
	case <-time.After(timeout):
		close(ch)            // finish check goroutine
		return IsMapEmpty(m) // timeout, retry again
	case <-ch:
		return true //map empty
	}
}
