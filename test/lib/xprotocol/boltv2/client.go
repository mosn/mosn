package boltv2

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/protocol/xprotocol/boltv2"
	"mosn.io/mosn/pkg/stream"
	_ "mosn.io/mosn/pkg/stream/xprotocol" // register xprotocol
	mtypes "mosn.io/mosn/pkg/types"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/types"
	"mosn.io/pkg/buffer"
)

func init() {
	lib.RegisterCreateClient("boltv2", NewBoltClient)
}

// MockBoltClient use mosn xprotocol.bolt protocol and stream
type MockBoltClient struct {
	// config
	config *BoltClientConfig
	// stats
	stats *types.ClientStats
	// connection pool
	curConnNum uint32
	maxConnNum uint32
	connPool   chan *BoltConn
}

func NewBoltClient(config interface{}) types.MockClient {
	cfg, err := NewBoltClientConfig(config)
	if err != nil {
		log.DefaultLogger.Errorf("new bolt client config error: %v", err)
		return nil
	}
	if cfg.MaxConn == 0 {
		cfg.MaxConn = 1
	}
	return &MockBoltClient{
		config:     cfg,
		stats:      types.NewClientStats(),
		maxConnNum: cfg.MaxConn,
		connPool:   make(chan *BoltConn, cfg.MaxConn),
	}
}

func (c *MockBoltClient) SyncCall() bool {
	conn, err := c.getOrCreateConnection()
	if err != nil {
		log.DefaultLogger.Errorf("get connection from pool error: %v", err)
		return false
	}
	defer func() {
		c.releaseConnection(conn)
	}()
	c.stats.Records().RecordRequest()
	resp, err := conn.SyncSendRequest(c.config.Request)
	status := false
	switch err {
	case ErrClosedConnection:
		c.stats.Records().RecordResponse(2)
	case ErrRequestTimeout:
		// TODO: support timeout verify
		c.stats.Records().RecordResponse(3)
	case nil:
		status = c.config.Verify.Verify(resp)
		c.stats.Records().RecordResponse(resp.GetResponseStatus())
	default:
		log.DefaultLogger.Errorf("unexpected error got: %v", err)
	}
	c.stats.Response(status)
	return status
}

// TODO: implement it
func (c *MockBoltClient) AsyncCall() {
}

func (c *MockBoltClient) Stats() types.ClientStatsReadOnly {
	return c.stats
}

// Close will close all the connections
func (c *MockBoltClient) Close() {
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
func (c *MockBoltClient) getOrCreateConnection() (*BoltConn, error) {
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
	conn, err := NewConn(c.config.TargetAddr, func() {
		c.stats.CloseConnection()
	})
	if err != nil {
		return nil, err
	}
	atomic.AddUint32(&c.curConnNum, 1)
	c.stats.ActiveConnection()
	return conn, nil

}

func (c *MockBoltClient) releaseConnection(conn *BoltConn) {
	if conn.IsClosed() {
		atomic.AddUint32(&c.curConnNum, ^uint32(0))
		return
	}
	select {
	case c.connPool <- conn:
	default:
	}
}

type BoltConn struct {
	conn          mtypes.ClientConnection
	stream        stream.Client
	stop          chan struct{}
	closeCallback func()
	reqId         uint32
}

func NewConn(addr string, cb func()) (*BoltConn, error) {
	remoteAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	hconn := &BoltConn{
		stop:          make(chan struct{}),
		closeCallback: cb,
	}
	conn := network.NewClientConnection(time.Second, nil, remoteAddr, make(chan struct{}))
	if err := conn.Connect(); err != nil {
		return nil, err
	}
	conn.AddConnectionEventListener(hconn)
	hconn.conn = conn
	ctx := context.Background()
	s := stream.NewStreamClient(ctx, boltv2.ProtocolName, conn, nil)
	if s == nil {
		return nil, errors.New("protocol not registered")
	}
	hconn.stream = s
	return hconn, nil
}

func (c *BoltConn) OnEvent(event api.ConnectionEvent) {
	if event.IsClose() {
		close(c.stop)
		if c.closeCallback != nil {
			c.closeCallback()
		}
	}
}

func (c *BoltConn) Close() {
	c.conn.Close(api.NoFlush, api.LocalClose)
}

func (c *BoltConn) IsClosed() bool {
	select {
	case <-c.stop:
		return true
	default:
		return false
	}
}

func (c *BoltConn) ReqID() uint32 {
	return atomic.AddUint32(&c.reqId, 1)
}

func (c *BoltConn) AsyncSendRequest(receiver *receiver, req *RequestConfig) {
	headers, body := req.BuildRequest(receiver.requestId)
	ctx := context.Background()
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

func (c *BoltConn) SyncSendRequest(req *RequestConfig) (*Response, error) {
	select {
	case <-c.stop:
		return nil, ErrClosedConnection
	default:
		ch := make(chan *Response)
		r := newReceiver(c.ReqID(), ch)
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
	requestId uint32
	data      *Response
	start     time.Time
	ch        chan<- *Response // write only
}

func newReceiver(id uint32, ch chan<- *Response) *receiver {
	return &receiver{
		requestId: id,
		data:      &Response{},
		start:     time.Now(),
		ch:        ch,
	}
}

func (r *receiver) OnReceive(ctx context.Context, headers api.HeaderMap, data buffer.IoBuffer, _ api.HeaderMap) {
	r.data.Cost = time.Now().Sub(r.start)
	cmd := headers.(api.XRespFrame)
	resp := cmd.(*boltv2.Response)
	r.data.Header = resp.ResponseHeader
	r.data.Header.RequestId = r.requestId
	r.data.Content = data
	r.ch <- r.data
}

func (r *receiver) OnDecodeError(context context.Context, err error, _ api.HeaderMap) {
	// build an error
	r.data.Cost = time.Now().Sub(r.start)
	r.data.Header = boltv2.ResponseHeader{
		Version1: boltv2.ProtocolVersion1,
		ResponseHeader: bolt.ResponseHeader{
			Protocol:       boltv2.ProtocolCode,
			CmdType:        bolt.CmdTypeResponse,
			CmdCode:        bolt.CmdCodeRpcResponse,
			Version:        boltv2.ProtocolVersion,
			Codec:          bolt.Hessian2Serialize,
			RequestId:      r.requestId,
			ResponseStatus: bolt.ResponseStatusError,
		},
	}
	r.data.Content = buffer.NewIoBufferString(err.Error())
	r.ch <- r.data
}

type Response struct {
	Header  boltv2.ResponseHeader
	Content buffer.IoBuffer
	Cost    time.Duration
}

func (r *Response) GetResponseStatus() int16 {
	return int16(r.Header.ResponseStatus)
}
