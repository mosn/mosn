/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package network

import (
	"context"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network/buffer"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/rcrowley/go-metrics"
	"io"
	"math/rand"
	"net"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Network related const
const (
	ConnectionCloseDebugMsg = "Close connection %d, event %s, type %s, data read %d, data write %d"
	DefaultBufferCapacity   = 1 << 12
)

var idCounter uint64 = 1
var readerBufferPool = buffer.NewIoBufferPool(DefaultBufferCapacity)
var writeBufferPool = buffer.NewIoBufferPool(DefaultBufferCapacity)

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
	tlsMng               types.TLSContextManager
	closeWithFlush       bool
	connCallbacks        []types.ConnectionEventListener
	bytesReadCallbacks   []func(bytesRead uint64)
	bytesSendCallbacks   []func(bytesSent uint64)
	filterManager        types.FilterManager

	stopChan            chan struct{}
	curWriteBufferData  []types.IoBuffer
	readBuffer          *buffer.IoBufferPoolEntry
	writeBuffer         *buffer.IoBufferPoolEntry
	writeBufferMux      sync.RWMutex
	writeBufferChan     chan bool
	internalLoopStarted bool
	internalStopChan    chan struct{}
	transferChan        chan uint64
	readerBufferPool    *buffer.IoBufferPool
	writeBufferPool     *buffer.IoBufferPool

	stats              *types.ConnectionStats
	lastBytesSizeRead  int64
	lastWriteSizeWrite int64

	closed    uint32
	startOnce sync.Once

	logger log.Logger
}

// NewServerConnection
// rawc is the raw connection from go/net
func NewServerConnection(ctx context.Context, rawc net.Conn, stopChan chan struct{}, logger log.Logger) types.Connection {
	id := atomic.AddUint64(&idCounter, 1)

	conn := &connection{
		id:               id,
		rawConnection:    rawc,
		localAddr:        rawc.LocalAddr(),
		remoteAddr:       rawc.RemoteAddr(),
		stopChan:         stopChan,
		readEnabled:      true,
		readEnabledChan:  make(chan bool, 1),
		internalStopChan: make(chan struct{}),
		writeBufferChan:  make(chan bool, 1),
		transferChan:     make(chan uint64),
		readerBufferPool: readerBufferPool,
		writeBufferPool:  writeBufferPool,
		stats: &types.ConnectionStats{
			ReadTotal:    metrics.NewCounter(),
			ReadCurrent:  metrics.NewGauge(),
			WriteTotal:   metrics.NewCounter(),
			WriteCurrent: metrics.NewGauge(),
		},
		logger: logger,
	}

	// transfer old mosn connection
	if ctx.Value(types.ContextKeyAcceptChan) != nil {
		if ctx.Value(types.ContextKeyAcceptBuffer) != nil {
			buf := ctx.Value(types.ContextKeyAcceptBuffer).([]byte)
			conn.readBuffer = conn.readerBufferPool.Take(conn.rawConnection)
			conn.readBuffer.Br.Write(buf)
		}

		ch := ctx.Value(types.ContextKeyAcceptChan).(chan types.Connection)
		ch <- conn
		logger.Infof("NewServerConnection id = %d, buffer = %d", conn.id, conn.readBuffer.Br.Len())
	}
	//conn.writeBuffer = buffer.NewWatermarkBuffer(DefaultWriteBufferCapacity, conn)

	conn.filterManager = newFilterManager(conn)

	return conn
}

// watermark listener
func (c *connection) OnHighWatermark() {
	c.aboveHighWatermark = true
}

func (c *connection) OnLowWatermark() {
	c.aboveHighWatermark = false
}

// basic

func (c *connection) ID() uint64 {
	return c.id
}

