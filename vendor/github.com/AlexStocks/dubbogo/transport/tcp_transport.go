// Copyright (c) 2015 Asim Aslam.
// Copyright (c) 2016 ~ 2018, Alex Stocks.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package transport

import (
	"io"
	"net"
	"runtime"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/common"
)

const (
	DefaultTCPReadBufferSize  = 256 * 1024
	DefaultTCPWriteBufferSize = 128 * 1024
)

//////////////////////////////////////////////
// tcp transport socket
//////////////////////////////////////////////

type tcpTransportSocket struct {
	t       *tcpTransport
	conn    net.Conn
	timeout time.Duration
	release func()
}

func initTCPTransportSocket(t *tcpTransport, c net.Conn, release func()) *tcpTransportSocket {
	return &tcpTransportSocket{
		t:       t,
		conn:    c,
		release: release,
	}
}

func (t *tcpTransportSocket) Reset(c net.Conn, release func()) {
	t.Close()
	t.conn = c
	t.release = release
}

func (t *tcpTransportSocket) Recv(p *Package) error {
	if p == nil {
		return jerrors.Errorf("message passed in is nil")
	}

	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}

	t.conn.Read(p.Body)

	return nil
}

func (t *tcpTransportSocket) Send(p *Package) error {
	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}

	n, err := t.conn.Write(p.Body)
	if err != nil {
		return jerrors.Trace(err)
	}
	if n < len(p.Body) {
		return jerrors.Annotatef(io.ErrShortWrite, "t.conn.Write(buf len:%d) = len:%d", len(p.Body), n)
	}

	return nil
}

func (t *tcpTransportSocket) Close() error {
	return jerrors.Trace(t.conn.Close())
}

func (t *tcpTransportSocket) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *tcpTransportSocket) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

//////////////////////////////////////////////
// tcp transport client
//////////////////////////////////////////////

type tcpTransportClient struct {
	t    *tcpTransport
	conn net.Conn
}

func initTCPTransportClient(t *tcpTransport, conn net.Conn) *tcpTransportClient {
	return &tcpTransportClient{
		t:    t,
		conn: conn,
	}
}

func (t *tcpTransportClient) Send(p *Package) error {
	if t.t.opts.Timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.t.opts.Timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}

	n, err := t.conn.Write(p.Body)
	if err != nil {
		return jerrors.Trace(err)
	}
	if n < len(p.Body) {
		return jerrors.Annotatef(io.ErrShortWrite, "t.conn.Write(buf len:%d) = len:%d", len(p.Body), n)
	}

	return nil
}

// tcp connection read
func (t *tcpTransportClient) read(p *Package) error {
	var (
		err    error
		bufLen int
		buf    []byte
	)

	buf = make([]byte, 4096)
	bufLen, err = t.conn.Read(buf)
	if err != nil {
		return jerrors.Trace(err)
	}
	if bufLen == 0 {
		return io.EOF
	}
	p.Body = append(p.Body, buf[:bufLen]...)

	return nil
}

func (t *tcpTransportClient) Recv(p *Package) error {
	if t.t.opts.Timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.t.opts.Timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}

	return jerrors.Trace(t.read(p))
}

func (t *tcpTransportClient) Close() error {
	return jerrors.Trace(t.conn.Close())
}

//////////////////////////////////////////////
// tcp transport listener
//////////////////////////////////////////////

type tcpTransportListener struct {
	t        *tcpTransport
	listener net.Listener
	sem      chan struct{}
}

func initTCPTransportListener(t *tcpTransport, listener net.Listener) *tcpTransportListener {
	return &tcpTransportListener{
		t:        t,
		listener: listener,
		sem:      make(chan struct{}, DefaultMAXConnNum),
	}
}

func (t *tcpTransportListener) acquire() { t.sem <- struct{}{} }
func (t *tcpTransportListener) release() { <-t.sem }

func (t *tcpTransportListener) Addr() string {
	return t.listener.Addr().String()
}

func (t *tcpTransportListener) Close() error {
	return t.listener.Close()
}

func (t *tcpTransportListener) Accept(fn func(Socket)) error {
	var (
		err       error
		c         net.Conn
		ok        bool
		ne        net.Error
		tempDelay time.Duration
	)

	for {
		t.acquire() // 若connect chan已满,则会阻塞在此处
		c, err = t.listener.Accept()
		if err != nil {
			t.release()
			if ne, ok = err.(net.Error); ok && ne.Temporary() {
				if tempDelay != 0 {
					tempDelay <<= 1
				} else {
					tempDelay = 5 * time.Millisecond
				}
				if tempDelay > DefaultMaxSleepTime {
					tempDelay = DefaultMaxSleepTime
				}
				log.Info("htp: Accept error: %v; retrying in %v\n", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return jerrors.Trace(err)
		}

		if tcpConn, ok := c.(*net.TCPConn); ok {
			tcpConn.SetReadBuffer(DefaultTCPReadBufferSize)
			tcpConn.SetWriteBuffer(DefaultTCPWriteBufferSize)
		}

		sock := initTCPTransportSocket(t.t, c, t.release)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					log.Error("tcp: panic serving %v: %v\n%s", c.RemoteAddr(), r, buf)
					sock.Close() // 遇到错误退出的时候保证socket fd的回收
				}
			}()

			fn(sock) // rpcServer:handlePkg 函数里面有一个defer语句段，保证了正常退出的情况下sock.Close()
		}()
	}
}

//////////////////////////////////////////////
// tcp transport
//////////////////////////////////////////////

type tcpTransport struct {
	opts Options
}

func (t *tcpTransport) Options() *Options {
	return &t.opts
}

func (t *tcpTransport) Dial(addr string, opts ...DialOption) (Client, error) {
	dopts := DialOptions{
		Timeout: DefaultDialTimeout,
	}
	for _, opt := range opts {
		opt(&dopts)
	}

	conn, err := net.DialTimeout("tcp", addr, dopts.Timeout)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	return initTCPTransportClient(t, conn), nil
}

func (t *tcpTransport) Listen(addr string, opts ...ListenOption) (Listener, error) {
	var options ListenOptions
	for _, o := range opts {
		o(&options)
	}

	fn := func(addr string) (net.Listener, error) {
		return net.Listen("tcp", addr)
	}
	l, err := listen(addr, fn)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	return initTCPTransportListener(t, l), nil
}

func (t *tcpTransport) String() string {
	return "tcp-transport"
}

func newTCPTransport(opts ...Option) Transport {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	return &tcpTransport{opts: options}
}
