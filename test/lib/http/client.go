package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	mosnhttp2 "mosn.io/mosn/pkg/protocol/http2"
	"mosn.io/mosn/pkg/stream"
	_ "mosn.io/mosn/pkg/stream/http"  // register http1
	_ "mosn.io/mosn/pkg/stream/http2" // register http2
	mtypes "mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/types"
	"mosn.io/mosn/test/lib/utils"
	"mosn.io/pkg/buffer"
)

func init() {
	lib.RegisterCreateClient("Http1", NewHttpClient)
}

// MockHttpClient use mosn http protocol and stream
// control metrics and connection
type MockHttpClient struct {
	// config
	config *HttpClientConfig
	// stats
	stats *types.ClientStats
	// connection pool
	protocolName api.ProtocolName
	curConnNum   uint32
	maxConnNum   uint32
	connPool     chan *HttpConn
}

func NewHttpClient(config interface{}) types.MockClient {
	cfg, err := NewHttpClientConfig(config)
	if err != nil {
		log.DefaultLogger.Errorf("new http client config error: %v", err)
		return nil
	}
	if cfg.MaxConn == 0 {
		cfg.MaxConn = 1
	}
	if cfg.ProtocolName == "" {
		cfg.ProtocolName = "Http1"
	}
	return &MockHttpClient{
		config:       cfg,
		stats:        types.NewClientStats(),
		protocolName: api.ProtocolName(cfg.ProtocolName),
		maxConnNum:   cfg.MaxConn,
		connPool:     make(chan *HttpConn, cfg.MaxConn),
	}
}

func (c *MockHttpClient) SyncCall() bool {
	conn, err := c.getOrCreateConnection()
	if err != nil {
		log.DefaultLogger.Errorf("get connection from pool error: %v", err)
		return false
	}
	defer func() {
		c.releaseConnection(conn)
	}()
	c.stats.Records().RecordRequest()
	resp, err := conn.SyncSendRequest(c.config.Request, c.protocolName)
	status := false
	switch err {
	case ErrClosedConnection:
	case ErrRequestTimeout:
		// TODO: support timeout verify
	case nil:
		status = c.config.Verify.Verify(resp)
		c.stats.Records().RecordResponse(int16(resp.StatusCode))
	default:
		log.DefaultLogger.Errorf("unexpected error got: %v", err)
	}
	c.stats.Response(status)
	return status
}

// TODO: implement it
func (c *MockHttpClient) AsyncCall() {
}

func (c *MockHttpClient) Stats() types.ClientStatsReadOnly {
	return c.stats
}

// Close will close all the connections
func (c *MockHttpClient) Close() {
	for {
		select {
		case conn := <-c.connPool:
			conn.Close()
			c.releaseConnection(conn)
		default:
			return // no more connections
		}
	}

}

// connpool implementation
func (c *MockHttpClient) getOrCreateConnection() (*HttpConn, error) {
	select {
	case conn := <-c.connPool:
		if !conn.IsClosed() {
			return conn, nil
		}
		// got a closed connection, try to make a new one
		atomic.AddUint32(&c.curConnNum, ^uint32(0))
	default:
		// try to make a new connection
	}
	// connection is full, wait connection
	// TODO: add timeout
	if atomic.LoadUint32(&c.curConnNum) >= c.maxConnNum {
		return <-c.connPool, nil
	}
	conn, err := NewConn(c.config.TargetAddr, c.protocolName, func() {
		c.stats.CloseConnection()
	})
	if err != nil {
		return nil, err
	}
	atomic.AddUint32(&c.curConnNum, 1)
	c.stats.ActiveConnection()
	return conn, nil

}

func (c *MockHttpClient) releaseConnection(conn *HttpConn) {
	if conn.IsClosed() {
		atomic.AddUint32(&c.curConnNum, ^uint32(0))
		return
	}
	select {
	case c.connPool <- conn:
	default:
	}
}

type Response struct {
	StatusCode int
	Header     map[string][]string
	Body       []byte
	Cost       time.Duration
}

type HttpConn struct {
	conn          mtypes.ClientConnection
	stream        stream.Client
	stop          chan struct{}
	closeCallback func()
}

