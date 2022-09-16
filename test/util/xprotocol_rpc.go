package util

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/basef"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/thrift/lib/go/thrift"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/protocol/xprotocol/boltv2"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbothrift"
	"mosn.io/mosn/pkg/protocol/xprotocol/tars"
	"mosn.io/mosn/pkg/stream"
	xstream "mosn.io/mosn/pkg/stream/xprotocol"
	"mosn.io/mosn/pkg/trace"
	tracehttp "mosn.io/mosn/pkg/trace/sofa/http"
	xtrace "mosn.io/mosn/pkg/trace/sofa/xprotocol"
	tracebolt "mosn.io/mosn/pkg/trace/sofa/xprotocol/bolt"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

func init() {
	// tracer driver register
	trace.RegisterDriver("SOFATracer", trace.NewDefaultDriverImpl())
	// xprotocol action register
	xprotocol.RegisterXProtocolAction(xstream.NewConnPool, xstream.NewStreamFactory, func(codec api.XProtocolCodec) {
		name := codec.ProtocolName()
		trace.RegisterTracerBuilder("SOFATracer", name, xtrace.NewTracer)
	})
	// xprotocol register
	_ = xprotocol.RegisterXProtocolCodec(&bolt.XCodec{})
	_ = xprotocol.RegisterXProtocolCodec(&boltv2.XCodec{})
	_ = xprotocol.RegisterXProtocolCodec(&dubbo.XCodec{})
	_ = xprotocol.RegisterXProtocolCodec(&dubbothrift.XCodec{})
	_ = xprotocol.RegisterXProtocolCodec(&tars.XCodec{})
	// trace register
	xtrace.RegisterDelegate(bolt.ProtocolName, tracebolt.Boltv1Delegate)
	xtrace.RegisterDelegate(boltv2.ProtocolName, tracebolt.Boltv1Delegate)
	trace.RegisterTracerBuilder("SOFATracer", protocol.HTTP1, tracehttp.NewTracer)
}

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
	rpcClient := &RPCClient{
		t:        t,
		ClientID: id,
		Protocol: proto,
		Waits:    sync.Map{},
	}
	switch proto {
	case bolt.ProtocolName:
		rpcClient.ExpectedStatus = int16(bolt.ResponseStatusSuccess)
	case dubbo.ProtocolName:
		rpcClient.ExpectedStatus = int16(dubbo.ResponseStatusSuccess)
	case tars.ProtocolName:
		rpcClient.ExpectedStatus = int16(tars.ResponseStatusSuccess)
	case dubbothrift.ProtocolName:
		rpcClient.ExpectedStatus = int16(dubbothrift.ResponseStatusSuccess)
	}
	return rpcClient

}

func (c *RPCClient) connect(addr string, tlsMng types.TLSClientContextManager) error {
	stopChan := make(chan struct{})
	var remoteAddr net.Addr
	remoteAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		remoteAddr, _ = net.ResolveUnixAddr("unix", addr)
	}
	cc := network.NewClientConnection(0, tlsMng, remoteAddr, stopChan)
	c.conn = cc
	if err := cc.Connect(); err != nil {
		c.t.Logf("client[%s] connect to server error: %v\n", c.ClientID, err)
		return err
	}

	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableUpstreamProtocol, c.Protocol)
	c.Codec = stream.NewStreamClient(ctx, c.Protocol, cc, nil)
	if c.Codec == nil {
		return fmt.Errorf("NewStreamClient error %v, %v", c.Protocol, cc)
	}
	return nil
}

