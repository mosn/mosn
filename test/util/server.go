package util

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

type UpstreamServer interface {
	GoServe()
	Close()
	Addr() string
}

type ServeConn func(t *testing.T, conn net.Conn)
type upstreamServer struct {
	Listener net.Listener
	Serve    ServeConn
	Address  string
	conns    []net.Conn
	mu       sync.Mutex
	t        *testing.T
	closed   bool
}

func NewUpstreamServer(t *testing.T, addr string, serve ServeConn) UpstreamServer {
	return &upstreamServer{
		Serve:   serve,
		Address: addr,
		conns:   []net.Conn{},
		mu:      sync.Mutex{},
		t:       t,
	}
}

// GoServe and Close exists concurrency
// expected action is Close should close the server, should not restart
func (s *upstreamServer) GoServe() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	//try listen
	var err error
	for i := 0; i < 3; i++ {
		s.Listener, err = net.Listen("tcp", s.Address)
		if s.Listener != nil {
			break
		}
		time.Sleep(time.Second)
	}
	if s.Listener == nil {
		s.t.Fatalf("listen %s failed, error : %v\n", s.Address, err)
	}
	go s.serve()
}
func (s *upstreamServer) serve() {
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
		if !s.closed {
			s.conns = append(s.conns, conn)
			go s.Serve(s.t, conn)
		}
		s.mu.Unlock()
	}
}

func (s *upstreamServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t.Logf("server %s closed\n", s.Address)
	if s.Listener != nil { // Close maybe called before GoServe
		s.Listener.Close()
	}
	for _, conn := range s.conns {
		conn.Close()
	}
	s.closed = true
}
func (s *upstreamServer) Addr() string {
	return s.Address
}

// Server Implement
type HTTPHandler struct{}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	fmt.Fprintf(w, "\nRequestId:%s\n", r.Header.Get("Requestid"))

}

type HTTP2Server struct {
	t       *testing.T
	Server  *http2.Server
	Handler http.Handler
}

func (s *HTTP2Server) ServeConn(t *testing.T, conn net.Conn) {
	s.Server.ServeConn(conn, &http2.ServeConnOpts{Handler: s.Handler})
}

func NewUpstreamHTTP2(t *testing.T, addr string, h http.Handler) UpstreamServer {
	if h == nil {
		h = &HTTPHandler{}
	}
	s := &HTTP2Server{
		t:       t,
		Server:  &http2.Server{IdleTimeout: 1 * time.Minute},
		Handler: h,
	}
	return NewUpstreamServer(t, addr, s.ServeConn)
}

// Http Server
type HTTPServer struct {
	t       *testing.T
	server  *httptest.Server
	Handler http.Handler
}

func (s *HTTPServer) GoServe() {
	s.server.Start()
}
func (s *HTTPServer) Close() {
	s.server.Close()
}
func (s *HTTPServer) Addr() string {
	addr := strings.Split(s.server.URL, "http://")
	if len(addr) == 2 {
		return addr[1]
	}
	return ""
}

func NewHTTPServer(t *testing.T, h http.Handler) UpstreamServer {
	if h == nil {
		h = &HTTPHandler{}
	}
	s := &HTTPServer{t: t, Handler: h}
	s.server = httptest.NewUnstartedServer(s.Handler)
	return s
}