func NewConn(addr string, pname api.ProtocolName, cb func()) (*HttpConn, error) {

	var remoteAddr net.Addr
	var err error
	if remoteAddr, err = net.ResolveTCPAddr("tcp", addr); err != nil {
		remoteAddr, err = net.ResolveUnixAddr("unix", addr)
	}

	if err != nil {
		return nil, err
	}
	hconn := &HttpConn{
		stop:          make(chan struct{}),
		closeCallback: cb,
	}
	conn := network.NewClientConnection(time.Second, nil, remoteAddr, make(chan struct{}))
	conn.AddConnectionEventListener(hconn)
	hconn.conn = conn
	s := stream.NewStreamClient(context.Background(), pname, conn, nil)
	if s == nil {
		return nil, fmt.Errorf("protocol %s not registered", pname)
	}
	if err := conn.Connect(); err != nil {
		return nil, err
	}
	hconn.stream = s
	return hconn, nil
}

func (c *HttpConn) OnEvent(event api.ConnectionEvent) {
	if event.IsClose() {
		close(c.stop)
		if c.closeCallback != nil {
			c.closeCallback()
		}
	}
}

func (c *HttpConn) Close() {
	c.conn.Close(api.NoFlush, api.LocalClose)
}

func (c *HttpConn) IsClosed() bool {
	select {
	case <-c.stop:
		return true
	default:
		return false
	}
}

func (c *HttpConn) AsyncSendRequest(receiver mtypes.StreamReceiveListener, req *RequestConfig) {
	ctx := variable.NewVariableContext(context.Background())
	headers, body := req.BuildRequest(ctx)
	encoder := c.stream.NewStream(ctx, receiver)
	encoder.AppendHeaders(ctx, headers, body == nil)

	if body != nil {
		encoder.AppendData(ctx, body, true)
	}
}

var (
	ErrClosedConnection = errors.New("send request on closed connection")
	ErrRequestTimeout   = errors.New("sync call timeout")
)

func (c *HttpConn) SyncSendRequest(req *RequestConfig, proto api.ProtocolName) (*Response, error) {
	select {
	case <-c.stop:
		return nil, ErrClosedConnection
	default:
		ch := make(chan *Response)
		r := newReceiver(ch, proto)
		c.AsyncSendRequest(r, req)
		// set default timeout, if a timeout is configured, use it
		timeout := 5 * time.Second
		if req != nil && req.Timeout > 0 {
			timeout = req.Timeout
		}
		timer := time.NewTimer(timeout)
		select {
		case resp := <-ch:
			timer.Stop()
			return resp, nil
		case <-timer.C:
			return nil, ErrRequestTimeout
		}
	}
}

type receiver struct {
	proto api.ProtocolName
	data  *Response
	start time.Time
	ch    chan<- *Response // write only
}

func newReceiver(ch chan<- *Response, proto api.ProtocolName) *receiver {
	return &receiver{
		proto: proto,
		data:  &Response{},
		start: time.Now(),
		ch:    ch,
	}
}

func (r *receiver) OnReceive(ctx context.Context, headers api.HeaderMap, data buffer.IoBuffer, _ api.HeaderMap) {
	r.data.Cost = time.Now().Sub(r.start)
	switch r.proto {
	case protocol.HTTP1:
		cmd := headers.(mosnhttp.ResponseHeader).ResponseHeader
		r.data.Header = utils.ReadFasthttpResponseHeaders(cmd)
		r.data.StatusCode = cmd.StatusCode()
	case protocol.HTTP2:
		cmd := headers.(*mosnhttp2.RspHeader)
		r.data.Header = mosnhttp2.EncodeHeader(cmd.HeaderMap)
		r.data.StatusCode = cmd.Rsp.StatusCode
	}
	if data != nil {
		r.data.Body = data.Bytes()
	}
	r.ch <- r.data
}

func (r *receiver) OnDecodeError(context context.Context, err error, _ api.HeaderMap) {
	r.data.Cost = time.Now().Sub(r.start)
	r.data.Cost = time.Now().Sub(r.start)
	r.data.StatusCode = http.StatusInternalServerError
	r.data.Header = map[string][]string{
		"X-Mosn-Error": []string{
			err.Error(),
		},
	}
	r.ch <- r.data
}