func (c *RPCClient) ConnectTLS(addr string, cfg *v2.TLSConfig) error {
	tlsMng, err := mtls.NewTLSClientContextManager(addr, cfg)
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
		c.conn.Close(api.NoFlush, api.LocalClose)
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
	var frame api.XFrame
	data := buffer.NewIoBufferString(in)
	// TODO: support boltv2
	switch c.Protocol {
	case bolt.ProtocolName:
		// header used for sofa routing
		frame = bolt.NewRpcRequest(uint32(ID), protocol.CommonHeader(map[string]string{"service": "testSofa"}), data)
	case dubbo.ProtocolName:
		frame = dubbo.NewRpcRequest(protocol.CommonHeader(map[string]string{"service": "testDubbo"}), buffer.NewIoBufferBytes(buildDubboRequest(ID)))
	case tars.ProtocolName:
		frame = tars.NewRpcRequest(protocol.CommonHeader(map[string]string{"service": "testTars"}), buffer.NewIoBufferBytes(buildTarsRequest(ID)))
	case dubbothrift.ProtocolName:
		frame = dubbothrift.NewRpcRequest(protocol.CommonHeader(map[string]string{"service": "testThrift"}), buffer.NewIoBufferBytes(buildDubboThriftRequest(ID)))
	default:
		c.t.Errorf("unsupport protocol")
		return
	}
	c.Waits.Store(streamID, streamID)
	requestEncoder.AppendHeaders(context.Background(), frame.GetHeader(), false)
	requestEncoder.AppendData(context.Background(), data, true)
	atomic.AddUint32(&c.requestCount, 1)
}

