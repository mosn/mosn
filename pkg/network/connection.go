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
	"io"
	"math/rand"
	"net"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/mtls"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/rcrowley/go-metrics"
)

// Network related const
const (
	ConnectionCloseDebugMsg   = "Close connection %d, event %s, type %s, data read %d, data write %d"
	DefaultBufferReadCapacity = 1 << 0
)

var idCounter uint64 = 1

type connection struct {
	id         uint64
	file       *os.File //copy of origin connection fd
	localAddr  net.Addr
	remoteAddr net.Addr

	nextProtocol         string
	noDelay              bool
	readEnabled          bool
	readEnabledChan      chan bool
	readDisableCount     int
	localAddressRestored bool
	bufferLimit          uint32 // todo: support soft buffer limit
	rawConnection        net.Conn
	tlsMng               types.TLSContextManager
	closeWithFlush       bool
	connCallbacks        []types.ConnectionEventListener
	bytesReadCallbacks   []func(bytesRead uint64)
	bytesSendCallbacks   []func(bytesSent uint64)
	transferCallbacks    func() bool
	filterManager        types.FilterManager

	stopChan           chan struct{}
	curWriteBufferData []types.IoBuffer
	readBuffer         types.IoBuffer
	writeBuffers       net.Buffers
	ioBuffers          []types.IoBuffer
	writeBufferChan    chan *[]types.IoBuffer
	transferChan       chan uint64

	// readLoop/writeLoop goroutine fields:
	internalLoopStarted bool
	internalStopChan    chan struct{}
	// eventLoop fields:
	writeSchedChan chan bool // writable if not scheduled yet.

	stats              *types.ConnectionStats
	lastBytesSizeRead  int64
	lastWriteSizeWrite int64

	closed    uint32
	startOnce sync.Once
	eventLoop *eventLoop

	logger log.ErrorLogger
}

// NewServerConnection new server-side connection, rawc is the raw connection from go/net
func NewServerConnection(ctx context.Context, rawc net.Conn, stopChan chan struct{}, logger log.ErrorLogger) types.Connection {
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
		writeBufferChan:  make(chan *[]types.IoBuffer, 32),
		writeSchedChan:   make(chan bool, 1),
		transferChan:     make(chan uint64),
		stats: &types.ConnectionStats{
			ReadTotal:     metrics.NewCounter(),
			ReadBuffered:  metrics.NewGauge(),
			WriteTotal:    metrics.NewCounter(),
			WriteBuffered: metrics.NewGauge(),
		},
		logger: logger,
	}

	// store fd
	if ctx.Value(types.ContextKeyConnectionFd) != nil {
		conn.file = ctx.Value(types.ContextKeyConnectionFd).(*os.File)
	}

	// transfer old mosn connection
	if ctx.Value(types.ContextKeyAcceptChan) != nil {
		if ctx.Value(types.ContextKeyAcceptBuffer) != nil {
			buf := ctx.Value(types.ContextKeyAcceptBuffer).([]byte)
			conn.readBuffer = buffer.GetIoBuffer(len(buf))
			conn.readBuffer.Write(buf)
		}

		ch := ctx.Value(types.ContextKeyAcceptChan).(chan types.Connection)
		ch <- conn
		logger.Infof("NewServerConnection id = %d, buffer = %d", conn.id, conn.readBuffer.Len())
	}

	conn.filterManager = newFilterManager(conn)

	return conn
}

// basic

func (c *connection) ID() uint64 {
	return c.id
}

func (c *connection) Start(lctx context.Context) {
	c.startOnce.Do(func() {
		if UseNetpollMode {
			c.attachEventLoop(lctx)
		} else {
			c.startRWLoop(lctx)
		}
	})
}

