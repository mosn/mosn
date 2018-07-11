package tests

import (
	"net"
	"sync"
	"testing"

	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
)

type ServeConn func(t *testing.T, conn net.Conn)
type UpstreamServer struct {
	Listener net.Listener
	Serve    ServeConn
	conns    []net.Conn
	mu       sync.Mutex
	t        *testing.T
}

func NewUpstreamServer(t *testing.T, addr string, serve ServeConn) *UpstreamServer {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen %s failed, error: %v\n", addr, err)
		return nil
	}
	return &UpstreamServer{
		Listener: l,
		conns:    []net.Conn{},
		mu:       sync.Mutex{},
		Serve:    serve,
		t:        t,
	}

}
func (s *UpstreamServer) GoServe() {
	go s.serve()
}
func (s *UpstreamServer) serve() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				s.t.Logf("Accept error: %v\n", err)
				continue
			}
			return
		}
		s.t.Logf("server %s Accept connection: %s\n", s.Listener.Addr().String(), conn.RemoteAddr().String())
		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()
		go s.Serve(s.t, conn)
	}
}

func (s *UpstreamServer) Close() {
	s.mu.Lock()
	s.t.Logf("server %s closed\n", s.Listener.Addr().String())
	s.Listener.Close()
	for _, conn := range s.conns {
		conn.Close()
	}
	s.mu.Unlock()
}

//BoltV1
func buildRespMag(req *sofarpc.BoltRequestCommand) *sofarpc.BoltResponseCommand {
	return &sofarpc.BoltResponseCommand{
		Protocol:       req.Protocol,
		CmdType:        sofarpc.RESPONSE,
		CmdCode:        sofarpc.RPC_RESPONSE,
		Version:        req.Version,
		ReqId:          req.ReqId,
		CodecPro:       req.CodecPro, //todo: read default codec from config
		ResponseStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
		HeaderLen:      req.HeaderLen,
		HeaderMap:      req.HeaderMap,
	}
}

// build request msg
// 22固定字节头部 + headermap长度
func buildRequestMsg(requestId uint32) *sofarpc.BoltRequestCommand {

	request := &sofarpc.BoltRequestCommand{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.RPC_REQUEST,
		Version:  1,
		ReqId:    requestId,
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
