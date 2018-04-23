package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rcrowley/go-metrics"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

const (
	ConnectionCloseDebugMsg = "Close connection %d, event %s, type %s, data read %d, data write %d"
	DefaultBufferCapacity   = 1 << 12
)

var idCounter uint64
var readerBufferPool = buffer.NewIoBufferPoolV2(1, DefaultBufferCapacity)
var writeBufferPool = buffer.NewIoBufferPoolV2(1, DefaultBufferCapacity*2)

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
	connCallbacks        []types.ConnectionEventListener
	bytesReadCallbacks   []func(bytesRead uint64)
	bytesSendCallbacks   []func(bytesSent uint64)
	filterManager        types.FilterManager

	stopChan           chan bool
	curWriteBufferData []types.IoBuffer
	readBuffer         *buffer.IoBufferPoolEntry
	writeBuffer        *buffer.IoBufferPoolEntry
	writeBufferMux     sync.RWMutex
	writeBufferChan    chan bool
	writeLoopStopChan  chan bool
	readerBufferPool   *buffer.IoBufferPoolV2
	writeBufferPool    *buffer.IoBufferPoolV2

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
		readEnabled:       true,
		readEnabledChan:   make(chan bool, 1),
		writeBufferChan:   make(chan bool),
		writeLoopStopChan: make(chan bool, 1),
		readerBufferPool:  readerBufferPool,
		writeBufferPool:   writeBufferPool,
		stats: &types.ConnectionStats{
			ReadTotal:    metrics.NewCounter(),
			ReadCurrent:  metrics.NewGauge(),
			WriteTotal:   metrics.NewCounter(),
			WriteCurrent: metrics.NewGauge(),
		},
		logger: log.DefaultLogger,
	}

	//conn.writeBuffer = buffer.NewWatermarkBuffer(DefaultWriteBufferCapacity, conn)
	conn.filterManager = newFilterManager(conn)

	return conn
}

// watermark listener
func (c *connection) OnHighWatermark() {
	c.aboveHighWatermark = true

	for _, cb := range c.connCallbacks {
		cb.OnAboveWriteBufferHighWatermark()
	}
}

func (c *connection) OnLowWatermark() {
	c.aboveHighWatermark = false

	for _, cb := range c.connCallbacks {
		cb.OnBelowWriteBufferLowWatermark()
	}
}

// basic

func (c *connection) Id() uint64 {
	return c.id
}

func (c *connection) Start(lctx context.Context) {
	c.startOnce.Do(func() {

		go func() {
			defer func() {
				if p := recover(); p != nil {
					// TODO: panic recover @wugou

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
					// TODO: panic recover @wugou

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

	if c.readBuffer.Br.Len() == 0 {
		c.readerBufferPool.Give(c.readBuffer)
		c.readBuffer = nil
	}

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

	c.writeBufferMux.Unlock()

	go func() {
		c.writeBufferChan <- true
	}()

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

		//if c.writeBuffer.Br.Len() == 0 {
		//	c.writeBufferPool.Give(c.writeBuffer)
		//	c.writeBuffer = nil
		//}
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

	wbLen := c.writeBuffer.Br.Len()

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

func (c *connection) AddConnectionEventListener(cb types.ConnectionEventListener) {
	exist := false

	for _, ccb := range c.connCallbacks {
		if &ccb == &cb {
			exist = true
		}
	}

	if !exist {
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
		rawc := c.rawConnection.(*net.TCPConn)
		rawc.SetNoDelay(enable)
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

func (c *connection) Ssl() *tls.Conn {
	return nil
}

func (c *connection) SetBufferLimit(limit uint32) {
	if limit > 0 {
		c.bufferLimit = limit

		//c.writeBuffer.(*buffer.WatermarkBuffer).SetWaterMark(limit)
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

func (c *connection) RawConn() net.Conn {
	return c.rawConnection
}

type clientConnection struct {
	connection

	connectOnce sync.Once
}

func NewClientConnection(sourceAddr net.Addr, remoteAddr net.Addr, stopChan chan bool) types.ClientConnection {
	id := atomic.AddUint64(&idCounter, 1)

	if log.DefaultLogger == nil {
		log.InitDefaultLogger("", log.DEBUG)
	}

	conn := &clientConnection{
		connection: connection{
			id:                id,
			localAddr:         sourceAddr,
			remoteAddr:        remoteAddr,
			stopChan:          stopChan,
			readEnabled:       true,
			readEnabledChan:   make(chan bool, 1),
			writeBufferChan:   make(chan bool),
			writeLoopStopChan: make(chan bool, 1),
			readerBufferPool:  readerBufferPool,
			writeBufferPool:   writeBufferPool,
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

	return conn
}

func (cc *clientConnection) Connect(ioEnabled bool) (err error) {
	cc.connectOnce.Do(func() {
		var localTcpAddr *net.TCPAddr

		if cc.localAddr != nil {
			localTcpAddr, err = net.ResolveTCPAddr("tcp", cc.localAddr.String())
		}

		var remoteTcpAddr *net.TCPAddr
		remoteTcpAddr, err = net.ResolveTCPAddr("tcp", cc.remoteAddr.String())

		cc.rawConnection, err = net.DialTCP("tcp", localTcpAddr, remoteTcpAddr)
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

			if ioEnabled {
				cc.Start(nil)
			}
		}

		for _, cccb := range cc.connCallbacks {
			cccb.OnEvent(event)
		}
	})

	return
}
