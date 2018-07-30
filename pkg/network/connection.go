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
	"encoding/binary"
	"errors"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network/buffer"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/rcrowley/go-metrics"
	"golang.org/x/sys/unix"
	"io"
	"math/rand"
	"net"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Network related const
const (
	ConnectionCloseDebugMsg = "Close connection %d, event %s, type %s, data read %d, data write %d"
	DefaultBufferCapacity   = 1 << 17
	transferDomainSocket = "/tmp/mosn.sock"

)

var idCounter uint64 = 1
var readerBufferPool = buffer.NewIoBufferPool(DefaultBufferCapacity)
var writeBufferPool = buffer.NewIoBufferPool(DefaultBufferCapacity)

var TransferTimeout = time.Second * 30  //default 30s

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
func NewServerConnection(rawc net.Conn, stopChan chan struct{}, logger log.Logger, ctx context.Context) types.Connection {
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

	if ctx.Value(types.ContextKeyAcceptBuffer) != nil {
		buf := ctx.Value(types.ContextKeyAcceptBuffer).([]byte)
		conn.readBuffer = conn.readerBufferPool.Take(conn.rawConnection)
		conn.readBuffer.Br.Write(buf)
		log.DefaultLogger.Infof("NewServerConnection id = %d, buffer = %d", conn.id, conn.readBuffer.Br.Len())
	}

	if ctx.Value(types.ContextKeyAcceptChan) != nil {
		ch := ctx.Value(types.ContextKeyAcceptChan).(chan types.Connection)
		ch <- conn
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
		case <-c.stopChan:
			if transferTime.IsZero() {
				randTime := time.Duration(rand.Intn(int(TransferTimeout.Nanoseconds())))
				transferTime = time.Now().Add(randTime)
				log.DefaultLogger.Infof("transferTime: Wait %d Second", randTime/1e9)
			}
		default:
		}

		select {
		case <-c.internalStopChan:
			return
		case <-c.readEnabledChan:
		default:
			if !transferTime.IsZero() {
				if transferTime.After(time.Now()) {
					goto transfer
				}
			}
			if c.readEnabled {
				c.rawConnection.SetReadDeadline(time.Now().Add(3 * time.Second))

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
	c.transferChan <- 0
	id, _ := transferRead(c.rawConnection, c.readBuffer.Br.Bytes())
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
			return nil
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
		case <-c.transferChan:
			id = <-c.transferChan
			goto transfer
		case <-c.internalStopChan:
			return
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
	log.DefaultLogger.Infof("TransferWrite begin")
	for {
		select {
		case <-c.writeBufferChan:
			if id == 0 {
				_, err := c.doWrite()
				if err != nil {
					c.logger.Errorf("Error on write. Connection = %d, Remote Address = %s, err = %s",
						c.id, c.RemoteAddr().String(), err)
					return
				}
			} else {
				if c.writeBufLen() == 0 {
					continue
				}
				c.writeBufferMux.Lock()
				transferWrite(c.writeBuffer.Br.Bytes(), id)
				c.writeBuffer.Br.Reset()
				c.writeBufferMux.Unlock()
			}
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

// transferServer is called on new mosn start
func TransferServer(handler types.ConnectionHandler) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("transferServer panic %v", r)
		}
	}()

	if os.Getenv("_MOSN_GRACEFUL_RESTART") != "true" {
		return
	}
	if _, err := os.Stat(transferDomainSocket); err == nil {
		os.Remove(transferDomainSocket)
	}
	l, err := net.Listen("unix", transferDomainSocket)
	if err != nil {
		log.DefaultLogger.Errorf("transfer net listen error %v", err)
		return
	}

	defer l.Close()

	log.DefaultLogger.Infof("TransferServer start")

	var transferMap sync.Map

	go func(handler types.ConnectionHandler) {
		defer func() {
			if r := recover(); r != nil {
				log.DefaultLogger.Errorf("TransferServer panic %v", r)
			}
		}()
		for {
			c, err := l.Accept()
			if err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					log.DefaultLogger.Errorf("listener %s stop accepting connections by deadline", l.Addr())
					return
				} else if ope, ok := err.(*net.OpError); ok {
					// not timeout error and not temporary, which means the error is non-recoverable
					// stop accepting loop and log the event
					if !(ope.Timeout() && ope.Temporary()) {
						// accept error raised by sockets closing
						if ope.Op == "accept" {
							log.DefaultLogger.Errorf("listener %s closed", l.Addr())
						} else {
							log.DefaultLogger.Errorf("listener %s occurs non-recoverable error, stop listening and accepting:%s", l.Addr(), err.Error())
						}
						return
					}
				} else {
					log.DefaultLogger.Errorf("listener %s occurs unknown error while accepting:%s", l.Addr(), err.Error())
				}
			}
			log.DefaultLogger.Infof("transfer Accept")
			go transferHandler(c, handler, &transferMap)
		}
	}(handler)

	select {
	case <-time.After(TransferTimeout * 2 + 10*time.Second):
		log.DefaultLogger.Infof("TransferServer exit")
		return
	}
}

