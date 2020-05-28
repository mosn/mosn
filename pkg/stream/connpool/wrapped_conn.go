package connpool

import (
	"context"
	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/buffer"
	"net"
	"sync"
	"time"
)

var _ api.Connection = &wrappedConn{}

type wrappedConn struct {
	connHoldLock sync.RWMutex
	conn         api.Connection
	ac           *activeClient
}

func (w *wrappedConn) Close(ccType api.ConnectionCloseType, eventType api.ConnectionEvent) error {
	return w.conn.Close(ccType, eventType)
}

func (w *wrappedConn) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *wrappedConn) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *wrappedConn) SetRemoteAddr(address net.Addr) {
	w.conn.SetRemoteAddr(address)
}

func (w *wrappedConn) AddConnectionEventListener(listener api.ConnectionEventListener) {
	w.conn.AddConnectionEventListener(listener)
}

func (w *wrappedConn) AddBytesReadListener(listener func(bytesRead uint64)) {
	w.conn.AddBytesReadListener(listener)
}

func (w *wrappedConn) AddBytesSentListener(listener func(bytesSent uint64)) {
	w.conn.AddBytesSentListener(listener)
}

func (w *wrappedConn) NextProtocol() string {
	return w.conn.NextProtocol()
}

func (w *wrappedConn) SetNoDelay(enable bool) {
	w.conn.SetNoDelay(enable)
}

func (w *wrappedConn) SetReadDisable(disable bool) {
	w.conn.SetReadDisable(disable)
}

func (w *wrappedConn) ReadEnabled() bool {
	return w.conn.ReadEnabled()
}

func (w *wrappedConn) TLS() net.Conn {
	return w.conn.TLS()
}

func (w *wrappedConn) SetBufferLimit(limit uint32) {
	w.conn.SetBufferLimit(limit)
}

func (w *wrappedConn) BufferLimit() uint32 {
	return w.conn.BufferLimit()
}

func (w *wrappedConn) SetLocalAddress(localAddress net.Addr, restored bool) {
	w.conn.SetLocalAddress(localAddress, restored)
}

func (w *wrappedConn) SetCollector(read, write gometrics.Counter) {
	w.conn.SetCollector(read, write)
}

func (w *wrappedConn) LocalAddressRestored() bool {
	return w.conn.LocalAddressRestored()
}

func (w *wrappedConn) GetWriteBuffer() []buffer.IoBuffer {
	return w.conn.GetWriteBuffer()
}

func (w *wrappedConn) GetReadBuffer() buffer.IoBuffer {
	return w.conn.GetReadBuffer()
}

func (w *wrappedConn) FilterManager() api.FilterManager {
	return w.conn.FilterManager()
}

func (w *wrappedConn) RawConn() net.Conn {
	return w.conn.RawConn()
}

func (w *wrappedConn) SetTransferEventListener(listener func() bool) {
	w.conn.SetTransferEventListener(listener)

}

func (w *wrappedConn) SetIdleTimeout(d time.Duration) {
	w.conn.SetIdleTimeout(d)
}

func (w *wrappedConn) State() api.ConnState {
	return w.conn.State()
}

func (w *wrappedConn) ID() uint64 {
	return w.conn.ID()
}

func (w *wrappedConn) Start(lctx context.Context) {
	w.conn.Start(lctx)
}

func (w *wrappedConn) Write(buf ...buffer.IoBuffer) error {

	w.connHoldLock.RLock()
	err := w.conn.Write(buf...)
	w.connHoldLock.RUnlock()

	if err == nil {
		return nil
	}

	reconnError := w.ac.Reconnect()

	if reconnError != nil {
		log.DefaultLogger.Warnf("reconnect fail : %v", reconnError.Error())
	}

	return err
}