func (c *connection) Start(lctx context.Context) {
	c.startOnce.Do(func() {
		c.internalLoopStarted = true

		go func() {
			defer func() {
				if p := recover(); p != nil {
					c.logger.Errorf("panic %v", p)

					debug.PrintStack()

					c.startReadLoop()
				}
			}()

			c.startReadLoop()
		}()

		go func() {
			defer func() {
				if p := recover(); p != nil {
					c.logger.Errorf("panic %v", p)

					debug.PrintStack()

					c.startWriteLoop()
				}
			}()

			c.startWriteLoop()
		}()
	})
}

func (c *connection) startReadLoop() {
	var transferTime time.Time
	for {
		// exit loop asap. one receive & one default block will be optimized by go compiler
		select {
		case <-c.internalStopChan:
			return
		default:
		}

		select {
		case <-c.stopChan:
			if transferTime.IsZero() {
				randTime := time.Duration(rand.Intn(int(TransferTimeout.Nanoseconds())))
				transferTime = time.Now().Add(randTime)
				c.logger.Infof("transferTime: Wait %d Second", randTime/1e9)
			} else {
				if transferTime.Before(time.Now()) {
					goto transfer
				}
			}
		default:
		}

		select {
		case <-c.internalStopChan:
			return
		case <-c.readEnabledChan:
		default:
			if c.readEnabled {
				err := c.doRead()

				if err != nil {

					if err == io.EOF {
						c.Close(types.NoFlush, types.RemoteClose)
					} else {
						c.Close(types.NoFlush, types.OnReadErrClose)
					}

					c.logger.Errorf("Error on read. Connection = %d, Remote Address = %s, err = %s",
						c.id, c.RemoteAddr().String(), err)

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

transfer:
	c.transferChan <- transferNotify
	id, _ := transferRead(c.rawConnection, c.readBuffer.Br, c.logger)
	c.transferChan <- id
}

func (c *connection) doRead() (err error) {
	if c.readBuffer == nil {
		c.readBuffer = c.readerBufferPool.Take(c.rawConnection)
	}

	var bytesRead int64

	bytesRead, err = c.readBuffer.Read()

	if err != nil {
		if te, ok := err.(net.Error); ok && te.Timeout() {
			return
		}

		c.readerBufferPool.Give(c.readBuffer)
		c.readBuffer = nil
		return err
	}

	c.updateReadBufStats(bytesRead, int64(c.readBuffer.Br.Len()))

	for _, cb := range c.bytesReadCallbacks {
		cb(uint64(bytesRead))
	}

	c.onRead(bytesRead)

	return
}

func (c *connection) updateReadBufStats(bytesRead int64, bytesBufSize int64) {
	if c.stats == nil {
		return
	}

	if bytesRead > 0 {
		c.stats.ReadTotal.Inc(bytesRead)
	}

	if bytesBufSize != c.lastBytesSizeRead {
		// todo: fix: when read blocks, ReadCurrent is out-of-date
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

func (c *connection) Write(buffers ...types.IoBuffer) error {
	c.curWriteBufferData = buffers
	fs := c.filterManager.OnWrite()
	c.curWriteBufferData = nil

	if fs == types.StopIteration {
		return nil
	}

	c.writeBufferMux.Lock()

	if c.writeBuffer == nil {
		c.writeBuffer = c.writeBufferPool.Take(c.rawConnection)
	}

	for _, buf := range buffers {
		if buf != nil {
			buf.WriteTo(c.writeBuffer.Br)
		}
	}

	if len(c.writeBufferChan) == 0 {
		c.writeBufferChan <- true
	}

	c.writeBufferMux.Unlock()

	return nil
}

func (c *connection) startWriteLoop() {
	var id uint64
	for {
		// exit loop asap. one receive & one default block will be optimized by go compiler
		select {
		case <-c.internalStopChan:
			return
		default:
		}

		select {
		case <-c.internalStopChan:
			return
		case <-c.transferChan:
			id = <-c.transferChan
			if id != transferErr {
				goto transfer
			}
		case <-c.writeBufferChan:
			_, err := c.doWrite()
			if err != nil {
				if te, ok := err.(net.Error); ok && te.Timeout() {
					continue
				}

				if err == io.EOF {
					// remote conn closed
					c.Close(types.NoFlush, types.RemoteClose)
				} else {
					// on non-timeout error
					c.Close(types.NoFlush, types.OnWriteErrClose)
				}

				c.logger.Errorf("Error on write. Connection = %d, Remote Address = %s, err = %s",
					c.id, c.RemoteAddr().String(), err)

				return
			}
		}
	}

transfer:
	c.logger.Infof("TransferWrite begin")
	for {
		select {
		case <-c.internalStopChan:
			return
		case <-c.writeBufferChan:
			if c.writeBufLen() == 0 {
				continue
			}
			c.writeBufferMux.Lock()
			transferWrite(c.writeBuffer.Br, id, c.logger)
			c.writeBufferMux.Unlock()
		}
	}
}

func (c *connection) doWrite() (int64, error) {
	bytesSent, err := c.doWriteIo()

	c.updateWriteBuffStats(bytesSent, int64(c.writeBufLen()))

	for _, cb := range c.bytesSendCallbacks {
		cb(uint64(bytesSent))
	}

	return bytesSent, err
}

func (c *connection) doWriteIo() (bytesSent int64, err error) {
	var m int64

	for c.writeBufLen() > 0 {
		c.writeBufferMux.Lock()
		m, err = c.writeBuffer.Write()
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
	defer c.writeBufferMux.RUnlock()

	if c.writeBuffer == nil {
		return 0
	}

	return c.writeBuffer.Br.Len()
}

func (c *connection) Close(ccType types.ConnectionCloseType, eventType types.ConnectionEvent) error {
	if !atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		return nil
	}

	// connection failed in client mode
	if c.rawConnection == nil {
		return nil
	}

	// shutdown read first
	if rawc, ok := c.rawConnection.(*net.TCPConn); ok {
		c.logger.Debugf("Close TCP Conn, Remote Address is = %s, eventType is = %s", rawc.RemoteAddr(), eventType)
		rawc.CloseRead()
	}

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

	// wait for io loops exit, ensure single thread operate streams on the connection
	if c.internalLoopStarted {
		// because close function must be called by one io loop thread, notify another loop here
		close(c.internalStopChan)
	}

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

func (c *connection) SetRemoteAddr(address net.Addr) {
	c.remoteAddr = address
}

func (c *connection) AddConnectionEventListener(cb types.ConnectionEventListener) {
	exist := false

	for _, ccb := range c.connCallbacks {
		if &ccb == &cb {
			exist = true
			c.logger.Debugf("AddConnectionEventListener Failed, %+v Already Exist", cb)
		}
	}

	if !exist {
		c.logger.Debugf("AddConnectionEventListener Success, cb = %+v", cb)
		c.connCallbacks = append(c.connCallbacks, cb)
	}
}

func (c *connection) AddBytesReadListener(cb func(bytesRead uint64)) {
	exist := false

	for _, brcb := range c.bytesReadCallbacks {
		if &brcb == &cb {
			exist = true
		}
	}

	if !exist {
		c.bytesReadCallbacks = append(c.bytesReadCallbacks, cb)
	}
}

func (c *connection) AddBytesSentListener(cb func(bytesSent uint64)) {
	exist := false

	for _, bscb := range c.bytesSendCallbacks {
		if &bscb == &cb {
			exist = true
		}
	}

	if !exist {
		c.bytesSendCallbacks = append(c.bytesSendCallbacks, cb)
	}
}

func (c *connection) NextProtocol() string {
	// TODO
	return ""
}

func (c *connection) SetNoDelay(enable bool) {
	if c.rawConnection != nil {

		if rawc, ok := c.rawConnection.(*net.TCPConn); ok {
			rawc.SetNoDelay(enable)
		}
	}
}

func (c *connection) SetReadDisable(disable bool) {
	if disable {
		if !c.readEnabled {
			c.readDisableCount++
			return
		}

		c.readEnabled = false
	} else {
		if c.readDisableCount > 0 {
			c.readDisableCount--
			return
		}

		c.readEnabled = true
		// only on read disable status, we need to trigger chan to wake read loop up
		c.readEnabledChan <- true
	}
}

func (c *connection) ReadEnabled() bool {
	return c.readEnabled
}

func (c *connection) TLS() net.Conn {
	return nil
}

func (c *connection) SetBufferLimit(limit uint32) {
	if limit > 0 {
		c.bufferLimit = limit

		//c.writeBuffer.(*buffer.watermarkBuffer).SetWaterMark(limit)
	}
}

func (c *connection) BufferLimit() uint32 {
	return c.bufferLimit
}

func (c *connection) SetLocalAddress(localAddress net.Addr, restored bool) {
	// TODO
	c.localAddressRestored = restored
}

func (c *connection) SetStats(stats *types.ConnectionStats) {
	c.stats = stats
}

func (c *connection) LocalAddressRestored() bool {
	return c.localAddressRestored
}

// BufferSource
func (c *connection) GetWriteBuffer() []types.IoBuffer {
	return c.curWriteBufferData
}

func (c *connection) GetReadBuffer() types.IoBuffer {
	if c.readBuffer != nil {
		return c.readBuffer.Br
	}

	return nil
}

func (c *connection) FilterManager() types.FilterManager {
	return c.filterManager
}

func (c *connection) RawConn() net.Conn {
	return c.rawConnection
}

type clientConnection struct {
	connection

	connectOnce sync.Once
}

// NewClientConnection
func NewClientConnection(sourceAddr net.Addr, tlsMng types.TLSContextManager, remoteAddr net.Addr,
	stopChan chan struct{}, logger log.Logger) types.ClientConnection {
	id := atomic.AddUint64(&idCounter, 1)

	conn := &clientConnection{
		connection: connection{
			id:               id,
			localAddr:        sourceAddr,
			remoteAddr:       remoteAddr,
			stopChan:         stopChan,
			readEnabled:      true,
			readEnabledChan:  make(chan bool, 1),
			internalStopChan: make(chan struct{}),
			writeBufferChan:  make(chan bool, 1),
			readerBufferPool: readerBufferPool,
			writeBufferPool:  writeBufferPool,
			stats: &types.ConnectionStats{
				ReadTotal:    metrics.NewCounter(),
				ReadCurrent:  metrics.NewGauge(),
				WriteTotal:   metrics.NewCounter(),
				WriteCurrent: metrics.NewGauge(),
			},
			logger: logger,
			tlsMng: tlsMng,
		},
	}

	conn.filterManager = newFilterManager(conn)

	return conn
}

func (cc *clientConnection) Connect(ioEnabled bool) (err error) {
	cc.connectOnce.Do(func() {
		var localTCPAddr *net.TCPAddr

		if cc.localAddr != nil {
			localTCPAddr, err = net.ResolveTCPAddr("tcp", cc.localAddr.String())
		}

		var remoteTCPAddr *net.TCPAddr
		remoteTCPAddr, err = net.ResolveTCPAddr("tcp", cc.remoteAddr.String())

		cc.rawConnection, err = net.DialTCP("tcp", localTCPAddr, remoteTCPAddr)
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

			if cc.tlsMng != nil && cc.tlsMng.Enabled() {
				cc.rawConnection = cc.tlsMng.Conn(cc.rawConnection)
			}

			if ioEnabled {
				cc.Start(nil)
			}
		}

		cc.connection.logger.Debugf("connect raw tcp, remote address = %s ,event = %+v, error = %+v", cc.remoteAddr.String(), event, err)
		for _, cccb := range cc.connCallbacks {
			cccb.OnEvent(event)
		}
	})

	return
}
