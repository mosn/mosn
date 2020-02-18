package sofarpc

import (
	"net"
	"sync"
	"time"

	"mosn.io/pkg/buffer"
)

type MockServer struct {
	Addr      string
	ServeFunc ServeFunc
	// runtime
	mutex       sync.Mutex
	listener    net.Listener
	id          uint64
	connections map[uint64]net.Conn
	delay       time.Duration // wait delay time before response
	idleTimeout time.Duration // if idle timeout is not zero, set a connection timeout, and will close the connection when timeout
	// stats
	ServerStats *ServerStats
}

func NewMockServer(addr string, f ServeFunc) *MockServer {
	if f == nil {
		f = DefaultBoltV1Serve.Serve
	}
	return &MockServer{
		Addr:        addr,
		ServeFunc:   f,
		mutex:       sync.Mutex{},
		connections: make(map[uint64]net.Conn),
		ServerStats: NewServerStats(),
	}
}

func (srv *MockServer) SetResponseDelay(d time.Duration) {
	srv.delay = d
}
func (srv *MockServer) SetConnectionIdleTimeout(d time.Duration) {
	srv.idleTimeout = d
}

func (srv *MockServer) Close() {
	srv.mutex.Lock()
	defer srv.mutex.Unlock()
	if srv.listener == nil {
		return
	}
	srv.listener.Close()
	srv.listener = nil

	for key, conn := range srv.connections {
		conn.Close()
		delete(srv.connections, key)
		srv.ServerStats.CloseConnection()
	}
}

func (srv *MockServer) Start() {
	srv.mutex.Lock()
	if srv.listener != nil {
		srv.mutex.Unlock()
		return
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		panic(err)
	}
	srv.listener = ln
	srv.mutex.Unlock()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return
		}
		srv.ServerStats.ActiveConnection()

		srv.mutex.Lock()
		srv.id++
		srv.connections[srv.id] = conn
		srv.mutex.Unlock()

		go srv.Serve(srv.id, conn)
	}
}

func (srv *MockServer) Serve(id uint64, conn net.Conn) {
	defer func() {
		srv.ServerStats.CloseConnection()

		srv.mutex.Lock()
		delete(srv.connections, id)
		srv.mutex.Unlock()
	}()

	iobuf := buffer.NewIoBuffer(10240)
	for {
		if srv.idleTimeout == 0 {
			conn.SetReadDeadline(time.Time{}) // no timeout
		} else {
			conn.SetReadDeadline(time.Now().Add(srv.idleTimeout))
		}
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			return
		}
		if n > 0 {
			iobuf.Write(buf[:n])
		DECODE:
			for iobuf.Len() > 0 {
				resp := srv.ServeFunc(iobuf)
				if resp == nil {
					break DECODE
				}
				srv.ServerStats.Request()
				if srv.delay > 0 {
					time.Sleep(srv.delay)
				}
				conn.Write(resp.DataToWrite)
				srv.ServerStats.Response(resp.Status)

			}
		}

	}
}
