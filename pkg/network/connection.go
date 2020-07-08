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
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"reflect"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rcrowley/go-metrics"
	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/utils"
)

// Network related const
const (
	DefaultBufferReadCapacity = 1 << 7

	NetBufferDefaultSize     = 0
	NetBufferDefaultCapacity = 1 << 4

	DefaultConnectTimeout = 3 * time.Second
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
	connCallbacks        []api.ConnectionEventListener
	bytesReadCallbacks   []func(bytesRead uint64)
	bytesSendCallbacks   []func(bytesSent uint64)
	transferCallbacks    func() bool
	filterManager        api.FilterManager
	idleEventListener    api.ConnectionEventListener

	stopChan           chan struct{}
	curWriteBufferData []buffer.IoBuffer
	readBuffer         buffer.IoBuffer
	writeBuffers       net.Buffers
	ioBuffers          []buffer.IoBuffer
	writeBufferChan    chan *[]buffer.IoBuffer
	transferChan       chan uint64

	// readLoop/writeLoop goroutine fields:
	internalLoopStarted bool
	internalStopChan    chan struct{}
	// eventLoop fields:
	writeSchedChan chan bool // writable if not scheduled yet.

	stats              *types.ConnectionStats
	readCollector      metrics.Counter
	writeCollector     metrics.Counter
	lastBytesSizeRead  int64
	lastWriteSizeWrite int64

	closed    uint32
	connected uint32
	startOnce sync.Once
	eventLoop *eventLoop

	tryMutex     *utils.Mutex
	needTransfer bool
	useWriteLoop bool
}

// NewServerConnection new server-side connection, rawc is the raw connection from go/net
func NewServerConnection(ctx context.Context, rawc net.Conn, stopChan chan struct{}) api.Connection {
	id := atomic.AddUint64(&idCounter, 1)

	conn := &connection{
		id:               id,
		rawConnection:    rawc,
		localAddr:        rawc.LocalAddr(),
		remoteAddr:       rawc.RemoteAddr(),
		stopChan:         stopChan,
		readEnabled:      true,
		connected:        1,
		readEnabledChan:  make(chan bool, 1),
		internalStopChan: make(chan struct{}),
		writeBufferChan:  make(chan *[]buffer.IoBuffer, 8),
		writeSchedChan:   make(chan bool, 1),
		transferChan:     make(chan uint64),
		stats: &types.ConnectionStats{
			ReadTotal:     metrics.NewCounter(),
			ReadBuffered:  metrics.NewGauge(),
			WriteTotal:    metrics.NewCounter(),
			WriteBuffered: metrics.NewGauge(),
		},
		readCollector:  metrics.NilCounter{},
		writeCollector: metrics.NilCounter{},
		tryMutex:       utils.NewMutex(),
	}

	// store fd
	if val := mosnctx.Get(ctx, types.ContextKeyConnectionFd); val != nil {
		conn.file = val.(*os.File)
	}

	// transfer old mosn connection
	if val := mosnctx.Get(ctx, types.ContextKeyAcceptChan); val != nil {
		if val := mosnctx.Get(ctx, types.ContextKeyAcceptBuffer); val != nil {
			buf := val.([]byte)
			conn.readBuffer = buffer.GetIoBuffer(len(buf))
			conn.readBuffer.Write(buf)
		}

		ch := val.(chan api.Connection)
		ch <- conn
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[network] [new server connection] NewServerConnection id = %d, buffer = %d", conn.id, conn.readBuffer.Len())
		}
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

func (c *connection) SetIdleTimeout(d time.Duration) {
	c.newIdleChecker(d)
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
						c.Close(api.NoFlush, api.RemoteClose)
					} else {
						c.Close(api.NoFlush, api.OnReadErrClose)
					}

					log.DefaultLogger.Errorf("[network] [event loop] [onRead] Error on read. Connection = %d, Remote Address = %s, err = %s",
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
			log.DefaultLogger.Errorf("[network] [event loop] [onHup] ReadHup error. Connection = %d, Remote Address = %s", c.id, c.RemoteAddr().String())
			c.Close(api.NoFlush, api.RemoteClose)
			return false
		},
	})

	if err != nil {
		log.DefaultLogger.Errorf("[network] [event loop] [register] conn %d register read failed:%s", c.id, err.Error())
	}
}

func (c *connection) checkUseWriteLoop() bool {
	tcpAddr, ok := c.remoteAddr.(*net.TCPAddr)
	if !ok {
		return false
	}
	if tcpAddr.IP.IsLoopback() {
		log.DefaultLogger.Debugf("[network] [check use writeloop] Connection = %d, Local Address = %+v, Remote Address = %+v",
			c.id, c.rawConnection.LocalAddr(), c.RemoteAddr())
		return true
	}
	return false
}

