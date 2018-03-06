package network

import (
	"net"
	"context"
	"io"
	"sync"
	"crypto/tls"
	"fmt"
	"runtime/debug"
	"time"
	"sync/atomic"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"bytes"
	"runtime"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"github.com/rcrowley/go-metrics"
)

const (
	ConnectionCloseDebugMsg = "Close connection %d, event %s, type %s, data read %d, data write %d"
)

var idCounter uint64

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
	bytesReadCallbacks   []func(bytesRead uint64)
	bytesSendCallbacks   []func(bytesSent uint64)
	filterManager        types.FilterManager

	stopChan           chan bool
	curWriteBufferData *[]byte
	readBuffer         *buffer.IoBufferPoolEntry
	writeBuffer        *bytes.Buffer
	writeBufferMux     sync.RWMutex
	writeBufferChan    chan bool
	writeLoopStopChan  chan bool
	readerBufferPool   *buffer.IoBufferPool

	stats              *types.ConnectionStats
	lastBytesSizeRead  int64
	lastWriteSizeWrite int64

	closed    uint32
	startOnce sync.Once

	logger log.Logger
}

func NewServerConnection(rawc net.Conn, stopChan chan bool, logger log.Logger) types.Connection {
	id := atomic.AddUint64(&idCounter, 1)

	conn := &connection{
		id:                id,
		rawConnection:     rawc,
		localAddr:         rawc.LocalAddr(),
		remoteAddr:        rawc.RemoteAddr(),
		stopChan:          stopChan,
		bufferLimit:       4 * 1024,
		readEnabled:       true,
		readEnabledChan:   make(chan bool, 1),
		writeBufferChan:   make(chan bool),
		writeBuffer:       bytes.NewBuffer(make([]byte, 0, 4*1024)),
		writeLoopStopChan: make(chan bool, 1),
		readerBufferPool:  buffer.NewIoBufferPool(1, 1024),
		stats: &types.ConnectionStats{
			ReadTotal:    metrics.NewCounter(),
			ReadCurrent:  metrics.NewGauge(),
			WriteTotal:   metrics.NewCounter(),
			WriteCurrent: metrics.NewGauge(),
		},
		logger: log.DefaultLogger,
	}

	conn.filterManager = newFilterManager(conn)

	conn.logger.Debugf("new downstream connection %d accepted", conn.Id())

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

func (c *connection) startReadLoop() {
	for {
		select {
		case <-c.stopChan:
			return
		case <-c.readEnabledChan:
		default:
			if c.readEnabled {
				err := c.doRead()

				if atomic.LoadUint32(&c.closed) > 0 {
					return
				}

				if err == io.EOF {
					// remote conn closed
					c.Close(types.NoFlush, types.RemoteClose)
					return
				}

				if err != nil {
					c.logger.Errorf("Error on read. Connection %d, err %s", c.id, err)
					c.Close(types.NoFlush, types.OnReadErrClose)
					return
				}
			} else {
				select {
				case <-c.readEnabledChan:
				case <-time.After(100 * time.Millisecond):
				}
			}

			runtime.Gosched()
		}
	}
}

func (c *connection) doRead() (err error) {
	if c.readBuffer == nil {
		c.readBuffer = c.readerBufferPool.Take(c.rawConnection)
	}

	var bytesRead int64

	if c.readBuffer.Br.Len() < int(c.bufferLimit) {
		bytesRead, err = c.readBuffer.Read()

		if err != nil {
			if te, ok := err.(net.Error); ok && te.Timeout() {
				return
			}

			c.readerBufferPool.Give(c.readBuffer)
			return err
		}
	}

	c.updateReadBufStats(bytesRead, int64(c.readBuffer.Br.Len()))

	for _, cb := range c.bytesReadCallbacks {
		cb(uint64(bytesRead))
	}

	c.onRead(bytesRead)

	if c.readBuffer.Br.Len() == 0 {
		c.readerBufferPool.Give(c.readBuffer)
	}

	return nil
}

func (c *connection) updateReadBufStats(bytesRead int64, bytesBufSize int64) {
	if c.stats == nil {
		return
	}

	if bytesRead > 0 {
		c.stats.ReadTotal.Inc(bytesRead)
	}

	if bytesBufSize != c.lastBytesSizeRead {
		c.stats.ReadCurrent.Update(bytesBufSize)
		c.lastBytesSizeRead = bytesBufSize
	}
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

func (c *connection) Write(buf types.IoBuffer) error {
	bufBytes := buf.Bytes()
	c.curWriteBufferData = &bufBytes
	fs := c.filterManager.OnWrite()
	c.curWriteBufferData = nil

	if fs == types.StopIteration {
		return nil
	}

	c.writeBufferMux.Lock()
	buf.WriteTo(c.writeBuffer)
	c.writeBufferMux.Unlock()

	c.writeBufferChan <- true

	return nil
}

func (c *connection) startWriteLoop() {
	for {
		select {
		case <-c.stopChan:
			return
		case <-c.writeLoopStopChan:
			return
		case <-c.writeBufferChan:
			if atomic.LoadUint32(&c.closed) > 0 {
				return
			}

			_, err := c.doWrite()

			if err != nil {
				if te, ok := err.(net.Error); ok && te.Timeout() {
					continue
				}

				if err == io.EOF {
					// remote conn closed
					c.Close(types.NoFlush, types.RemoteClose)
				} else {
					c.logger.Errorf("Error on write. Connection %d, err %s", c.id, err)
					// on non-timeout error
					c.Close(types.NoFlush, types.OnWriteErrClose)
				}

				return
			}
		}
	}
}

func (c *connection) doWrite() (int64, error) {
	bytesSent, err := c.doWriteIo()

	c.updateWriteBuffStats(bytesSent, int64(c.writeBuffer.Len()))

	for _, cb := range c.bytesSendCallbacks {
		cb(uint64(bytesSent))
	}

	return bytesSent, err
}

func (c *connection) doWriteIo() (int64, error) {
	var bytesSent int64
	var err error

	for c.writeBufLen() > 0 {
		c.writeBufferMux.Lock()
		m, err := c.writeBuffer.WriteTo(c.rawConnection)
		c.writeBufferMux.Unlock()

		bytesSent += m

		if err != nil {
			if te, ok := err.(net.Error); ok && te.Timeout() {
				continue
			}

			break
		}
	}

	return bytesSent, err
}

func (c *connection) updateWriteBuffStats(bytesWrite int64, bytesBufSize int64) {
	if c.stats == nil {
		return
	}

	if bytesWrite > 0 {
		c.stats.WriteTotal.Inc(bytesWrite)
	}

	if bytesBufSize != c.lastWriteSizeWrite {
		c.stats.WriteCurrent.Update(bytesBufSize)
		c.lastWriteSizeWrite = bytesBufSize
	}
}

func (c *connection) writeBufLen() int {
	c.writeBufferMux.RLock()
	wbLen := c.writeBuffer.Len()
	c.writeBufferMux.RUnlock()

	return wbLen
}

func (c *connection) Close(ccType types.ConnectionCloseType, eventType types.ConnectionEvent) error {
	if !atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		return nil
	}

	// shutdown read first
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

	c.writeLoopStopChan <- true
	c.rawConnection.Close()

	c.logger.Debugf(ConnectionCloseDebugMsg, c.id, eventType,
		ccType, c.stats.ReadTotal.Count(), c.stats.WriteTotal.Count())

	c.updateReadBufStats(0, 0)
	c.updateWriteBuffStats(0, 0)

	for _, cb := range c.connCallbacks {
		cb.OnEvent(eventType)
	}

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

func (c *connection) AddBytesReadCallback(cb func(bytesRead uint64)) {

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
		if c.readDisableCount > 0 {
			c.readDisableCount--
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
	if limit > 0 {
		c.bufferLimit = limit
	}
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
	if c.readBuffer != nil {
		return c.readBuffer.Br
	} else {
		return nil
	}
}

func (c *connection) AboveHighWatermark() bool {
	return c.aboveHighWatermark
}

func (c *connection) FilterManager() types.FilterManager {
	return c.filterManager
}

type clientConnection struct {
	connection

	connectOnce sync.Once
}

func NewClientConnection(sourceAddr net.Addr, remoteAddr net.Addr, stopChan chan bool) types.ClientConnection {
	id := atomic.AddUint64(&idCounter, 1)

	conn := &clientConnection{
		connection: connection{
			id:                id,
			localAddr:         sourceAddr,
			remoteAddr:        remoteAddr,
			stopChan:          stopChan,
			bufferLimit:       4 * 1024,
			readEnabled:       true,
			readEnabledChan:   make(chan bool, 1),
			writeBufferChan:   make(chan bool),
			writeBuffer:       bytes.NewBuffer(make([]byte, 0, 4*1024)),
			writeLoopStopChan: make(chan bool, 1),
			readerBufferPool:  buffer.NewIoBufferPool(1, 1024),
			stats: &types.ConnectionStats{
				ReadTotal:    metrics.NewCounter(),
				ReadCurrent:  metrics.NewGauge(),
				WriteTotal:   metrics.NewCounter(),
				WriteCurrent: metrics.NewGauge(),
			},
			logger: log.DefaultLogger,
		},
	}
	conn.filterManager = newFilterManager(conn)

	conn.logger.Debugf("new upstream connection %d created", conn.Id())

	return conn
}

func (cc *clientConnection) Connect() (err error) {
	cc.connectOnce.Do(func() {
		var localTcpAddr *net.TCPAddr

		if cc.localAddr != nil {
			localTcpAddr, err = net.ResolveTCPAddr("tcp", cc.localAddr.String())
		}

		var remoteTcpAddr *net.TCPAddr
		remoteTcpAddr, err = net.ResolveTCPAddr("tcp", cc.remoteAddr.String())

		var rawc *net.TCPConn
		rawc, err = net.DialTCP("tcp", localTcpAddr, remoteTcpAddr)
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

	return
}