func (c *connection) attachEventLoop(lctx context.Context) {
	// Choose one event loop to register, the implement is platform-dependent(epoll for linux and kqueue for bsd)
	c.eventLoop = attach()

	// Register read only, write is supported now because it is more complex than read.
	// We need to write our own code based on syscall.write to deal with the EAGAIN and writable epoll event
	err := c.eventLoop.registerRead(c, &connEventHandler{
		onRead: func() bool {
			if c.readEnabled {
				err := c.doRead()

				if err != nil {
					if te, ok := err.(net.Error); ok && te.Timeout() {
						if c.readBuffer != nil && c.readBuffer.Len() == 0 {
							c.readBuffer.Free()
							c.readBuffer.Alloc(DefaultBufferReadCapacity)
						}
						return true
					}

					if err == io.EOF {
						c.Close(types.NoFlush, types.RemoteClose)
					} else {
						c.Close(types.NoFlush, types.OnReadErrClose)
					}

					c.logger.Errorf("Error on read. Connection = %d, Remote Address = %s, err = %s",
						c.id, c.RemoteAddr().String(), err)

					return false
				}
			} else {
				select {
				case <-c.readEnabledChan:
				case <-time.After(100 * time.Millisecond):
				}
			}
			return true
		},

		onHup: func() bool {
			c.logger.Errorf("ReadHup error. Connection = %d, Remote Address = %s", c.id, c.RemoteAddr().String())
			c.Close(types.NoFlush, types.RemoteClose)
			return false
		},
	})

	if err != nil {
		c.logger.Errorf("conn %d register read failed:%s", c.id, err.Error())
	}
}

func (c *connection) startRWLoop(lctx context.Context) {
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
}

func (c *connection) scheduleWrite() {
	writePool.ScheduleAlways(func() {
		defer func() { <-c.writeSchedChan }()

		for len(c.writeBufferChan) > 0 {
			// at least 1 buffer need to avoid all chan-recv missed by select.default option
			c.appendBuffer(<-c.writeBufferChan)

			//todo: dynamic set loop nums
			//slots := len(c.writeBufferChan)
			//if slots < 10 {
			//	slots = 10
			//}

			//if len(c.writeBufferChan) < 10 {
			//	runtime.Gosched()
			//}

			for i := 0; i < 10; i++ {
				select {
				case buf := <-c.writeBufferChan:
					c.appendBuffer(buf)
				default:
				}
			}

			_, err := c.doWrite()
			if err != nil {
				if err == io.EOF {
					// remote conn closed
					c.Close(types.NoFlush, types.RemoteClose)
				} else {
					// on non-timeout error
					c.Close(types.NoFlush, types.OnWriteErrClose)
				}
				c.logger.Errorf("Error on write. Connection = %d, Remote Address = %s, err = %s",
					c.id, c.RemoteAddr().String(), err)
			}

		}
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
				if c.transferCallbacks != nil && c.transferCallbacks() {
					randTime := time.Duration(rand.Intn(int(TransferTimeout.Nanoseconds())))
					transferTime = time.Now().Add(TransferTimeout).Add(randTime)
					c.logger.Infof("transferTime: Wait %d Second", (TransferTimeout+randTime)/1e9)
				} else {
					// set a long time, not transfer connection, wait mosn exit.
					transferTime = time.Now().Add(10 * TransferTimeout)
					c.logger.Infof("not support transfer connection, Connection = %d, Local Address = %s, Remote Address = %s",
						c.id, c.rawConnection.LocalAddr().String(), c.RemoteAddr().String())
				}
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
					if te, ok := err.(net.Error); ok && te.Timeout() {
						if c.readBuffer != nil && c.readBuffer.Len() == 0 {
							c.readBuffer.Free()
							c.readBuffer.Alloc(DefaultBufferReadCapacity)
						}
						continue
					}
					if err == io.EOF {
						c.Close(types.NoFlush, types.RemoteClose)
					} else {
						c.Close(types.NoFlush, types.OnReadErrClose)
					}

					c.logger.Errorf("Error on read. Connection = %d, Local Address = %s, Remote Address = %s, err = %s",
						c.id, c.rawConnection.LocalAddr().String(), c.RemoteAddr().String(), err)

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
	id, _ := transferRead(c)
	c.transferChan <- id
}

func (c *connection) doRead() (err error) {
	if c.readBuffer == nil {
		c.readBuffer = buffer.GetIoBuffer(DefaultBufferReadCapacity)
	}

	var bytesRead int64

	bytesRead, err = c.readBuffer.ReadOnce(c.rawConnection)

	if err != nil {
		if te, ok := err.(net.Error); ok && te.Timeout() {
			for _, cb := range c.connCallbacks {
				cb.OnEvent(types.OnReadTimeout) // run read timeout callback, for keep alive if configured
			}
			if bytesRead == 0 {
				return err
			}
		} else if err != io.EOF {
			return err
		}
	}

	for _, cb := range c.bytesReadCallbacks {
		cb(uint64(bytesRead))
	}

	c.onRead()
	c.updateReadBufStats(bytesRead, int64(c.readBuffer.Len()))
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
		c.stats.ReadBuffered.Update(bytesBufSize)
		c.lastBytesSizeRead = bytesBufSize
	}
}

func (c *connection) onRead() {
	if !c.readEnabled {
		return
	}

	if c.readBuffer.Len() == 0 {
		return
	}

	c.filterManager.OnRead()
}

func (c *connection) Write(buffers ...types.IoBuffer) error {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Errorf("connection has closed. Connection = %d, Local Address = %+v, Remote Address = %+v",
				c.id, c.LocalAddr(), c.RemoteAddr())
		}
	}()

	fs := c.filterManager.OnWrite(buffers)

	if fs == types.Stop {
		return nil
	}

	if c.internalLoopStarted {
		c.writeBufferChan <- &buffers
	} else {
		// Start schedule if not started
		select {
		case c.writeSchedChan <- true:
			c.scheduleWrite()
		default:
		}

	wait:
		// we use for-loop with select:c.writeSchedChan to avoid chan-send blocking
		// 'c.writeBufferChan <- &buffers' might block if write goroutine costs much time on 'doWriteIo'
		for {
			select {
			case c.writeBufferChan <- &buffers:
				break wait
			case c.writeSchedChan <- true:
				c.scheduleWrite()
			}
		}
	}

	return nil
}