func (c *connection) startRWLoop(lctx context.Context) {
	c.internalLoopStarted = true

	utils.GoWithRecover(func() {
		c.startReadLoop()
	}, func(r interface{}) {
		c.Close(api.NoFlush, api.LocalClose)
	})

	if c.checkUseWriteLoop() {
		c.useWriteLoop = true
		utils.GoWithRecover(func() {
			c.startWriteLoop()
		}, func(r interface{}) {
			c.Close(api.NoFlush, api.LocalClose)
		})
	}
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
				case buf, ok := <-c.writeBufferChan:
					if !ok {
						return
					}
					c.appendBuffer(buf)
				default:
				}
			}

			_, err := c.doWrite()
			if err != nil {
				if err == io.EOF {
					// remote conn closed
					c.Close(api.NoFlush, api.RemoteClose)
				} else {
					// on non-timeout error
					c.Close(api.NoFlush, api.OnWriteErrClose)
				}
				log.DefaultLogger.Errorf("[network] [schedule write] Error on write. Connection = %d, Remote Address = %s, err = %s",
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
					log.DefaultLogger.Infof("[network] [read loop] transferTime: Wait %d Second", (TransferTimeout+randTime)/1e9)
				} else {
					// set a long time, not transfer connection, wait mosn exit.
					transferTime = time.Now().Add(10 * TransferTimeout)
					log.DefaultLogger.Infof("[network] [read loop] not support transfer connection, Connection = %d, Local Address = %+v, Remote Address = %+v",
						c.id, c.rawConnection.LocalAddr(), c.RemoteAddr())
				}
			} else {
				if transferTime.Before(time.Now()) {
					c.transfer()
					return
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
						if c.readBuffer != nil && c.readBuffer.Len() == 0 && c.readBuffer.Cap() > DefaultBufferReadCapacity {
							c.readBuffer.Free()
							c.readBuffer.Alloc(DefaultBufferReadCapacity)
						}
						continue
					}

					// normal close or health check, modify log level
					if c.lastBytesSizeRead == 0 || err == io.EOF {
						if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
							log.DefaultLogger.Debugf("[network] [read loop] Error on read. Connection = %d, Local Address = %+v, Remote Address = %+v, err = %v",
								c.id, c.rawConnection.LocalAddr(), c.RemoteAddr(), err)
						}
					} else {
						log.DefaultLogger.Errorf("[network] [read loop] Error on read. Connection = %d, Local Address = %+v, Remote Address = %+v, err = %v",
							c.id, c.rawConnection.LocalAddr(), c.RemoteAddr(), err)
					}

					if err == io.EOF {
						c.Close(api.NoFlush, api.RemoteClose)
					} else {
						c.Close(api.NoFlush, api.OnReadErrClose)
					}

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

func (c *connection) transfer() {
	c.notifyTransfer()
	id, _ := transferRead(c)
	c.transferWrite(id)
}

func (c *connection) notifyTransfer() {
	if c.useWriteLoop {
		c.transferChan <- transferNotify
	} else {
		locked := c.tryMutex.TryLock(types.DefaultConnTryTimeout)
		if locked {
			c.needTransfer = true
			c.tryMutex.Unlock()
		}
	}
}

func (c *connection) transferWrite(id uint64) {
	log.DefaultLogger.Infof("[network] TransferWrite begin")
	for {
		select {
		case <-c.internalStopChan:
			return
		case buf, ok := <-c.writeBufferChan:
			if !ok {
				return
			}
			c.appendBuffer(buf)
			transferWrite(c, id)
		}
	}
}

