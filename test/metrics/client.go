package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
	"golang.org/x/net/http2"
)

type Client interface {
	Send() <-chan error
	DestroyConn()
}

// HTTP Client, used in http and http2
type HTTPClient struct {
	Addr   string
	client http.Client
}

func NewHTTP1Client(addr string) Client {
	return &HTTPClient{
		Addr:   addr,
		client: http.Client{},
	}
}
func NewHTTP2Client(addr string) Client {
	tr := &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(netw, addr)
		},
	}
	return &HTTPClient{
		Addr:   addr,
		client: http.Client{Transport: tr},
	}
}

func (c *HTTPClient) Send() <-chan error {
	ch := make(chan error)
	go func(ch chan<- error) {
		resp, err := c.client.Get("http://" + c.Addr)
		if err != nil {
			ch <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			ch <- errors.New(resp.Status)
			return
		}
		ioutil.ReadAll(resp.Body)
		ch <- nil
	}(ch)
	return ch
}
func (c *HTTPClient) DestroyConn() {}

// RPC Client
type streamReceiver struct {
	ch chan<- error
}

func (s *streamReceiver) OnReceiveData(context context.Context, data types.IoBuffer, endStream bool) {
}
func (s *streamReceiver) OnReceiveTrailers(context context.Context, trailers types.HeaderMap) {
}
func (s *streamReceiver) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
}
func (s *streamReceiver) OnReceiveHeaders(context context.Context, headers types.HeaderMap, endStream bool) {
	if cmd, ok := headers.(sofarpc.ProtoBasicCmd); ok {
		status := cmd.GetRespStatus()
		if int16(status) != sofarpc.RESPONSE_STATUS_SUCCESS {
			s.ch <- errors.New(fmt.Sprintf("%d", status))
			return
		}
		s.ch <- nil
		return
	}

	s.ch <- errors.New("no response status")
}

type RPCClient struct {
	Addr     string
	conn     types.ClientConnection
	codec    stream.CodecClient
	streamID uint32
}

func NewRPCClient(addr string) Client {
	return &RPCClient{
		Addr: addr,
	}
}

func (c *RPCClient) connect() error {
	stopChan := make(chan struct{})
	remoteAddr, _ := net.ResolveTCPAddr("tcp", c.Addr)
	cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
	if err := cc.Connect(true); err != nil {
		return err
	}
	c.conn = cc
	return nil
}

func (c *RPCClient) Send() <-chan error {
	ch := make(chan error)
	go func(ch chan<- error) {
		if c.conn == nil {
			if err := c.connect(); err != nil {
				ch <- err
				return
			}
			c.codec = stream.NewCodecClient(context.Background(), protocol.SofaRPC, c.conn, nil)
		}
		id := atomic.AddUint32(&c.streamID, 1)
		streamID := protocol.StreamIDConv(id)
		encoder := c.codec.NewStream(context.Background(), streamID, &streamReceiver{ch})
		headers := util.BuildBoltV1Request(id)
		encoder.AppendHeaders(context.Background(), headers, true)
	}(ch)
	return ch
}
func (c *RPCClient) DestroyConn() {
	if c.conn != nil {
		c.conn.Close(types.NoFlush, types.LocalClose)
		c.conn = nil
	}
}