func (c *RPCClient) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	if cmd, ok := headers.(api.XRespFrame); ok {
		streamID := protocol.StreamIDConv(cmd.GetRequestId())

		if _, ok := c.Waits.Load(streamID); ok {
			c.t.Logf("RPC client receive streamId:%s, status: %v \n", streamID, cmd.GetStatusCode())
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

func buildDubboRequest(requestId uint64) []byte {
	service := hessian.Service{
		Path:      "com.alipay.test",
		Interface: "test",
		Group:     "test",
		Version:   "v1",
		Method:    "testCall",
	}
	codec := hessian.NewHessianCodec(nil)
	header := hessian.DubboHeader{
		SerialID: 2,
		Type:     hessian.PackageRequest,
		ID:       int64(requestId),
	}
	body := hessian.NewRequest([]interface{}{}, nil)
	reqData, err := codec.Write(service, header, body)
	if err != nil {
		return nil
	}
	return reqData
}

func buildDubboThriftRequest(requestId uint64) []byte {
	bufferBytes := buffer.NewIoBuffer(1024)
	transport := thrift.NewStreamTransportW(bufferBytes)
	defer transport.Close()
	protocol := thrift.NewTBinaryProtocolTransport(transport)

	serviceName := "com.company.test"
	methodName := "test"

	//header
	transport.Write(dubbothrift.MagicTag)
	protocol.WriteI32(math.MaxInt32)
	protocol.WriteI16(math.MaxInt16)
	protocol.WriteByte(1)
	protocol.WriteString(serviceName)
	protocol.WriteI64(int64(requestId))
	protocol.Flush(nil)

	headerLen := bufferBytes.Len()

	//message body
	protocol.WriteMessageBegin(methodName, thrift.CALL, 1)
	protocol.WriteMessageEnd()
	protocol.Flush(nil)

	message := bufferBytes.Bytes()
	messageLen := len(message)

	binary.BigEndian.PutUint16(message[dubbothrift.MessageHeaderLenIdx:dubbothrift.MessageHeaderLenIdx+dubbothrift.MessageLenSize], uint16(headerLen))
	binary.BigEndian.PutUint32(message[dubbothrift.MessageLenIdx:dubbothrift.MessageLenIdx+dubbothrift.MessageLenSize], uint32(messageLen))

	data := make([]byte, messageLen+dubbothrift.MessageLenSize)
	binary.BigEndian.PutUint32(data, uint32(messageLen))
	copy(data[dubbothrift.MessageLenSize:], message)

	return data
}

func buildTarsRequest(requestId uint64) []byte {
	request := requestf.RequestPacket{
		IVersion:     basef.TARSVERSION,
		CPacketType:  basef.TARSNORMAL,
		IMessageType: basef.TARSMESSAGETYPESAMPLE,
		IRequestId:   int32(requestId),
		SServantName: "testServant",
		SFuncName:    "testFunc",
		SBuffer:      []int8{},
	}
	sbuf := bytes.NewBuffer(nil)
	sbuf.Write(make([]byte, 4))
	os := codec.NewBuffer()
	err := request.WriteTo(os)
	if err != nil {
		return nil
	}
	bs := os.ToBytes()
	sbuf.Write(bs)
	len := sbuf.Len()
	binary.BigEndian.PutUint32(sbuf.Bytes(), uint32(len))
	return sbuf.Bytes()
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
	case dubbo.ProtocolName:
		s.UpstreamServer = NewUpstreamServer(t, addr, s.ServeDubbo)
	case tars.ProtocolName:
		s.UpstreamServer = NewUpstreamServer(t, addr, s.ServeTars)
	case dubbothrift.ProtocolName:
		s.UpstreamServer = NewUpstreamServer(t, addr, s.ServeDubboThrift)
	default:
		t.Errorf("unsupport protocol")
		return nil
	}
	return s
}

func (s *RPCServer) ServeBoltV1(t *testing.T, conn net.Conn) {
	response := func(iobuf types.IoBuffer) ([]byte, bool) {
		protocol := (&bolt.XCodec{}).NewXProtocol(context.Background())
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
	ServeRPC(t, conn, response)

}

func (s *RPCServer) ServeDubbo(t *testing.T, conn net.Conn) {
	response := func(iobuf types.IoBuffer) ([]byte, bool) {
		protocol := (&dubbo.XCodec{}).NewXProtocol(context.Background())
		cmd, _ := protocol.Decode(context.Background(), iobuf)
		if cmd == nil {
			return nil, false
		}
		if req, ok := cmd.(*dubbo.Frame); ok {
			t.Logf("RPC Server receive streamId: %d \n", req.Id)
			atomic.AddUint32(&s.Count, 1)
			resp := dubbo.NewRpcResponse(nil, buffer.NewIoBufferBytes(buildDubboResponse(req.Id)))
			ioBufResp, err := protocol.Encode(context.Background(), resp)
			if err != nil {
				t.Errorf("Build response error: %v\n", err)
				return nil, true
			}
			return ioBufResp.Bytes(), true
		} else {
			t.Logf("Unrecognized request:%+v \n", cmd)
		}
		return nil, true
	}
	ServeRPC(t, conn, response)

}

func (s *RPCServer) ServeDubboThrift(t *testing.T, conn net.Conn) {
	response := func(iobuf types.IoBuffer) ([]byte, bool) {
		protocol := (&dubbothrift.XCodec{}).NewXProtocol(context.Background())
		cmd, _ := protocol.Decode(context.Background(), iobuf)
		if cmd == nil {
			return nil, false
		}
		if req, ok := cmd.(*dubbothrift.Frame); ok {
			t.Logf("RPC Server receive streamId: %d \n", req.Id)
			atomic.AddUint32(&s.Count, 1)
			resp := dubbothrift.NewRpcResponse(nil, buffer.NewIoBufferBytes(buildDubboThriftResponse(req.Id)))
			ioBufResp, err := protocol.Encode(context.Background(), resp)
			if err != nil {
				t.Errorf("Build response error: %v\n", err)
				return nil, true
			}
			return ioBufResp.Bytes(), true
		} else {
			t.Logf("Unrecognized request:%+v \n", cmd)
		}
		return nil, true
	}
	ServeRPC(t, conn, response)

}

func (s *RPCServer) ServeTars(t *testing.T, conn net.Conn) {
	response := func(iobuf types.IoBuffer) ([]byte, bool) {
		protocol := (&tars.XCodec{}).NewXProtocol(context.Background())
		cmd, _ := protocol.Decode(context.Background(), iobuf)
		if cmd == nil {
			return nil, false
		}
		if req, ok := cmd.(*tars.Request); ok {
			t.Logf("RPC Server receive streamId: %d \n", req.GetRequestId())
			atomic.AddUint32(&s.Count, 1)
			resp := tars.NewRpcResponse(nil, buffer.NewIoBufferBytes(buildTarsResponse(req.GetRequestId())))
			ioBufResp, err := protocol.Encode(context.Background(), resp)
			if err != nil {
				t.Errorf("Build response error: %v\n", err)
				return nil, true
			}
			return ioBufResp.Bytes(), true
		} else {
			t.Logf("Unrecognized request:%+v \n", cmd)
		}
		return nil, true
	}
	ServeRPC(t, conn, response)

}

func buildDubboResponse(requestId uint64) []byte {
	service := hessian.Service{
		Path:      "com.alipay.test",
		Interface: "test",
		Group:     "test",
		Version:   "v1",
		Method:    "testCall",
	}
	codec := hessian.NewHessianCodec(nil)
	header := hessian.DubboHeader{
		SerialID:       2,
		Type:           hessian.PackageResponse,
		ID:             int64(requestId),
		ResponseStatus: uint8(dubbo.ResponseStatusSuccess),
	}
	body := hessian.NewResponse(nil, nil, nil)
	reqData, err := codec.Write(service, header, body)
	if err != nil {
		return nil
	}
	return reqData
}

func buildDubboThriftResponse(requestId uint64) []byte {
	bufferBytes := buffer.NewIoBuffer(1024)
	transport := thrift.NewStreamTransportW(bufferBytes)
	defer transport.Close()
	protocol := thrift.NewTBinaryProtocolTransport(transport)

	serviceName := "com.company.test"
	methodName := "test"

	//header
	transport.Write(dubbothrift.MagicTag)
	protocol.WriteI32(math.MaxInt32)
	protocol.WriteI16(math.MaxInt16)
	protocol.WriteByte(1)
	protocol.WriteString(serviceName)
	protocol.WriteI64(int64(requestId))
	protocol.Flush(nil)

	headerLen := bufferBytes.Len()

	//message body
	protocol.WriteMessageBegin(methodName, thrift.REPLY, 1)
	protocol.WriteMessageEnd()
	protocol.Flush(nil)

	message := bufferBytes.Bytes()
	messageLen := len(message)

	binary.BigEndian.PutUint16(message[dubbothrift.MessageHeaderLenIdx:dubbothrift.MessageHeaderLenIdx+dubbothrift.MessageLenSize], uint16(headerLen))
	binary.BigEndian.PutUint32(message[dubbothrift.MessageLenIdx:dubbothrift.MessageLenIdx+dubbothrift.MessageLenSize], uint32(messageLen))

	data := make([]byte, messageLen+dubbothrift.MessageLenSize)
	binary.BigEndian.PutUint32(data, uint32(messageLen))
	copy(data[dubbothrift.MessageLenSize:], message)

	return data
}

func buildTarsResponse(requestId uint64) []byte {
	response := requestf.ResponsePacket{
		IVersion:     basef.TARSVERSION,
		CPacketType:  basef.TARSNORMAL,
		IMessageType: basef.TARSMESSAGETYPESAMPLE,
		IRequestId:   int32(requestId),
		SBuffer:      []int8{},
		IRet:         0,
	}
	os := codec.NewBuffer()
	response.WriteTo(os)
	bs := os.ToBytes()
	sbuf := bytes.NewBuffer(nil)
	sbuf.Write(make([]byte, 4))
	sbuf.Write(bs)
	len := sbuf.Len()
	binary.BigEndian.PutUint32(sbuf.Bytes(), uint32(len))
	return sbuf.Bytes()
}

func ServeRPC(t *testing.T, conn net.Conn, responseHandler func(iobuf types.IoBuffer) ([]byte, bool)) {
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