func (c *connection) startWriteLoop() {
	var id uint64
	var err error
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
		case buf := <-c.writeBufferChan:
			c.appendBuffer(buf)

			//todo: dynamic set loop nums
			for i := 0; i < 10; i++ {
				select {
				case buf := <-c.writeBufferChan:
					c.appendBuffer(buf)
				default:
				}
			}
			_, err = c.doWrite()
		}

		/*
				i := 0
				timer := time.NewTimer(3 * time.Millisecond)
				for {
					select {
					case buf := <-c.writeBufferChan:
						c.appendBuffer(buf)
						i++
						if i > 100 {
							_, err = c.doWriteIo()
							goto end
						}
					case <-timer.C:
						_, err = c.doWriteIo()
						goto end
					}
				}
			end:
		*/

		if err != nil {
			if te, ok := err.(net.Error); ok && te.Timeout() {
				continue
			}

			if err == buffer.EOF {
				c.Close(types.NoFlush, types.LocalClose)
			} else if err == io.EOF {
				// remote conn closed
				c.Close(types.NoFlush, types.RemoteClose)
			} else {
				// on non-timeout error
				c.Close(types.NoFlush, types.OnWriteErrClose)
			}

			c.logger.Errorf("Error on write. Connection = %d, Remote Address = %s, err = %s, conn = %p",
				c.id, c.RemoteAddr().String(), err, c)

			return
		}

		runtime.Gosched()
	}

transfer:
	c.logger.Infof("TransferWrite begin")
	for {
		select {
		case <-c.internalStopChan:
			return
		case buf := <-c.writeBufferChan:
			c.appendBuffer(buf)
			transferWrite(c, id)
		}
	}
}