func (c *connection) doRead() (err error) {
	if c.readBuffer == nil {
		c.readBuffer = buffer.GetIoBuffer(DefaultBufferReadCapacity)
	}

	var bytesRead int64

	bytesRead, err = c.readBuffer.ReadOnce(c.rawConnection)

	if err != nil {
		if atomic.LoadUint32(&c.closed) == 1 {
			return err
		}
		if te, ok := err.(net.Error); ok && te.Timeout() {
			for _, cb := range c.connCallbacks {
				cb.OnEvent(api.OnReadTimeout) // run read timeout callback, for keep alive if configured
			}
			if bytesRead == 0 {
				return err
			}
		} else if err != io.EOF {
			return err
		}
	}

	//todo: ReadOnce maybe always return (0, nil) and causes dead loop (hack)
	if bytesRead == 0 && err == nil {
		err = io.EOF
		log.DefaultLogger.Errorf("[network] ReadOnce maybe always return (0, nil) and causes dead loop, Connection = %d, Local Address = %+v, Remote Address = %+v",
			c.id, c.rawConnection.LocalAddr(), c.RemoteAddr())
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
		c.readCollector.Inc(bytesRead)
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

func (c *connection) Write(buffers ...buffer.IoBuffer) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[network] [write] connection has closed. Connection = %d, Local Address = %+v, Remote Address = %+v, err = %+v",
				c.id, c.LocalAddr(), c.RemoteAddr(), r)
			err = types.ErrConnectionHasClosed
		}
	}()

	fs := c.filterManager.OnWrite(buffers)

	if fs == api.Stop {
		return nil
	}

	if !UseNetpollMode {
		if c.useWriteLoop {
			c.writeBufferChan <- &buffers
		} else {
			err = c.writeDirectly(&buffers)
		}
	} else {
		if atomic.LoadUint32(&c.connected) == 1 {
			return fmt.Errorf("can note schedule write on the un-connected connection %d", c.id)
		}

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

	return
}

func (c *connection) writeDirectly(buf *[]buffer.IoBuffer) (err error) {
	select {
	case <-c.internalStopChan:
		return types.ErrConnectionHasClosed
	default:
	}

	locked := c.tryMutex.TryLock(types.DefaultConnTryTimeout)

	if locked {
		defer c.tryMutex.Unlock()
		if c.needTransfer {
			c.writeBufferChan <- buf
			return
		}

		c.appendBuffer(buf)

		c.rawConnection.SetWriteDeadline(time.Now().Add(types.DefaultConnWriteTimeout))
		_, err = c.doWrite()
	} else {
		// trylock timeouted
		err = types.ErrWriteTryLockTimeout
	}

	if err != nil {
		log.DefaultLogger.Errorf("[network] [write directly] Error on write. Connection = %d, Remote Address = %s, err = %s, conn = %p",
			c.id, c.RemoteAddr().String(), err, c)

		if te, ok := err.(net.Error); ok && te.Timeout() {
			c.Close(api.NoFlush, api.OnWriteTimeout)
		}

		if err == buffer.EOF {
			c.Close(api.NoFlush, api.LocalClose)
		}

		//other write errs not close connection, beacause readbuffer may have unread data, wait for readloop close connection,

		return
	}

	return nil
}

func (c *connection) startWriteLoop() {
	var needTransfer bool
	defer func() {
		if !needTransfer {
			close(c.writeBufferChan)
		}
	}()

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
			needTransfer = true
			return
		case buf, ok := <-c.writeBufferChan:
			if !ok {
				return
			}
			c.appendBuffer(buf)

			//todo: dynamic set loop nums
			for i := 0; i < 10; i++ {
				select {
				case buf, ok := <-c.writeBufferChan:
					if !ok {
						return
					}
					c.appendBuffer(buf)
				default:
					break
				}
			}

			c.rawConnection.SetWriteDeadline(time.Now().Add(types.DefaultConnWriteTimeout))
			_, err = c.doWrite()
		}

		if err != nil {
			log.DefaultLogger.Errorf("[network] [write loop] Error on write. Connection = %d, Remote Address = %s, err = %s, conn = %p",
				c.id, c.RemoteAddr().String(), err, c)

			if te, ok := err.(net.Error); ok && te.Timeout() {
				c.Close(api.NoFlush, api.OnWriteTimeout)
			}

			if err == buffer.EOF {
				c.Close(api.NoFlush, api.LocalClose)
			}

			//other write errs not close connection, beacause readbuffer may have unread data, wait for readloop close connection,

			return
		}
	}
}

