package http

import (
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/types"
)

func init() {
	lib.RegisterCreateServer("Http2", NewHTTP2Server)
	lib.RegisterCreateClient("Http2", NewHttp2Client)
}

type MockHttp2Server struct {
	*MockHttpServer
}

func NewHTTP2Server(config interface{}) types.MockServer {
	s := NewHTTPServer(config)
	return &MockHttp2Server{
		MockHttpServer: s.(*MockHttpServer),
	}
}

func (s *MockHttp2Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Proto != "HTTP/2.0" {
		w.WriteHeader(http.StatusHTTPVersionNotSupported)
		return
	}
	s.MockHttpServer.ServeHTTP(w, r)
}

func (s *MockHttp2Server) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.listener != nil {
		return
	}
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.DefaultLogger.Fatalf("listen %s failed", s.addr)
	}
	s.listener = NewStatsListener(ln, s.stats)

	h2s := &http2.Server{IdleTimeout: 1 * time.Minute}
	handler := http.HandlerFunc(s.ServeHTTP)
	s.server.Handler = h2c.NewHandler(handler, h2s)
	go s.server.Serve(s.listener)
}

func NewHttp2Client(config interface{}) types.MockClient {
	client := NewHttpClient(config)
	rc := client.(*MockHttpClient)
	rc.protocolName = protocol.HTTP2
	return rc
}
