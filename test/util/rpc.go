package util

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/mtls"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc/codec"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
)

const (
	Bolt1 = "boltV1"
	Bolt2 = "boltV2"
)

type RPCClient struct {
	t              *testing.T
	ClientID       string
	Protocol       string //bolt1, bolt2
	Codec          stream.Client
	Waits          sync.Map
	conn           types.ClientConnection
	streamID       uint64
	respCount      uint32
	requestCount   uint32
	ExpectedStatus int16
}

func NewRPCClient(t *testing.T, id string, proto string) *RPCClient {
	return &RPCClient{
		t:              t,
		ClientID:       id,
		Protocol:       proto,
		Waits:          sync.Map{},
		ExpectedStatus: sofarpc.RESPONSE_STATUS_SUCCESS, // default expected success
	}
}

func (c *RPCClient) connect(addr string, tlsMng types.TLSContextManager) error {
	stopChan := make(chan struct{})
	remoteAddr, _ := net.ResolveTCPAddr("tcp", addr)
	cc := network.NewClientConnection(nil, tlsMng, remoteAddr, stopChan, log.DefaultLogger)
	c.conn = cc
	if err := cc.Connect(true); err != nil {
		c.t.Logf("client[%s] connect to server error: %v\n", c.ClientID, err)
		return err
	}
	c.Codec = stream.NewStreamClient(context.Background(), protocol.SofaRPC, cc, nil)
	if c.Codec == nil {
		return fmt.Errorf("NewStreamClient error %v, %v", protocol.SofaRPC, cc)
	}
	return nil
}

func (c *RPCClient) ConnectTLS(addr string, cfg *v2.TLSConfig) error {
	tlsMng, err := mtls.NewTLSClientContextManager(cfg, nil)
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
	var headers sofarpc.SofaRpcCmd
	data := buffer.NewIoBufferString(in)
	switch c.Protocol {
	case Bolt1:
		headers = BuildBoltV1RequestWithContent(ID, data)
	case Bolt2:
		headers = BuildBoltV2Request(ID)
	default:
		c.t.Errorf("unsupport protocol")
		return
	}
	requestEncoder.AppendHeaders(context.Background(), headers, false)
	requestEncoder.AppendData(context.Background(), data, true)
	atomic.AddUint32(&c.requestCount, 1)
	c.Waits.Store(streamID, streamID)
}

func (c *RPCClient) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	c.OnReceiveHeaders(ctx, headers, true)
}
func (c *RPCClient) OnReceiveData(context context.Context, data types.IoBuffer, endStream bool) {
}
func (c *RPCClient) OnReceiveTrailers(context context.Context, trailers types.HeaderMap) {
}
func (c *RPCClient) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
}
func (c *RPCClient) OnReceiveHeaders(context context.Context, headers types.HeaderMap, endStream bool) {
	if cmd, ok := headers.(sofarpc.SofaRpcCmd); ok {
		streamID := protocol.StreamIDConv(cmd.RequestID())

		if _, ok := c.Waits.Load(streamID); ok {
			c.t.Logf("RPC client receive streamId:%s \n", streamID)
			atomic.AddUint32(&c.respCount, 1)
			// add status check
			if resp, ok := cmd.(rpc.RespStatus); ok {
				status := int16(resp.RespStatus())
				if status == c.ExpectedStatus {
					c.Waits.Delete(streamID)
				}
			}
		} else {
			c.t.Errorf("get a unexpected stream ID %s", streamID)
		}
	} else {
		c.t.Errorf("get a unexpected header type")
	}
}

func BuildBoltV1RequestWithContent(requestID uint64, data types.IoBuffer) *sofarpc.BoltRequest {
	request := &sofarpc.BoltRequest{
		Protocol:   sofarpc.PROTOCOL_CODE_V1,
		CmdType:    sofarpc.REQUEST,
		CmdCode:    sofarpc.RPC_REQUEST,
		Version:    1,
		ReqID:      uint32(requestID),
		Codec:      sofarpc.HESSIAN2_SERIALIZE,
		Timeout:    -1,
		ContentLen: data.Len(),
	}
	return buildBoltV1Request(request)

}

func BuildBoltV1Request(requestID uint64) *sofarpc.BoltRequest {
	request := &sofarpc.BoltRequest{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.RPC_REQUEST,
		Version:  1,
		ReqID:    uint32(requestID),
		Codec:    sofarpc.HESSIAN2_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}
	return buildBoltV1Request(request)
}

func buildBoltV1Request(request *sofarpc.BoltRequest) *sofarpc.BoltRequest {

	headers := map[string]string{"service": "testSofa"} // used for sofa routing

	if headerBytes, err := serialize.Instance.Serialize(headers); err != nil {
		panic("serialize headers error")
	} else {
		request.HeaderMap = headerBytes
		request.HeaderLen = int16(len(headerBytes))
	}

	return request
}

func BuildBoltV2Request(requestID uint64) *sofarpc.BoltRequestV2 {
	//TODO:
	return nil
}

func BuildBoltV1Response(req *sofarpc.BoltRequest) *sofarpc.BoltResponse {
	return &sofarpc.BoltResponse{
		Protocol:       req.Protocol,
		CmdType:        sofarpc.RESPONSE,
		CmdCode:        sofarpc.RPC_RESPONSE,
		Version:        req.Version,
		ReqID:          req.ReqID,
		Codec:          req.Codec, //todo: read default codec from config
		ResponseStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
		HeaderLen:      req.HeaderLen,
		HeaderMap:      req.HeaderMap,
	}
}
func BuildBoltV2Response(req *sofarpc.BoltRequestV2) *sofarpc.BoltResponseV2 {
	//TODO:
	return nil
}

type RPCServer struct {
	UpstreamServer
	Client *RPCClient
	// Statistic
	Name  string
	Count uint32
}

func NewRPCServer(t *testing.T, addr string, proto string) UpstreamServer {
	s := &RPCServer{
		Client: NewRPCClient(t, "rpcClient", proto),
		Name:   addr,
	}
	switch proto {
	case Bolt1:
		s.UpstreamServer = NewUpstreamServer(t, addr, s.ServeBoltV1)
	case Bolt2:
		s.UpstreamServer = NewUpstreamServer(t, addr, s.ServeBoltV2)
	default:
		t.Errorf("unsupport protocol")
		return nil
	}
	return s
}

func (s *RPCServer) ServeBoltV1(t *testing.T, conn net.Conn) {
	response := func(iobuf types.IoBuffer) ([]byte, bool) {
		cmd, _ := codec.BoltCodec.Decode(nil, iobuf)
		if cmd == nil {
			return nil, false
		}
		if req, ok := cmd.(*sofarpc.BoltRequest); ok {
			atomic.AddUint32(&s.Count, 1)
			resp := BuildBoltV1Response(req)
			iobufresp, err := codec.BoltCodec.Encode(nil, resp)
			if err != nil {
				t.Errorf("Build response error: %v\n", err)
				return nil, true
			}
			return iobufresp.Bytes(), true
		}
		return nil, true
	}
	ServeSofaRPC(t, conn, response)

}
func (s *RPCServer) ServeBoltV2(t *testing.T, conn net.Conn) {
	//TODO:
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
