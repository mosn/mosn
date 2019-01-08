package main

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/http2"
	"github.com/alipay/sofa-mosn/pkg/stream"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func main() {
	clt, err := NewClient("127.0.0.1:2046")
	if err != nil {
		fmt.Println(err)
		return
	}
	resp := clt.SendRequest()
	fmt.Println("HEADER:", resp.header.(*http2.RspHeader).HeaderMap)
	fmt.Println("----")
	fmt.Println("BODY:", resp.data)
	fmt.Println("----")
	fmt.Println("Trailer:", resp.trailer)
}

type Client struct {
	client stream.Client
}

func NewClient(addr string) (*Client, error) {
	stopChan := make(chan struct{})
	remoteAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	conn := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
	if err := conn.Connect(true); err != nil {
		return nil, err
	}
	client := stream.NewStreamClient(context.Background(), protocol.HTTP2, conn, nil)
	if client == nil {
		return nil, errors.New("create failed")
	}
	return &Client{
		client: client,
	}, nil

}

type Response struct {
	header  types.HeaderMap
	data    types.IoBuffer
	trailer types.HeaderMap
}

/*
Accept-Encoding gzip
User-Agent Go-http-client/2.0
Content-Type application/x-www-form-urlencoded
Content-Length 4
*/

func (c *Client) SendRequest() *Response {
	ch := make(chan *Response, 1)
	receiver := &receiver{
		resp: &Response{},
		ch:   ch,
	}
	ctx := context.Background()
	headers := protocol.CommonHeader(map[string]string{
		"Accept-Encoding": "gzip",
		"User-Agent":      "Go-http-client/2.0",
		"Content-Type":    "application/x-www-form-urlencoded",
		"Content-Length":  "4",
	})
	data := buffer.NewIoBufferString("test")
	trailers := protocol.CommonHeader(map[string]string{
		"TestTrailer": "Trailer",
	})
	encoder := c.client.NewStream(ctx, receiver)
	encoder.AppendHeaders(ctx, headers, false)
	encoder.AppendData(ctx, data, false)
	encoder.AppendTrailers(ctx, trailers)
	return <-ch
}

type receiver struct {
	resp *Response
	ch   chan<- *Response
}

func (r *receiver) OnReceiveHeaders(context context.Context, headers types.HeaderMap, endStream bool) {
	r.resp.header = headers
	if endStream {
		r.ch <- r.resp
	}
}

func (r *receiver) OnReceiveData(context context.Context, data types.IoBuffer, endStream bool) {
	r.resp.data = data
	if endStream {
		r.ch <- r.resp
	}
}

func (r *receiver) OnReceiveTrailers(context context.Context, trailers types.HeaderMap) {
	r.resp.trailer = trailers
	r.ch <- r.resp
}

func (r *receiver) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
	r.resp.header = protocol.CommonHeader(map[string]string{
		"status": "failed",
		"error":  err.Error(),
	})
	r.ch <- r.resp
}