// transferHandler is called on recv transfer request
func transferHandler(c net.Conn, handler types.ConnectionHandler, transferMap *sync.Map) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("transferHandler panic %v", r)
		}
	}()

	defer c.Close()

	uc, ok := c.(*net.UnixConn)
	if !ok {
		log.DefaultLogger.Errorf("unexpected FileConn type; expected UnixConn, got %T", c)
		return
	}
	// recv type
	conn, err := transferRecvType(uc)
	if err != nil {
		return
	}
	// send ack
	err = transferSendMsg(uc, []byte{0})
	if err != nil {
		return
	}
	// recv header
	size, id, err := transferRecvHead(uc)
	if err != nil {
		return
	}
	// recv read/write buffer
	buf, err := transferRecvMsg(uc, size)
	if err != nil {
		return
	}
	if conn != nil {
		// transfer read
		connection := transferNewConn(conn, buf, handler, transferMap)
		if connection != nil {
			transferSendID(uc, connection.id)
		} else {
			transferSendID(uc, 0)
		}
	} else {
		// transfer write
		connection := transferFindConnection(transferMap, uint64(id))
		if connection == nil {
			log.DefaultLogger.Errorf("transferFindConnection failed")
			return
		}
		transferSendBuffer(connection, buf)
	}
}

// old mosn transfer readloop
func transferRead(conn net.Conn, buf []byte) (uint64, error) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("transferRead panic %v", r)
		}
	}()
	c, err := net.Dial("unix", transferDomainSocket)
	if err != nil {
		log.DefaultLogger.Errorf("net Dial unix failed %v", err)
		return 0, err
	}

	defer c.Close()

	tcpconn, ok := conn.(*net.TCPConn)
	if !ok {
		log.DefaultLogger.Errorf("unexpected net.Conn type; expected TCPConn, got %T", conn)
		return 0, errors.New("unexpected Conn type")
	}
	uc := c.(*net.UnixConn)
	// send type and TCP FD
	err = transferSendType(uc, tcpconn, true)
	if err != nil {
		return 0, err
	}
	// recv ack
	_, err = transferRecvMsg(uc, 1)
	if err != nil {
		return 0, err
	}
	// send header
	err = transferSendHead(uc, uint32(len(buf)), 0)
	if err != nil {
		return 0, err
	}
	// send read buffer
	err = transferSendMsg(uc, buf)
	if err != nil {
		return 0, err
	}
	// recv ID
	id := transferRecvID(uc)
	log.DefaultLogger.Infof("TransferRead NewConn id = %d, buffer = %d", id, len(buf))

	return id, nil
}

// old mosn transfer writeloop
func transferWrite(buf []byte, id uint64) error {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("transferWrite panic %v", r)
		}
	}()
	c, err := net.Dial("unix", transferDomainSocket)
	if err != nil {
		log.DefaultLogger.Errorf("net Dial unix failed %v", err)
		return err
	}
	defer c.Close()

	log.DefaultLogger.Infof("TransferWrite id = %d, buffer = %d", id, len(buf))
	uc := c.(*net.UnixConn)
	// send type
	err = transferSendType(uc, nil, false)
	if err != nil {
		return err
	}
	// recv ack
	_, err = transferRecvMsg(uc, 1)
	if err != nil {
		return err
	}
	// send header
	err = transferSendHead(uc, uint32(len(buf)), uint32(id))
	if err != nil {
		return err
	}
	// send write buffer
	return transferSendMsg(uc, buf)
}

func transferSendBuffer(conn *connection, buf []byte) error {
	iobuf := buffer.NewIoBufferBytes(buf)
	return conn.Write(iobuf)
}

func transferFindConnection(transferMap *sync.Map, id uint64) *connection {
	conn, ok := transferMap.Load(id)
	if !ok {
		return nil
	}
	return conn.(*connection)
}

func transferSendHead(uc *net.UnixConn, size uint32, id uint32) error {
	buf := transferBuildHead(size, id)
	return transferSendMsg(uc, buf)
}

/**
 * header protocol (9 bytes)
 * 0     1                 5                       9
 * +-----+-----+-----+-----+-----+-----+-----+-----+
 * |     |   length        |    connetcion  ID     |
 * +-----------+-----------+-----------+-----------+
 *
 **/
func transferBuildHead(size uint32, id uint32) []byte {
	buf := make([]byte, 9)
	binary.BigEndian.PutUint32(buf[1:], size)
	binary.BigEndian.PutUint32(buf[5:], id)
	if id == 0 {
		buf[0] = 0
	} else {
		buf[0] = 1
	}
	return buf
}
/**
 * Type (1 bytes)
 *  0 : transfer read
 *  1 : transfer write
 **/
func transferSendType(uc *net.UnixConn, conn *net.TCPConn, read bool) error {
	buf := make([]byte, 1)
	// transfer write
	if read == false {
		buf[0] = 1
		return transferSendMsg(uc, buf)
	}
	// transfer read, send FD
	return transferSendFD(uc, conn)
}

