package tests

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/orcaman/concurrent-map"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"golang.org/x/net/http2"
)

//Common UpstreamServer
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

//Server Implement
type Http2Server struct {
	t      *testing.T
	Server *http2.Server
}

func (s *Http2Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.t.Logf("[server] Recieve request\n")
	w.Header().Set("Content-Type", "text/plain")

	for k, _ := range r.Header {
		w.Header().Set(k, r.Header.Get(k))
	}

	fmt.Fprintf(w, "\nRequestId:%s\n", r.Header.Get("Requestid"))

}
func (s *Http2Server) ServeConn(t *testing.T, conn net.Conn) {
	s.Server.ServeConn(conn, &http2.ServeConnOpts{Handler: s})
}

func NewUpstreamHttp2(t *testing.T, addr string) *UpstreamServer {
	s := &Http2Server{
		t:      t,
		Server: &http2.Server{IdleTimeout: 1 * time.Minute},
	}
	return NewUpstreamServer(t, addr, s.ServeConn)
}

type Http2Response struct {
	re *regexp.Regexp
}

func (resp *Http2Response) Filter(data string, records cmap.ConcurrentMap) {
	if resp.re == nil {
		resp.re = regexp.MustCompile("\nRequestId:[0-9]+\n")
	}
	bodys := strings.Split(
		strings.Trim(resp.re.FindString(data), "\n"), ":",
	)
	if len(bodys) == 2 {
		requestId := bodys[1]
		if _, ok := records.Get(requestId); ok {
			records.Remove(requestId)
		}
	}
}

//Http Server
//use in net/http/httptest
type HttpServer struct {
	t    *testing.T
	name string
}

func (s *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.t.Logf("server %s Recieve request\n", s.name)
	w.Header().Set("Content-Type", "text/plain")
	for k, _ := range r.Header {
		w.Header().Set(k, r.Header.Get(k))
	}
	fmt.Fprintf(w, "\nServerName:%s\n", s.name)
	fmt.Fprintf(w, "\nRequestId:%s\n", r.Header.Get("Requestid"))
}

type ReponseFilter interface {
	Filter(data string, records cmap.ConcurrentMap)
}

//Rpc client
type RpcClient struct {
	t               *testing.T
	conn            types.ClientConnection
	addr            string
	response_filter ReponseFilter
	wait_reponse    cmap.ConcurrentMap
}

//types.ReadFilter
func (c *RpcClient) OnNewConnection() types.FilterStatus {
	return types.Continue
}
func (c *RpcClient) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}
func (c *RpcClient) OnData(buffer types.IoBuffer) types.FilterStatus {
	c.t.Logf("[client] receive data: \n")
	//c.t.Logf("%s\n", buffer.String())
	resp := buffer.String()
	c.response_filter.Filter(resp, c.wait_reponse)
	buffer.Reset()
	return types.Continue
}

//types.ConnectionEventListener
func (c *RpcClient) OnEvent(event types.ConnectionEvent) {}

func (c *RpcClient) Connect() error {
	stopChan := make(chan bool)
	remoteAddr, _ := net.ResolveTCPAddr("tcp", c.addr)
	cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
	c.conn = cc
	cc.AddConnectionEventListener(c)
	cc.FilterManager().AddReadFilter(c)
	if err := cc.Connect(true); err != nil {
		//c.t.Logf("[client] connection failed\n")
		return err
	}
	return nil
}
func (c *RpcClient) SendRequest(id uint32, req []byte) {
	c.conn.Write(buffer.NewIoBufferBytes(req))
	c.wait_reponse.Set(fmt.Sprintf("%d", id), id)
}

//Protocols
func buildBoltV1Request(requestId uint32) *sofarpc.BoltRequestCommand {
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
func buildBoltV1Resposne(req *sofarpc.BoltRequestCommand) *sofarpc.BoltResponseCommand {
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

//SofaRpc Serve
func ServeBoltV1(t *testing.T, conn net.Conn) {
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
				_, cmd := codec.BoltV1.GetDecoder().Decode(nil, iobuf)
				if cmd == nil {
					break
				}
				if req, ok := cmd.(*sofarpc.BoltRequestCommand); ok {
					resp := buildBoltV1Resposne(req)
					err, iobufresp := codec.BoltV1.GetEncoder().EncodeHeaders(nil, resp)
					if err != nil {
						t.Errorf("Build response error: %v\n", err)
					} else {
						t.Logf("server %s write to remote: %d\n", conn.LocalAddr().String(), resp.GetReqId)
						respdata := iobufresp.Bytes()
						conn.Write(respdata)
					}
				}
			}
		}
	}

}
