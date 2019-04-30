package transport

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TarsClientProtocol interface {
	Recv(pkg []byte)
	ParsePackage(buff []byte) (int, int)
}
type TarsClientConf struct {
	Proto        string
	ClientProto  TarsClientProtocol
	QueueLen     int
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type TarsClient struct {
	address string
	//TODO remove it
	conn *connection

	cp        TarsClientProtocol
	conf      *TarsClientConf
	sendQueue chan []byte
	//recvQueue chan []byte
}

type connection struct {
	tc *TarsClient

	conn     net.Conn
	connLock *sync.Mutex

	isClosed  bool
	idleTime  time.Time
	invokeNum int32
}

func NewTarsClient(address string, cp TarsClientProtocol, conf *TarsClientConf) *TarsClient {
	if conf.QueueLen <= 0 {
		conf.QueueLen = 100
	}
	sendQueue := make(chan []byte, conf.QueueLen)
	tc := &TarsClient{conf: conf, address: address, cp: cp, sendQueue: sendQueue}
	tc.conn = &connection{tc: tc, isClosed: true, connLock: &sync.Mutex{}}
	return tc
}

func (tc *TarsClient) Send(req []byte) error {
	w := tc.conn
	if err := w.reConnect(); err != nil {
		return err
	}
	tc.sendQueue <- req
	return nil
}

func (tc *TarsClient) Close() {
	w := tc.conn
	if !w.isClosed && w.conn != nil {
		w.isClosed = true
		w.conn.Close()
	}
}

func (c *connection) send(conn net.Conn) {
	var req []byte
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case req = <-c.tc.sendQueue: // Fetch jobs
		case <-t.C:
			if c.isClosed {
				return
			}
			// TODO: check one-way invoke for idle detect
			if c.invokeNum == 0 && c.idleTime.Add(c.tc.conf.IdleTimeout).Before(time.Now()) {
				c.close(conn)
				return
			}
			continue
		}
		atomic.AddInt32(&c.invokeNum, 1)
		if c.tc.conf.WriteTimeout != 0 {
			conn.SetWriteDeadline(time.Now().Add(c.tc.conf.WriteTimeout))
		}
		c.idleTime = time.Now()
		_, err := conn.Write(req)
		if err != nil {
			//TODO
			TLOG.Error("send request error:", err)
			c.close(conn)
			return
		}
	}
}

func (c *connection) recv(conn net.Conn) {
	buffer := make([]byte, 1024*4)
	var currBuffer []byte
	var n int
	var err error
	for {
		if c.tc.conf.ReadTimeout != 0 {
			conn.SetReadDeadline(time.Now().Add(c.tc.conf.ReadTimeout))
		}
		n, err = conn.Read(buffer)
		if err != nil {
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() && netErr.Temporary() {
				continue // no data, not error
			}
			if _, ok := err.(*net.OpError); ok {
				TLOG.Error("netOperror", conn.RemoteAddr())
				c.close(conn)
				return // connection is closed
			}
			if err == io.EOF {
				TLOG.Debug("connection closed by remote:", conn.RemoteAddr())
			} else {
				TLOG.Error("read packge error:", err)
			}
			c.close(conn)
			return
		}
		currBuffer = append(currBuffer, buffer[:n]...)
		for {
			pkgLen, status := c.tc.cp.ParsePackage(currBuffer)
			if status == PACKAGE_LESS {
				break
			}
			if status == PACKAGE_FULL {
				atomic.AddInt32(&c.invokeNum, -1)
				pkg := make([]byte, pkgLen-4)
				copy(pkg, currBuffer[4:pkgLen])
				currBuffer = currBuffer[pkgLen:]
				go c.tc.cp.Recv(pkg)
				if len(currBuffer) > 0 {
					continue
				}
				currBuffer = nil
				break
			}
			TLOG.Error("parse package error")
			c.close(conn)
			return
		}
	}
}

func (c *connection) reConnect() (err error) {
	c.connLock.Lock()
	if c.isClosed {
		TLOG.Debug("Connect:", c.tc.address)
		c.conn, err = net.Dial(c.tc.conf.Proto, c.tc.address)

		if err != nil {
			c.connLock.Unlock()
			return err
		}
		if c.tc.conf.Proto == "tcp" {
			if c.conn != nil {
				c.conn.(*net.TCPConn).SetKeepAlive(true)
			}
		}
		c.idleTime = time.Now()
		c.isClosed = false
		go c.recv(c.conn)
		go c.send(c.conn)
	}
	c.connLock.Unlock()
	return nil
}

func (c *connection) close(conn net.Conn) {
	c.connLock.Lock()
	c.isClosed = true
	if conn != nil {
		conn.Close()
	}
	c.connLock.Unlock()
}
