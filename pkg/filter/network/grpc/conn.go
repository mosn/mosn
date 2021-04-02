package grpc

import (
	"fmt"
	"io"
	"net"
	"syscall"
	"time"

	"go.uber.org/atomic"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/buffer"
)

type Connection struct {
	// Reader channel
	r chan buffer.IoBuffer
	// endRead channel send messages that read finished
	endRead chan struct{}
	// raw connections
	raw api.Connection
	//
	closed atomic.Bool
	event  api.ConnectionEvent
}

func NewConn(c api.Connection) *Connection {
	conn := &Connection{
		r:       make(chan buffer.IoBuffer),
		endRead: make(chan struct{}),
		raw:     c,
	}
	// the connection should be closed when real connection is closed
	c.AddConnectionEventListener(conn)
	return conn
}

var _ net.Conn = &Connection{}

func (c *Connection) Read(b []byte) (n int, err error) {
	data, ok := <-c.r
	if !ok { // connction closed
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("grpc connection read error: %s", c.event)
		}
		if c.event == api.RemoteClose {
			return 0, io.EOF
		}
		rc := c.raw.RawConn()
		return 0, &net.OpError{
			Op:     "read",
			Net:    rc.LocalAddr().Network(),
			Source: rc.LocalAddr(),
			Addr:   rc.RemoteAddr(),
			Err:    fmt.Errorf("connection has been closed by %s", c.event),
		}
	}
	n = copy(b, data.Bytes())
	data.Drain(n)
	c.endRead <- struct{}{}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc connection read data:  %d, %v", n, err)
	}
	return
}

func (c *Connection) Write(b []byte) (n int, err error) {
	n = len(b)
	buf := buffer.GetIoBuffer(n)
	buf.Write(b)
	err = c.raw.Write(buf) // write directly to raw connection
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc connection write data: %d, %v", n, err)
	}
	return
}

func (c *Connection) Close() error {
	if !c.closed.CAS(false, true) {
		return syscall.EINVAL
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc connection: closed")
	}
	close(c.r)
	close(c.endRead)
	return nil
}

func (c *Connection) LocalAddr() net.Addr {
	return c.raw.LocalAddr()
}

func (c *Connection) RemoteAddr() net.Addr {
	return c.raw.RemoteAddr()
}

// The real connection timeout is managed by MOSN.
// This connection's timeout takes no effect on real connection.
// If the real connection reads timeout, no data send to this connection
func (c *Connection) SetDeadline(t time.Time) error {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc connection: set a deadline: %v", t)
	}
	return nil
}

func (c *Connection) SetReadDeadline(t time.Time) error {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc connection: set a read deadline: %v", t)
	}
	return nil
}

func (c *Connection) SetWriteDeadline(t time.Time) error {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc connection: set a write deadline: %v", t)
	}
	return nil
}

func (c *Connection) OnEvent(event api.ConnectionEvent) {
	if c.closed.Load() {
		return
	}
	if event.IsClose() {
		c.event = event
		c.Close()
	}
}

// Send awakes connection Read, and will wait Read finished.
func (c *Connection) Send(buf buffer.IoBuffer) {
	for buf.Len() > 0 {
		c.r <- buf
		<-c.endRead
	}
}
