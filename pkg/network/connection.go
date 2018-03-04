package network

import (
	"net"
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"io"
	"sync"
	"crypto/tls"
	"fmt"
	"bytes"
	"runtime/debug"
	"time"
	"sync/atomic"
)

type connection struct {
	id         uint64
	localAddr  net.Addr
	remoteAddr net.Addr

	nextProtocol         string
	noDelay              bool
	readEnabled          bool
	readEnabledChan      chan bool
	readDisableCount     int
	localAddressRestored bool
	aboveHighWatermark   bool
	bufferLimit          uint32
	rawConnection        net.Conn
	closeWithFlush       bool
	connCallbacks        []types.ConnectionCallbacks
	bytesSendCallbacks   []func(bytesSent uint64)
	filterManager        types.FilterManager

	stopChan           chan bool
	readLoopStopChan   chan bool
	writeLoopStopChan  chan bool
	curWriteBufferData *[]byte
	readBuffer         *buffer.IoBuffer
	readerBufferPool   *buffer.IoBufferPool
	writeBuffer        *bytes.Buffer
	// writebuffer mux
	// TODO: change write buffer to lock-free
	writeMux        sync.Mutex
	writeBufferChan chan bool

	closed    uint32
	startOnce sync.Once
}

func NewServerConnection(rawc net.Conn, id uint64, stopChan chan bool) types.Connection {
	conn := &connection{
		id:                id,
		rawConnection:     rawc,
		localAddr:         rawc.LocalAddr(),
		remoteAddr:        rawc.RemoteAddr(),
		stopChan:          stopChan,
		readLoopStopChan:  make(chan bool, 1),
		readEnabled:       false,
		readEnabledChan:   make(chan bool, 1),
		writeLoopStopChan: make(chan bool, 1),
		writeBuffer:       bytes.NewBuffer(make([]byte, 0, 1024)),
		writeBufferChan:   make(chan bool, 1),
		readerBufferPool:  buffer.NewIoBufferPool(1, 1),
	}

	conn.filterManager = newFilterManager(conn)

	return conn
}

// basic

func (c *connection) Id() uint64 {
	return c.id
}

func (c *connection) Start(lctx context.Context) {
	c.startOnce.Do(func() {
		// TODO: panic recover

		go func() {
			defer func() {
				if p := recover(); p != nil {
					fmt.Printf("panic %v", p)
					fmt.Println()

					debug.PrintStack()
				}
			}()

			c.startReadLoop()
		}()

		go func() {
			defer func() {
				if p := recover(); p != nil {
					fmt.Printf("panic %v", p)
					fmt.Println()

					debug.PrintStack()
				}
			}()

			c.startWriteLoop()
		}()
	})
}

func (c *connection) Write(buf types.IoBuffer) error {
	bufBytes := buf.Bytes()
	c.curWriteBufferData = &bufBytes
	fs := c.filterManager.OnWrite()
	c.curWriteBufferData = nil

	if fs == types.StopIteration {
		return nil
	}

	c.writeMux.Lock()

	if _, err := buf.WriteTo(c.writeBuffer); err != nil {
		c.writeMux.Unlock()

		return err
	}

	c.writeMux.Unlock()

	if len(c.writeBufferChan) == 0 {
		c.writeBufferChan <- true
	}

	return nil
}

func (c *connection) Close(ccType types.ConnectionCloseType) error {
	// disable read first
	c.readEnabled = false
	c.readEnabledChan <- false
	c.rawConnection.(*net.TCPConn).CloseRead()

	if ccType == types.FlushWrite {
		if c.writeBufLen() > 0 {
			c.closeWithFlush = true

			for {
				bytesSent, err := c.doWrite()

				if err != nil {
					if te, ok := err.(net.Error); !(ok && te.Timeout()) {
						break
					}
				}

				if bytesSent == 0 {
					break
				}
			}
		}
	}

	c.closeConnection(types.LocalClose)

	return nil
}

