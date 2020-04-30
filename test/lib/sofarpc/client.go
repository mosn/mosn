package sofarpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/stream"
	_ "mosn.io/mosn/pkg/stream/xprotocol"
	"mosn.io/mosn/pkg/types"
)

type receiver struct {
	Data  *Response
	start time.Time
	ch    chan<- *Response
}

func (r *receiver) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	cmd := headers.(xprotocol.XRespFrame)
	r.Data.Header = make(map[string]string)
	cmd.GetHeader().Range(func(key, value string) bool {
		r.Data.Header[key] = value
		return true
	})
	r.Data.Status = cmd.GetStatusCode()
	if data != nil {
		r.Data.Data = data.Bytes()
	}
	r.Data.Cost = time.Now().Sub(r.start)
	r.ch <- r.Data
}

func (r *receiver) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
	r.Data.Status = 1 // RESPONSE_STATUS_ERROR
	r.Data.Header = map[string]string{
		"error_message": err.Error(),
	}
	r.Data.Cost = time.Now().Sub(r.start)
	r.ch <- r.Data
}

type ConnClient struct {
	MakeRequest MakeRequestFunc
	SyncTimeout time.Duration
	//
	isClosed bool
	close    chan struct{}
	client   stream.Client
	conn     types.ClientConnection
	id       uint64
}

func NewConnClient(addr string, f MakeRequestFunc) (*ConnClient, error) {
	remoteAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	c := &ConnClient{
		MakeRequest: f,
		close:       make(chan struct{}),
	}
	conn := network.NewClientConnection(nil, 0, nil, remoteAddr, make(chan struct{}))
	if err := conn.Connect(); err != nil {
		return nil, err
	}
	conn.AddConnectionEventListener(c)
	c.conn = conn
	ctx := context.WithValue(context.Background(), types.ContextSubProtocol, string(bolt.ProtocolName))
	client := stream.NewStreamClient(ctx, protocol.Xprotocol, conn, nil)
	if client == nil {
		return nil, errors.New("protocol not registered")
	}
	c.client = client
	return c, nil
}

func (c *ConnClient) OnEvent(event api.ConnectionEvent) {
	if event.IsClose() {
		c.isClosed = true
		close(c.close)
	}
}

func (c *ConnClient) Close() {
	c.conn.Close(api.NoFlush, api.LocalClose)
}

func (c *ConnClient) IsClosed() bool {
	return c.isClosed
}

func (c *ConnClient) sendRequest(receiver types.StreamReceiveListener, header map[string]string, body []byte) {
	ctx := context.Background()
	streamEncoder := c.client.NewStream(ctx, receiver)
	atomic.AddUint64(&c.id, 1)
	cmd, data := c.MakeRequest(c.id, header, body)
	streamEncoder.AppendHeaders(ctx, cmd, false)
	streamEncoder.AppendData(ctx, data, true)

}

func (c *ConnClient) SyncSend(header map[string]string, body []byte) (*Response, error) {
	select {
	case <-c.close:
		return nil, errors.New("closed connection client")
	default:
		ch := make(chan *Response)
		r := &receiver{
			Data:  &Response{},
			start: time.Now(),
			ch:    ch,
		}
		c.sendRequest(r, header, body)
		// set default timeout, if a timeout is configured, use it
		timeout := 5 * time.Second
		if c.SyncTimeout > 0 {
			timeout = c.SyncTimeout
		}
		// use timeout to make sure sync send will receive a result
		select {
		case resp := <-ch:
			return resp, nil
		case <-time.After(timeout):
			return nil, errors.New("sync call timeout")
		}
	}
}

// TODO: Async client

type ClientConfig struct {
	Addr          string
	MakeRequest   MakeRequestFunc
	RequestHeader map[string]string
	RequestBody   []byte
	// request timeout is used for sync call
	// if zero, we set default request time, 5 second
	RequestTImeout time.Duration
	// if Verify is nil, just expected returns success
	Verify ResponseVerify
}

func CreateSimpleConfig(addr string) *ClientConfig {
	return &ClientConfig{
		Addr:        addr,
		MakeRequest: BuildBoltV1Request,
		RequestHeader: map[string]string{
			"service": "mosn-test-default-service",
		},
		RequestBody: []byte("mosn-test-default"),
	}
}

type Client struct {
	Cfg *ClientConfig
	// Stats
	Stats *ClientStats
	// conn pool
	connNum  uint32
	maxNum   uint32
	connPool chan *ConnClient
}

func NewClient(cfg *ClientConfig, maxConnections uint32) *Client {
	return &Client{
		Cfg:      cfg,
		Stats:    NewClientStats(),
		maxNum:   maxConnections,
		connPool: make(chan *ConnClient, maxConnections),
	}
}

func (c *Client) getOrCreateConnection() (*ConnClient, error) {
	select {
	case conn := <-c.connPool:
		if !conn.IsClosed() {
			return conn, nil // return a not closed conn
		}
		atomic.AddUint32(&c.connNum, ^uint32(0))
		c.Stats.CloseConnection()
	default:
	}
	if atomic.LoadUint32(&c.connNum) >= c.maxNum {
		return <-c.connPool, nil
	}
	conn, err := NewConnClient(c.Cfg.Addr, c.Cfg.MakeRequest)
	if err != nil {
		return nil, err
	}
	if c.Cfg.RequestTImeout > 0 {
		conn.SyncTimeout = c.Cfg.RequestTImeout
	}
	atomic.AddUint32(&c.connNum, 1)
	c.Stats.ActiveConnection()
	return conn, nil
}

func (c *Client) release(conn *ConnClient) {
	if conn.IsClosed() {
		atomic.AddUint32(&c.connNum, ^uint32(0))
		c.Stats.CloseConnection()
		return
	}
	select {
	case c.connPool <- conn:
	default:
	}
}

func (c *Client) SyncCall() bool {
	conn, err := c.getOrCreateConnection()
	if err != nil {
		fmt.Println("get connection from pool error: ", err)
		return false
	}
	defer func() {
		c.release(conn)
	}()
	c.Stats.Request()
	resp, err := conn.SyncSend(c.Cfg.RequestHeader, c.Cfg.RequestBody)
	if err != nil {
		fmt.Println("sync call failed: ", err)
		return false
	}
	if c.Cfg.Verify == nil {
		c.Cfg.Verify = DefaultVeirfy.Verify
	}
	ok := c.Cfg.Verify(resp)
	c.Stats.Response(ok)
	return ok

}

// Close All the connections
func (c *Client) Close() {
	for {
		select {
		case conn := <-c.connPool:
			conn.Close()
			c.release(conn)
		default:
			return // no more conn
		}
	}
}
