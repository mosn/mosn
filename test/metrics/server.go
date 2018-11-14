package main

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/codec"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
	"golang.org/x/net/http2"
)

type Server interface {
	Start() error
}

type ServeConn func(conn net.Conn)

func StartServer(addr string, serve ServeConn) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return err
		}
		go serve(conn)
	}
}

type RPCServer struct {
	addr string
}

func NewRPCServer(addr string) Server {
	return &RPCServer{
		addr,
	}
}

func (s *RPCServer) ServeConn(conn net.Conn) {
	iobuf := buffer.NewIoBuffer(102400)
	for {
		now := time.Now()
		conn.SetReadDeadline(now.Add(30 * time.Second))
		buf := make([]byte, 10*1024)
		bytesRead, err := conn.Read(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				continue
			}
			return
		}
		if bytesRead > 0 {
			iobuf.Write(buf[:bytesRead])
			for iobuf.Len() > 1 {
				data, ok := s.serveBoltV1(iobuf)
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
func (s *RPCServer) serveBoltV1(iobuf types.IoBuffer) ([]byte, bool) {
	cmd, _ := codec.BoltV1.GetDecoder().Decode(nil, iobuf)
	if cmd == nil {
		return nil, false
	}
	if req, ok := cmd.(*sofarpc.BoltRequestCommand); ok {
		resp := util.BuildBoltV1Response(req)
		iobufresp, err := codec.BoltV1.GetEncoder().EncodeHeaders(nil, resp)
		if err != nil {
			return nil, true
		}
		return iobufresp.Bytes(), true
	}
	return nil, true
}
func (s *RPCServer) Start() error {
	return StartServer(s.addr, s.ServeConn)
}

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	for k := range r.Header {
		w.Header().Set(k, r.Header.Get(k))
	}
	fmt.Fprintf(w, "\nRequestId:%s\n", r.Header.Get("Requestid"))
}

type HTTPServer struct {
	addr string
}

func NewHTTP1Server(addr string) Server {
	return &HTTPServer{addr}
}
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", ServeHTTP)
	return http.ListenAndServe(s.addr, mux)
}

type HTTP2Server struct {
	addr   string
	server *http2.Server
}

func NewHTTP2Server(addr string) Server {
	return &HTTP2Server{
		addr:   addr,
		server: &http2.Server{IdleTimeout: 1 * time.Minute},
	}
}
func (s *HTTP2Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ServeHTTP(w, r)
}

func (s *HTTP2Server) ServeConn(conn net.Conn) {
	s.server.ServeConn(conn, &http2.ServeConnOpts{Handler: s})
}

func (s *HTTP2Server) Start() error {
	return StartServer(s.addr, s.ServeConn)
}