func (c *connection) appendBuffer(iobuffers *[]buffer.IoBuffer) {
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
	if err != nil && atomic.LoadUint32(&c.closed) == 1 {
		return 0, nil
	}

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
		//todo: writev(runtime) has memroy leak.
		bytesSent, err = buffers.WriteTo(c.rawConnection)
	}
	if err != nil {
		return bytesSent, err
	}
	for i, buf := range c.ioBuffers {
		c.ioBuffers[i] = nil
		c.writeBuffers[i] = nil
		if buf.EOF() {
			err = buffer.EOF
		}
		if e := buffer.PutIoBuffer(buf); e != nil {
			log.DefaultLogger.Errorf("[network] [do write] PutIoBuffer error: %v", e)
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
		c.writeCollector.Inc(bytesWrite)
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

func (c *connection) Close(ccType api.ConnectionCloseType, eventType api.ConnectionEvent) error {
	defer func() {
		if p := recover(); p != nil {
			log.DefaultLogger.Errorf("[network] [close connection] panic %v\n%s", p, string(debug.Stack()))
		}
	}()

	if ccType == api.FlushWrite {
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
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[network] [close connection] Close TCP Conn, Remote Address is = %s, eventType is = %s", rawc.RemoteAddr(), eventType)
		}
		rawc.CloseRead()
	}

	// wait for io loops exit, ensure single thread operate streams on the connection
	// because close function must be called by one io loop thread, notify another loop here
	close(c.internalStopChan)
	if c.eventLoop != nil {
		// unregister events while connection close
		c.eventLoop.unregister(c.id)
		// close copied fd
		c.file.Close()
	}

	c.rawConnection.Close()

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[network] [close connection] Close connection %d, event %s, type %s", c.id, eventType, ccType)
	}

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

func (c *connection) AddConnectionEventListener(cb api.ConnectionEventListener) {
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

func (c *connection) SetCollector(read, write metrics.Counter) {
	c.readCollector = read
	c.writeCollector = write
}

func (c *connection) LocalAddressRestored() bool {
	return c.localAddressRestored
}

// BufferSource
func (c *connection) GetWriteBuffer() []buffer.IoBuffer {
	return c.curWriteBufferData
}

func (c *connection) GetReadBuffer() buffer.IoBuffer {
	return c.readBuffer
}

func (c *connection) FilterManager() api.FilterManager {
	return c.filterManager
}

func (c *connection) RawConn() net.Conn {
	return c.rawConnection
}

func (c *connection) SetTransferEventListener(listener func() bool) {
	c.transferCallbacks = listener
}

func (c *connection) State() api.ConnState {
	if atomic.LoadUint32(&c.closed) == 1 {
		return api.ConnClosed
	}
	if atomic.LoadUint32(&c.connected) == 1 {
		return api.ConnActive
	}
	return api.ConnInit
}

type clientConnection struct {
	connection

	connectTimeout time.Duration

	connectOnce sync.Once
}

// NewClientConnection new client-side connection
func NewClientConnection(sourceAddr net.Addr, connectTimeout time.Duration, tlsMng types.TLSContextManager, remoteAddr net.Addr, stopChan chan struct{}) types.ClientConnection {
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
			writeBufferChan:  make(chan *[]buffer.IoBuffer, 8),
			writeSchedChan:   make(chan bool, 1),
			stats: &types.ConnectionStats{
				ReadTotal:     metrics.NewCounter(),
				ReadBuffered:  metrics.NewGauge(),
				WriteTotal:    metrics.NewCounter(),
				WriteBuffered: metrics.NewGauge(),
			},
			readCollector:  metrics.NilCounter{},
			writeCollector: metrics.NilCounter{},
			tlsMng:         tlsMng,
			tryMutex:       utils.NewMutex(),
		},
		connectTimeout: connectTimeout,
	}

	conn.filterManager = newFilterManager(conn)

	return conn
}

func (cc *clientConnection) Connect() (err error) {
	cc.connectOnce.Do(func() {
		var event api.ConnectionEvent

		timeout := cc.connectTimeout
		if timeout == 0 {
			timeout = DefaultConnectTimeout
		}

		addr := cc.RemoteAddr()
		if addr != nil {
			cc.rawConnection, err = net.DialTimeout("tcp", cc.RemoteAddr().String(), timeout)
		} else {
			err = errors.New("ClientConnection RemoteAddr is nil")
		}

		if err != nil {
			if err == io.EOF {
				// remote conn closed
				event = api.RemoteClose
			} else if err, ok := err.(net.Error); ok && err.Timeout() {
				event = api.ConnectTimeout
			} else {
				event = api.ConnectFailed
			}
		} else {
			atomic.StoreUint32(&cc.connected, 1)
			event = api.Connected

			// ensure ioEnabled and UseNetpollMode
			if UseNetpollMode {
				// store fd
				if tc, ok := cc.rawConnection.(*net.TCPConn); ok {
					cc.file, err = tc.File()
					if err != nil {
						return
					}
				}
			}

			if cc.tlsMng != nil {
				// usually, the client tls manager will never returns an error
				cc.rawConnection, err = cc.tlsMng.Conn(cc.rawConnection)

			}

			if err != nil {
				event = api.ConnectFailed
				cc.rawConnection.Close()
			} else {
				cc.Start(nil)
			}
		}

		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[network] [client connection connect] connect raw tcp, remote address = %s ,event = %+v, error = %+v", cc.remoteAddr, event, err)
		}

		for _, cccb := range cc.connCallbacks {
			cccb.OnEvent(event)
		}
	})

	return
}