func (c *connection) appendBuffer(iobuffers *[]types.IoBuffer) {
	if iobuffers == nil {
		return
	}
	for _, buf := range *iobuffers {
		if buf == nil {
			continue
		}
		c.ioBuffers = append(c.ioBuffers, buf)
		c.writeBuffers = append(c.writeBuffers, buf.Bytes())
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
	buffers := c.writeBuffers
	if tlsConn, ok := c.rawConnection.(*mtls.TLSConn); ok {
		bytesSent, err = tlsConn.WriteTo(&buffers)
	} else {
		bytesSent, err = buffers.WriteTo(c.rawConnection)
	}
	if err != nil {
		return bytesSent, err
	}
	for _, buf := range c.ioBuffers {
		if buf.EOF() {
			err = buffer.EOF
		}
		if e := buffer.PutIoBuffer(buf); e != nil {
			c.logger.Errorf("PutIoBuffer error: %v", e)
		}
	}
	c.ioBuffers = c.ioBuffers[:0]
	c.writeBuffers = c.writeBuffers[:0]
	return
}

func (c *connection) updateWriteBuffStats(bytesWrite int64, bytesBufSize int64) {
	if c.stats == nil {
		return
	}

	if bytesWrite > 0 {
		c.stats.WriteTotal.Inc(bytesWrite)
	}

	if bytesBufSize != c.lastWriteSizeWrite {
		c.stats.WriteBuffered.Update(bytesBufSize)
		c.lastWriteSizeWrite = bytesBufSize
	}
}

func (c *connection) writeBufLen() (bufLen int) {
	for _, buf := range c.writeBuffers {
		bufLen += len(buf)
	}
	return
}

func (c *connection) Close(ccType types.ConnectionCloseType, eventType types.ConnectionEvent) error {
	if ccType == types.FlushWrite {
		c.Write(buffer.NewIoBufferEOF())
		return nil
	}

	if !atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		return nil
	}

	// connection failed in client mode
	if c.rawConnection == nil || reflect.ValueOf(c.rawConnection).IsNil() {
		return nil
	}

	// shutdown read first
	if rawc, ok := c.rawConnection.(*net.TCPConn); ok {
		c.logger.Debugf("Close TCP Conn, Remote Address is = %s, eventType is = %s", rawc.RemoteAddr(), eventType)
		rawc.CloseRead()
	}

	// wait for io loops exit, ensure single thread operate streams on the connection
	if c.internalLoopStarted {
		// because close function must be called by one io loop thread, notify another loop here
		close(c.internalStopChan)
		close(c.writeBufferChan)
	} else if c.eventLoop != nil {
		// unregister events while connection close
		c.eventLoop.unregister(c.id)
		// close copied fd
		c.file.Close()
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
	c.connCallbacks = append(c.connCallbacks, cb)
}

func (c *connection) AddBytesReadListener(cb func(bytesRead uint64)) {
	c.bytesReadCallbacks = append(c.bytesReadCallbacks, cb)
}

func (c *connection) AddBytesSentListener(cb func(bytesSent uint64)) {
	c.bytesSendCallbacks = append(c.bytesSendCallbacks, cb)
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
	return c.readBuffer
}

func (c *connection) FilterManager() types.FilterManager {
	return c.filterManager
}

func (c *connection) RawConn() net.Conn {
	return c.rawConnection
}

func (c *connection) SetTransferEventListener(listener func() bool) {
	c.transferCallbacks = listener
}

type clientConnection struct {
	connection

	connectOnce sync.Once
}

// NewClientConnection new client-side connection
func NewClientConnection(sourceAddr net.Addr, tlsMng types.TLSContextManager, remoteAddr net.Addr,
	stopChan chan struct{}, logger log.ErrorLogger) types.ClientConnection {
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
			writeBufferChan:  make(chan *[]types.IoBuffer, 32),
			writeSchedChan:   make(chan bool, 1),
			stats: &types.ConnectionStats{
				ReadTotal:     metrics.NewCounter(),
				ReadBuffered:  metrics.NewGauge(),
				WriteTotal:    metrics.NewCounter(),
				WriteBuffered: metrics.NewGauge(),
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
		var event types.ConnectionEvent

		cc.rawConnection, err = net.DialTimeout("tcp", cc.RemoteAddr().String(), time.Second*3)

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

			// ensure ioEnabled and UseNetpollMode
			if ioEnabled && UseNetpollMode {
				// store fd
				if tc, ok := cc.rawConnection.(*net.TCPConn); ok {
					cc.file, err = tc.File()
					if err != nil {
						return
					}
				}
			}

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
