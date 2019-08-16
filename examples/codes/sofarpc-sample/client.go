package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"time"

	"sofastack.io/sofa-mosn/pkg/buffer"
	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/network"
	"sofastack.io/sofa-mosn/pkg/protocol"
	"sofastack.io/sofa-mosn/pkg/protocol/rpc"
	"sofastack.io/sofa-mosn/pkg/protocol/rpc/sofarpc"
	_ "sofastack.io/sofa-mosn/pkg/protocol/rpc/sofarpc/codec"
	"sofastack.io/sofa-mosn/pkg/protocol/serialize"
	"sofastack.io/sofa-mosn/pkg/stream"
	_ "sofastack.io/sofa-mosn/pkg/stream/sofarpc"
	"sofastack.io/sofa-mosn/pkg/types"
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
	conn := network.NewClientConnection(nil, nil, remoteAddr, stopChan)
	if err := conn.Connect(); err != nil {
		fmt.Println(err)
		return nil
	}
	c.Client = stream.NewStreamClient(context.Background(), protocol.SofaRPC, conn, nil)
	c.conn = conn
	return c
}

func (c *Client) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	fmt.Printf("[RPC Client] Receive Data:")
	if cmd, ok := headers.(sofarpc.SofaRpcCmd); ok {
		streamID := protocol.StreamIDConv(cmd.RequestID())

		if resp, ok := cmd.(rpc.RespStatus); ok {
			fmt.Println("stream:", streamID, " status:", resp.RespStatus())
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

func buildBoltV1Request(requestID uint64) *sofarpc.BoltRequest {
	request := &sofarpc.BoltRequest{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.RPC_REQUEST,
		Version:  1,
		ReqID:    uint32(requestID),
		Codec:    sofarpc.HESSIAN2_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}

	headers := map[string]string{"service": "testSofa"} // used for sofa routing

	buf := buffer.NewIoBuffer(100)
	if err := serialize.Instance.SerializeMap(headers, buf); err != nil {
		panic("serialize headers error")
	} else {
		request.HeaderMap = buf.Bytes()
		request.HeaderLen = int16(buf.Len())
	}

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
