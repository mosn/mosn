package network

import (
	"net"
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"bytes"
	"io"
	"sync"
	"crypto/tls"
	"fmt"
	"time"
)

type connection struct {
	localAddr  net.Addr
	remoteAddr net.Addr

	stopChan                         chan bool
	nextProtocol                     string
	noDelay                          bool
	readEnabled                      bool
	readEnabledChan                  chan bool
	readDisableCount                 int
	detectEarlyCloseWhenReadDisabled bool
	localAddressRestored             bool
	aboveHighWatermark               bool
	bufferLimit                      uint32
	rawConnection                    net.Conn
	closeWithFlush                   bool
	connCallbacks                    []types.ConnectionCallbacks
	bytesSendCallbacks               []func(bytesSent uint64)
	filterManager                    types.FilterManager

	readBuffer       *bytes.Buffer
	writeMux         sync.Mutex
	writeBufferEntry *buffer.WriteBufferPoolEntry
	writeBuffer      *[]byte
	readerPool       *buffer.ReaderBufferPool
	writerPool       *buffer.WriteBufferPool

	startOnce sync.Once
}

func NewServerConnection(rawc net.Conn, stopChan chan bool) types.Connection {
	conn := &connection{
		rawConnection:   rawc,
		localAddr:       rawc.LocalAddr(),
		remoteAddr:      rawc.RemoteAddr(),
		stopChan:        stopChan,
		readEnabled:     false,
		readEnabledChan: make(chan bool),
		readBuffer:      &bytes.Buffer{},
		readerPool:      buffer.NewReadBufferPool(128, 4*1024),
		writerPool:      buffer.NewWriteBufferPool(128, 4*1024),
	}

	conn.filterManager = newFilterManager(conn)

	return conn
}

// basic

func (c *connection) Start(lctx context.Context) {
	c.startOnce.Do(func() {
		// TODO: panic recover

		go func() {
			defer func() {
				if p := recover(); p != nil {
					fmt.Printf("panic %v", p)
				}
			}()

			c.startReadLoop()
		}()
	})
}

// sync write buf to underlying conn io
// TODO: write buf in a async way
func (c *connection) Write(buf *bytes.Buffer) error {
	c.writeMux.Lock()
	defer c.writeMux.Unlock()

	bufBytes := buf.Bytes()
	c.writeBuffer = &bufBytes
	fs := c.filterManager.OnWrite()
	c.writeBuffer = nil

	if fs == types.StopIteration {
		return nil
	}

	wbe := c.writerPool.Take(c.rawConnection)
	c.writeBufferEntry = wbe
	defer func() {
		c.writerPool.Give(wbe)
	}()

	var bytesSent int64

	for {
		n, err := buf.WriteTo(wbe.Br)

		if err == io.EOF {
			// remote conn closed
			c.closeConnection(types.RemoteClose)
		}

		if err != nil {
			c.closeConnection(types.OnWriteErrClose)
			return err
		}

		if n == 0 {
			break
		}

		bytesSent += n
	}

	err := wbe.Br.Flush()

	if err == io.EOF {
		// remote conn closed
		c.closeConnection(types.RemoteClose)
	}

	if err != nil {
		c.closeConnection(types.OnWriteErrClose)
		return err
	}

	if bytesSent > 0 {
		for _, cb := range c.bytesSendCallbacks {
			cb(uint64(bytesSent))
		}
	}

	return nil
}

func (c *connection) Close(ccType types.ConnectionCloseType) error {
	if ccType == types.FlushWrite {
		if c.writeBufferEntry != nil {
			if c.writeBufferEntry.Br.Buffered() > 0 {
				c.closeWithFlush = true
				c.readEnabled = false
				c.readEnabledChan <- false

				for {
					if c.writeBufferEntry.Br.Buffered() > 0 {
						if err := c.writeBufferEntry.Br.Flush(); err != nil {
							if err == io.EOF {
								break
							}

							return err
						}
					} else {
						break
					}
				}
			}
		}
	}

	c.closeConnection(types.LocalClose)

	return nil
}

func (c *connection) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *connection) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *connection) AddConnectionCallbacks(cb types.ConnectionCallbacks) {
	c.connCallbacks = append(c.connCallbacks, cb)
}

func (c *connection) AddBytesSentCallback(cb func(bytesSent uint64)) {
	c.bytesSendCallbacks = append(c.bytesSendCallbacks, cb)
}

func (c *connection) NextProtocol() string {
	// TODO
	return ""
}

func (c *connection) SetNoDelay(enable bool) {
	rawc := c.rawConnection.(*net.TCPConn)
	rawc.SetNoDelay(enable)
}