func transferSendFD(uc *net.UnixConn, conn *net.TCPConn) error {
	var rights []byte
	buf := []byte{0}
	if conn == nil {
		return errors.New("transferSendFD conn is nil")
	}
	f, err := conn.File()
	if err != nil {
		log.DefaultLogger.Errorf("TCP File failed %v", err)
		return err
	}
	rights = syscall.UnixRights(int(f.Fd()))
	n, oobn, err := uc.WriteMsgUnix(buf, rights, nil)
	if err != nil {
		log.DefaultLogger.Errorf("WriteMsgUnix: %v", err)
		return err
	}
	if n != len(buf) || oobn != len(rights) {
		log.DefaultLogger.Errorf("WriteMsgUnix = %d, %d; want 1, %d", n, oobn, len(rights))
		return err
	}
	return nil
}

func transferRecvFD(oob []byte) (net.Conn, error) {
	scms, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		log.DefaultLogger.Errorf("ParseSocketControlMessage: %v", err)
		return nil, err
	}
	if len(scms) != 1 {
		log.DefaultLogger.Errorf("expected 1 SocketControlMessage; got scms = %#v", scms)
		return nil, err
	}
	scm := scms[0]
	gotFds, err := unix.ParseUnixRights(&scm)
	if err != nil {
		log.DefaultLogger.Errorf("unix.ParseUnixRights: %v", err)
		return nil, err
	}
	if len(gotFds) != 1 {
		log.DefaultLogger.Errorf("wanted 1 fd; got %#v", gotFds)
		return nil, err
	}
	f := os.NewFile(uintptr(gotFds[0]), "fd-from-old")
	conn, err := net.FileConn(f)
	if err != nil {
		log.DefaultLogger.Errorf("FileConn error :%v", gotFds)
		return nil, err
	}
	return conn, nil
}

func transferRecvType(uc *net.UnixConn) (net.Conn, error) {
	buf := make([]byte, 1)
	oob := make([]byte, 32)
	_, oobn, _, _, err := uc.ReadMsgUnix(buf, oob)
	if err != nil {
		log.DefaultLogger.Errorf("ReadMsgUnix error: %v", err)
		return nil, err
	}
	// transfer write
	if buf[0] == 1 {
		return nil, nil
	}
	// transfer read, recv FD
	conn, err := transferRecvFD(oob[0:oobn])
	if err != nil {
		return nil, err
	}
	return conn, err
}

func transferRecvHead(uc *net.UnixConn) (int, int, error) {
	buf, err := transferRecvMsg(uc, 9)
	if err != nil {
		log.DefaultLogger.Errorf("ReadMsgUnix error: %v", err)
		return 0, 0, err
	}
	size := int(binary.BigEndian.Uint32(buf[1:]))
	id := int(binary.BigEndian.Uint32(buf[5:]))
	return size, id, nil
}

func transferSendMsg(uc *net.UnixConn, b []byte) error {
	if len(b) == 0 {
		return nil
	}
	_, err := uc.Write(b)
	if err != nil {
		log.DefaultLogger.Errorf("transferSendMsg failed: %v", err)
		return err
	}
	return nil
}

func transferRecvMsg(uc *net.UnixConn, size int) ([]byte, error) {
	if size == 0 {
		return nil, nil
	}
	b := make([]byte, size)
	var n, off int
	var err error
	for {
		n, err = uc.Read(b[off:])
		if err != nil {
			log.DefaultLogger.Errorf("transferRecvMsg failed: %v", err)
			return nil, err
		}
		off += n
		if off == size {
			return b, nil
		}
	}
}

func transferSendID(uc *net.UnixConn, id uint64) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(id))
	return transferSendMsg(uc, b)
}

func transferRecvID(uc *net.UnixConn) uint64 {
	b, err := transferRecvMsg(uc, 4)
	if err != nil {
		return 0
	}
	return uint64(binary.BigEndian.Uint32(b))
}

func transferNewConn(conn net.Conn, buf []byte, handler types.ConnectionHandler, transferMap *sync.Map) *connection {
	address := conn.LocalAddr().(*net.TCPAddr)
	port := strconv.FormatInt(int64(address.Port), 10)
	ipv4, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:"+port)
	ipv6, _ := net.ResolveTCPAddr("tcp", "[::]:"+port)

	var listener types.Listener
	listener = handler.FindListenerByAddress(ipv6)
	if listener == nil {
		listener = handler.FindListenerByAddress(ipv4)
	}
	if listener == nil {
		log.DefaultLogger.Errorf("Find Listener failed %v", address)
		return nil
	}
	ch := make(chan types.Connection)
	go listener.GetListenerCallbacks().OnAccept(conn, false, nil, ch, buf)
	select {
	case rch := <-ch:
		conn, ok := rch.(*connection)
		if !ok {
			log.DefaultLogger.Errorf("transfer NewConn failed %v", address)
			return nil
		}
		log.DefaultLogger.Infof("transfer NewConn id: %d", conn.id)
		transferMap.Store(conn.id, conn)
		return conn
	case <-time.After(1000 * time.Millisecond):
		log.DefaultLogger.Errorf("transfer NewConn timeout %v", address)
		return nil
	}
}