func (c *connection) closeConnection(ccEvent types.ConnectionEvent) {
	if atomic.AddUint32(&c.closed, 1) > 1 {
		return
	}

	if len(c.readLoopStopChan) == 0 {
		c.readLoopStopChan <- true
	}

	if len(c.writeLoopStopChan) == 0 {
		c.writeLoopStopChan <- true
	}

	c.rawConnection.Close()

	// TODO: clean up stats

	for _, cb := range c.connCallbacks {
		cb.OnEvent(ccEvent)
	}
}

func (c *connection) writeBufLen() int {
	c.writeMux.Lock()
	len := c.writeBuffer.Len()
	c.writeMux.Unlock()

	return len
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
	return c.curWriteBufferData
}

func (c *connection) GetReadBuffer() types.IoBuffer {
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
			c.writeLoopStopChan <- true
			return
		case <-c.readLoopStopChan:
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

				if err != nil {
					c.closeConnection(types.OnReadErrClose)
					return
				}
			} else {
				select {
				case <-c.readEnabledChan:
				case <-time.After(100 * time.Millisecond):
				}
			}
		}
	}
}

func (c *connection) onReadReady() error {
	bytesRead := int64(0)
	rbe := c.readerBufferPool.Take(c.rawConnection)

	for {
		nr, err := rbe.Read()
		bytesRead += nr

		if err == buffer.ErrBufFull {
			break
		}

		if err != nil {
			if te, ok := err.(net.Error); ok && te.Timeout() {
				fmt.Println("read timeout")

				continue
			}

			c.readerBufferPool.Give(rbe)
			return err
		}

		if nr == 0 {
			break
		}

		if rbe.Br.Len() >= int(c.bufferLimit) {
			break
		}
	}

	// TODO: update stats

	if bytesRead > 0 {
		c.readBuffer = rbe.Br
		c.onRead(bytesRead)
	}

	c.readerBufferPool.Give(rbe)

	return nil
}

func (c *connection) onRead(bytesRead int64) {
	if !c.readEnabled {
		return
	}

	if bytesRead == 0 {
		return
	}

	c.filterManager.OnRead()
}

func (c *connection) startWriteLoop() {
	for {
		select {
		case <-c.stopChan:
			if len(c.readLoopStopChan) == 0 {
				c.readLoopStopChan <- true
			}

			return
		case <-c.writeLoopStopChan:
			return
		case <-c.writeBufferChan:
			_, err := c.doWrite()

			if err != nil {
				if te, ok := err.(net.Error); ok && te.Timeout() {
					continue
				}

				if err == io.EOF {
					// remote conn closed
					c.closeConnection(types.RemoteClose)
				} else {
					// on non-timeout error
					c.closeConnection(types.OnWriteErrClose)
				}

				return
			}
		}
	}
}

func (c *connection) doWrite() (int64, error) {
	bytesSent, err := c.doWriteIo()

	if bytesSent > 0 {
		for _, cb := range c.bytesSendCallbacks {
			cb(uint64(bytesSent))
		}
	}

	return bytesSent, err
}

func (c *connection) doWriteIo() (int64, error) {
	c.writeMux.Lock()

	if c.writeBuffer.Len() <= 0 {
		c.writeMux.Unlock()
		return 0, nil
	}

	//t := time.Now().Add(200 * time.Millisecond)
	//c.rawConnection.SetWriteDeadline(t)
	bytesSent, err := c.writeBuffer.WriteTo(c.rawConnection)

	c.writeMux.Unlock()

	return bytesSent, err
}

type clientConnection struct {
	connection

	connectOnce sync.Once
}

func NewClientConnection(sourceAddr net.Addr, remoteAddr net.Addr, stopChan chan bool) types.ClientConnection {
	conn := &clientConnection{
		connection: connection{
			localAddr:         sourceAddr,
			remoteAddr:        remoteAddr,
			stopChan:          stopChan,
			readLoopStopChan:  make(chan bool, 1),
			readEnabled:       false,
			readEnabledChan:   make(chan bool, 1),
			writeLoopStopChan: make(chan bool, 1),
			readerBufferPool:  buffer.NewIoBufferPool(1, 1),
			writeBuffer:       bytes.NewBuffer(make([]byte, 0, 1024)),
			writeBufferChan:   make(chan bool, 1),
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

			cc.Start(nil)
		}

		for _, cccb := range cc.connCallbacks {
			cccb.OnEvent(event)
		}
	})
}
