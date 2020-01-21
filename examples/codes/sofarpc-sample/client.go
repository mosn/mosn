package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"time"

	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
)

type Client struct {
	Client stream.Client
	conn   types.ClientConnection
	Id     uint64
}

func NewClient(addr string) *Client {
	c := &Client{}
	stopChan := make(chan struct{})
	remoteAddr, _ := net.ResolveTCPAddr("tcp", addr)
	conn := network.NewClientConnection(nil, 0, nil, remoteAddr, stopChan)
	if err := conn.Connect(); err != nil {
		fmt.Println(err)
		return nil
	}
	ctx := context.WithValue(context.Background(), types.ContextSubProtocol, string(bolt.ProtocolName))
	c.Client = stream.NewStreamClient(ctx, protocol.Xprotocol, conn, nil)
	c.conn = conn
	return c
}

func (c *Client) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	fmt.Printf("[RPC Client] Receive Data:")
	if cmd, ok := headers.(xprotocol.XFrame); ok {
		streamID := protocol.StreamIDConv(cmd.GetRequestId())

		if resp, ok := cmd.(xprotocol.XRespFrame); ok {
			fmt.Println("stream:", streamID, " status:", resp.GetStatusCode())
		}
	}
}

func (c *Client) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {}

func (c *Client) Request() {
	c.Id++
	requestEncoder := c.Client.NewStream(context.Background(), c)
	headers := buildBoltV1Request(c.Id)
	requestEncoder.AppendHeaders(context.Background(), headers, true)
}

func buildBoltV1Request(requestID uint64) *bolt.Request {
	request := &bolt.Request{
		RequestHeader: bolt.RequestHeader{
			Protocol:  bolt.ProtocolCode,
			CmdType:   bolt.CmdTypeRequest,
			CmdCode:   bolt.CmdCodeRpcRequest,
			Version:   bolt.ProtocolVersion,
			RequestId: uint32(requestID),
			Codec:     bolt.Hessian2Serialize,
			Timeout:   -1,
		},
	}

	request.Set("service", "testSofa")
	return request
}

func main() {
	log.InitDefaultLogger("", log.DEBUG)
	t := flag.Bool("t", false, "-t")
	flag.Parse()
	if client := NewClient("127.0.0.1:2045"); client != nil {
		for {
			client.Request()
			time.Sleep(200 * time.Millisecond)
			if !*t {
				time.Sleep(3 * time.Second)
				return
			}
		}
	}
}
