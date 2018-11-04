package util

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/stream/xprotocol/subprotocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// XProtocol needs subprotocol for rpc
// is different from other protocol (extension)
type XProtocolClient struct {
	t           *testing.T
	ClientID    string
	SubProtocol types.SubProtocol
	Codec       types.Multiplexing
	conn        types.ClientConnection
	streamID    uint64
}

// support SubProtocol
const (
	XExample = "rpc-example"
)

func NewXClient(t *testing.T, id string, subproto string) *XProtocolClient {
	return &XProtocolClient{
		t:           t,
		ClientID:    id,
		SubProtocol: types.SubProtocol(subproto),
	}
}

func (c *XProtocolClient) Connect(addr string) error {
	stopChan := make(chan struct{})
	remoteAddr, _ := net.ResolveTCPAddr("tcp", addr)
	cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
	cc.SetReadDisable(true)
	c.conn = cc
	if err := cc.Connect(true); err != nil {
		c.t.Logf("client[%s] connect to server error: %v\n", c.ClientID, err)
		return err
	}
	c.Codec = subprotocol.CreateSubProtocolCodec(context.Background(), c.SubProtocol)
	return nil
}

func (c *XProtocolClient) RequestAndWaitReponse() error {
	var req []byte
	reqID := fmt.Sprintf("%d", atomic.AddUint64(&c.streamID, 1))
	switch c.SubProtocol {
	case XExample:
		// build request
		req = make([]byte, 16)
		data := []byte{14, 1, 0, 8, 0, 0, 3, 0}
		copy(req, data)
	default:
		return fmt.Errorf("unsupport sub protocol")
	}
	req = c.Codec.SetStreamID(req, reqID)
	c.conn.Write(buffer.NewIoBufferBytes(req))
	// wait response
	for {
		now := time.Now()
		conn := c.conn.RawConn()
		conn.SetReadDeadline(now.Add(30 * time.Second))
		resp := make([]byte, 10*1024)
		bytesRead, err := conn.Read(resp)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				continue
			}
			return err
		}
		if bytesRead > 0 {
			respID := c.Codec.GetStreamID(resp[:bytesRead])
			if reqID != respID {
				return fmt.Errorf("reponse streamID: %s not match request: %s", respID, reqID)
			}
			return nil
		}
	}

}
func (c *XProtocolClient) Close() {
	if c.conn != nil {
		c.conn.Close(types.NoFlush, types.LocalClose)
	}
}

type XProtocolServer struct {
	UpstreamServer
	Client *XProtocolClient
}

func NewXProtocolServer(t *testing.T, addr string, subproto string) UpstreamServer {
	s := &XProtocolServer{
		Client: NewXClient(t, "xClient", subproto),
	}
	switch subproto {
	case XExample:
		s.UpstreamServer = NewUpstreamServer(t, addr, s.ServeXExample)
	default:
		t.Errorf("unsupport sub protocol")
		return nil
	}
	return s
}

func (s *XProtocolServer) ServeXExample(t *testing.T, conn net.Conn) {
	response := func(iobuf types.IoBuffer) ([]byte, bool) {
		codec := subprotocol.NewRPCExample()
		streamID := codec.GetStreamID(iobuf.Bytes())
		resp := make([]byte, 16)
		data := []byte{14, 1, 1, 20, 8, 0, 0, 0}
		copy(resp[:8], data)
		resp = codec.SetStreamID(resp, streamID)
		iobuf.Drain(iobuf.Len())
		return resp, true
	}
	// can reuse
	ServeSofaRPC(t, conn, response)
}
