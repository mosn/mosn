package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	_ "github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/codec"
	"github.com/alipay/sofa-mosn/pkg/stream"
	_ "github.com/alipay/sofa-mosn/pkg/stream/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type Client struct {
	Codec stream.CodecClient
	conn  types.ClientConnection
	Id    uint32
}

func NewClient(addr string) *Client {
	c := &Client{}
	stopChan := make(chan struct{})
	remoteAddr, _ := net.ResolveTCPAddr("tcp", addr)
	conn := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
	if err := conn.Connect(true); err != nil {
		fmt.Println(err)
		return nil
	}
	c.Codec = stream.NewCodecClient(context.Background(), protocol.SofaRPC, conn, nil)
	c.conn = conn
	return c
}

func (c *Client) OnReceiveData(context context.Context, data types.IoBuffer, endStream bool) {}
func (c *Client) OnReceiveTrailers(context context.Context, trailers types.HeaderMap)        {}
func (c *Client) OnDecodeError(context context.Context, err error, headers types.HeaderMap)  {}
func (c *Client) OnReceiveHeaders(context context.Context, headers types.HeaderMap, endStream bool) {
	fmt.Printf("[RPC Client] Receive Data:")
	if cmd, ok := headers.(sofarpc.ProtoBasicCmd); ok {
		streamID := protocol.StreamIDConv(cmd.GetReqID())
		fmt.Println("stream:", streamID, " status:", cmd.GetRespStatus())
	}
}

func (c *Client) Request() {
	c.Id++
	streamID := protocol.StreamIDConv(c.Id)
	requestEncoder := c.Codec.NewStream(context.Background(), streamID, c)
	headers := buildBoltV1Request(c.Id)
	requestEncoder.AppendHeaders(context.Background(), headers, true)
}

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
