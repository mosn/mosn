package util

import (
	"context"
	"fmt"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"mosn.io/mosn/pkg/api/v2"
	"mosn.io/mosn/pkg/buffer"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
)

type RPCClient struct {
	t              *testing.T
	ClientID       string
	Protocol       types.ProtocolName //bolt1, bolt2
	Codec          stream.Client
	Waits          sync.Map
	conn           types.ClientConnection
	streamID       uint64
	respCount      uint32
	requestCount   uint32
	ExpectedStatus int16
}

func NewRPCClient(t *testing.T, id string, proto types.ProtocolName) *RPCClient {
	return &RPCClient{
		t:              t,
		ClientID:       id,
		Protocol:       proto,
		Waits:          sync.Map{},
		ExpectedStatus: int16(bolt.ResponseStatusSuccess), // default expected success
	}
}

func (c *RPCClient) connect(addr string, tlsMng types.TLSContextManager) error {
	stopChan := make(chan struct{})
	remoteAddr, _ := net.ResolveTCPAddr("tcp", addr)
	cc := network.NewClientConnection(nil, 0, tlsMng, remoteAddr, stopChan)
	c.conn = cc
	if err := cc.Connect(); err != nil {
		c.t.Logf("client[%s] connect to server error: %v\n", c.ClientID, err)
		return err
	}
	ctx := context.WithValue(context.Background(), types.ContextSubProtocol, string(c.Protocol))
	c.Codec = stream.NewStreamClient(ctx, protocol.Xprotocol, cc, nil)
	if c.Codec == nil {
		return fmt.Errorf("NewStreamClient error %v, %v", protocol.Xprotocol, cc)
	}
	return nil
}

func (c *RPCClient) ConnectTLS(addr string, cfg *v2.TLSConfig) error {
	tlsMng, err := mtls.NewTLSClientContextManager(cfg)
	if err != nil {
		return err
	}
	return c.connect(addr, tlsMng)

}

func (c *RPCClient) Connect(addr string) error {
	return c.connect(addr, nil)
}

func (c *RPCClient) Stats() bool {
	c.t.Logf("client %s send request:%d, get response:%d \n", c.ClientID, c.requestCount, c.respCount)
	return c.requestCount == c.respCount
}

func (c *RPCClient) Close() {
	if c.conn != nil {
		c.conn.Close(types.NoFlush, types.LocalClose)
		c.streamID = 0 // reset connection stream id
	}
}

func (c *RPCClient) SendRequest() {
	c.SendRequestWithData("testdata")
}
func (c *RPCClient) SendRequestWithData(in string) {
	ID := atomic.AddUint64(&c.streamID, 1)
	streamID := protocol.StreamIDConv(ID)
	requestEncoder := c.Codec.NewStream(context.Background(), c)
	var frame xprotocol.XFrame
	data := buffer.NewIoBufferString(in)
	// TODO: support boltv2, dubbo, tars
	switch c.Protocol {
	case bolt.ProtocolName:
		// header used for sofa routing
		frame = bolt.NewRpcRequest(uint32(ID), protocol.CommonHeader(map[string]string{"service": "testSofa"}), data)
	default:
		c.t.Errorf("unsupport protocol")
		return
	}
	requestEncoder.AppendHeaders(context.Background(), frame.GetHeader(), false)
	requestEncoder.AppendData(context.Background(), data, true)
	atomic.AddUint32(&c.requestCount, 1)
	c.Waits.Store(streamID, streamID)
}

func (c *RPCClient) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	if cmd, ok := headers.(xprotocol.XRespFrame); ok {
		streamID := protocol.StreamIDConv(cmd.GetRequestId())

		if _, ok := c.Waits.Load(streamID); ok {
			c.t.Logf("RPC client receive streamId:%s \n", streamID)
			atomic.AddUint32(&c.respCount, 1)
			status := int16(cmd.GetStatusCode())
			if status == c.ExpectedStatus {
				c.Waits.Delete(streamID)
			}
		} else {
			c.t.Errorf("get a unexpected stream ID %s", streamID)
		}
	} else {
		c.t.Errorf("get a unexpected header type")
	}
}

func (c *RPCClient) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
}

type RPCServer struct {
	UpstreamServer
	Client *RPCClient
	// Statistic
	Name  string
	Count uint32
}

func NewRPCServer(t *testing.T, addr string, proto types.ProtocolName) UpstreamServer {
	s := &RPCServer{
		Client: NewRPCClient(t, "rpcClient", proto),
		Name:   addr,
	}
	// TODO: support boltv2, dubbo, tars
	switch proto {
	case bolt.ProtocolName:
		s.UpstreamServer = NewUpstreamServer(t, addr, s.ServeBoltV1)
	default:
		t.Errorf("unsupport protocol")
		return nil
	}
	return s
}

func (s *RPCServer) ServeBoltV1(t *testing.T, conn net.Conn) {
	response := func(iobuf types.IoBuffer) ([]byte, bool) {
		protocol := xprotocol.GetProtocol(bolt.ProtocolName)
		cmd, _ := protocol.Decode(context.Background(), iobuf)
		if cmd == nil {
			return nil, false
		}
		if req, ok := cmd.(*bolt.Request); ok {
			t.Logf("RPC Server receive streamId: %d \n", req.RequestId)
			atomic.AddUint32(&s.Count, 1)
			resp := bolt.NewRpcResponse(req.RequestId, bolt.ResponseStatusSuccess, nil, nil)
			iobufresp, err := protocol.Encode(context.Background(), resp)
			if err != nil {
				t.Errorf("Build response error: %v\n", err)
				return nil, true
			}
			return iobufresp.Bytes(), true
		} else {
			t.Logf("Unrecognized request:%+v \n", cmd)
		}
		return nil, true
	}
	ServeSofaRPC(t, conn, response)

}

func ServeSofaRPC(t *testing.T, conn net.Conn, responseHandler func(iobuf types.IoBuffer) ([]byte, bool)) {
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
				// ok means receive a full data
				data, ok := responseHandler(iobuf)
				if !ok {
					break
				}
				if data != nil {
					conn.Write(data)
				}
			}
		}
	}
}