func (c *connection) SetReadDisable(disable bool) {
	if disable {
		if !c.readEnabled {
			c.readDisableCount++
			return
		}

		c.readEnabled = false
		c.readEnabledChan <- false
	} else {
		c.readDisableCount--

		if c.readDisableCount > 0 {
			return
		}

		c.readEnabled = true
		c.readEnabledChan <- true
	}
}

func (c *connection) ReadEnabled() bool {
	return c.readEnabled
}

func (c *connection) Ssl() *tls.Conn {
	return nil
}

func (c *connection) SetBufferLimit(limit uint32) {
	c.bufferLimit = limit
}

func (c *connection) BufferLimit() uint32 {
	return c.bufferLimit
}

func (c *connection) SetLocalAddress(localAddress net.Addr, restored bool) {
	// TODO
	c.localAddressRestored = restored
}

func (c *connection) LocalAddressRestored() bool {
	return c.localAddressRestored
}

// BufferSource
func (c *connection) GetWriteBuffer() *[]byte {
	return c.writeBuffer
}

func (c *connection) GetReadBuffer() *bytes.Buffer {
	return c.readBuffer
}

func (c *connection) AboveHighWatermark() bool {
	return c.aboveHighWatermark
}

func (c *connection) FilterManager() types.FilterManager {
	return c.filterManager
}

func (c *connection) startReadLoop() {
	for {
		select {
		case <-c.stopChan:
			return
		case <-c.readEnabledChan:
		default:
			if c.readEnabled {
				err := c.onReadReady()

				if err == io.EOF {
					// remote conn closed
					c.closeConnection(types.RemoteClose)
					return
				}

				if err, ok := err.(net.Error); ok && err.Timeout() {
					continue
				}

				if err != nil {
					c.closeConnection(types.OnReadErrClose)
					return
				}
			} else {
				select {
				case <-c.readEnabledChan:
				}
			}
		}
	}
}

func (c *connection) onReadReady() error {
	bytesRead := 0
	// TODO: enhance: rewrite iobuf, reduce data copy
	buf := make([]byte, 4*1024)

	for {
		t := time.Now()
		// take a break to check channels
		c.rawConnection.SetReadDeadline(t.Add(3 * time.Second))

		rbe := c.readerPool.Take(c.rawConnection)
		nr, err := rbe.Br.Read(buf)
		c.readerPool.Give(rbe)

		if err != nil {
			return err
		}

		if nr == 0 {
			break
		}

		bytesRead += nr
		c.readBuffer.Write(buf[:nr])
		buf = buf[:0]

		if c.readBuffer.Len() >= int(c.bufferLimit) {
			break
		}
	}

	// TODO: update stats

	if bytesRead > 0 {
		c.onRead(bytesRead)
	}

	return nil
}

func (c *connection) onRead(bytesRead int) {
	if !c.readEnabled {
		return
	}

	if bytesRead == 0 {
		return
	}

	c.filterManager.OnRead()
}

func (c *connection) closeConnection(ccEvent types.ConnectionEvent) {
	c.rawConnection.Close()

	// TODO: clean up buffer
	// TODO: clean up stats

	for _, cb := range c.connCallbacks {
		cb.OnEvent(ccEvent)
	}
}

type clientConnection struct {
	connection

	connectOnce sync.Once
}

func NewClientConnection(sourceAddr net.Addr, remoteAddr net.Addr, stopChan chan bool) types.ClientConnection {
	conn := &clientConnection{
		connection: connection{
			localAddr:       sourceAddr,
			remoteAddr:      remoteAddr,
			stopChan:        stopChan,
			readEnabled:     false,
			readEnabledChan: make(chan bool),
			readBuffer:      &bytes.Buffer{},
			readerPool:      buffer.NewReadBufferPool(128, 4*1024),
			writerPool:      buffer.NewWriteBufferPool(128, 4*1024),
		},
	}
	conn.filterManager = newFilterManager(conn)

	return conn
}

func (cc *clientConnection) Connect() {
	cc.connectOnce.Do(func() {
		var localTcpAddr *net.TCPAddr

		if cc.localAddr != nil {
			localTcpAddr, _ = net.ResolveTCPAddr("tcp", cc.localAddr.String())
		}

		remoteTcpAddr, _ := net.ResolveTCPAddr("tcp", cc.remoteAddr.String())

		rawc, err := net.DialTCP("tcp", localTcpAddr, remoteTcpAddr)
		cc.rawConnection = rawc
		var event types.ConnectionEvent

		if err != nil {
			if err == io.EOF {
				// remote conn closed
				event = types.RemoteClose
			} else if err, ok := err.(net.Error); ok && err.Timeout() {
				event = types.ConnectTimeout
			} else {
				event = types.ConnectFailed
			}
		} else {
			event = types.Connected
		}

		for _, cccb := range cc.connCallbacks {
			cccb.OnEvent(event)
		}

		cc.Start(nil)
	})
}
