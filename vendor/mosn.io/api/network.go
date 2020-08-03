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

package api

import (
	"context"
	"net"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"mosn.io/pkg/buffer"
)

// Connection status
type ConnState int

// Connection statuses
const (
	ConnInit ConnState = iota
	ConnActive
	ConnClosed
)

// ConnectionCloseType represent connection close type
type ConnectionCloseType string

//Connection close types
const (
	// FlushWrite means write buffer to underlying io then close connection
	FlushWrite ConnectionCloseType = "FlushWrite"
	// NoFlush means close connection without flushing buffer
	NoFlush ConnectionCloseType = "NoFlush"
)

// Connection interface
type Connection interface {
	// ID returns unique connection id
	ID() uint64

	// Start starts connection with context.
	// See context.go to get available keys in context
	Start(lctx context.Context)

	// Write writes data to the connection.
	// Called by other-side stream connection's read loop. Will loop through stream filters with the buffer if any are installed.
	Write(buf ...buffer.IoBuffer) error

	// Close closes connection with connection type and event type.
	// ConnectionCloseType - how to close to connection
	//      - FlushWrite: connection will be closed after buffer flushed to underlying io
	//      - NoFlush: close connection asap
	// ConnectionEvent - why to close the connection
	//      - RemoteClose
	//  - LocalClose
	//      - OnReadErrClose
	//  - OnWriteErrClose
	//  - OnConnect
	//  - Connected:
	//      - ConnectTimeout
	//      - ConnectFailed
	Close(ccType ConnectionCloseType, eventType ConnectionEvent) error

	// LocalAddr returns the local address of the connection.
	// For client connection, this is the origin address
	// For server connection, this is the proxy's address
	// TODO: support get local address in redirected request
	// TODO: support transparent mode
	LocalAddr() net.Addr

	// RemoteAddr returns the remote address of the connection.
	RemoteAddr() net.Addr

	// SetRemoteAddr is used for originaldst we need to replace remoteAddr
	SetRemoteAddr(address net.Addr)

	// AddConnectionEventListener add a listener method will be called when connection event occur.
	AddConnectionEventListener(listener ConnectionEventListener)

	// AddBytesReadListener add a method will be called everytime bytes read
	AddBytesReadListener(listener func(bytesRead uint64))

	// AddBytesSentListener add a method will be called everytime bytes write
	AddBytesSentListener(listener func(bytesSent uint64))

	// NextProtocol returns network level negotiation, such as ALPN. Returns empty string if not supported.
	NextProtocol() string

	// SetNoDelay enable/disable tcp no delay
	SetNoDelay(enable bool)

	// SetReadDisable enable/disable read on the connection.
	// If reads are enabled after disable, connection continues to read and data will be dispatched to read filter chains.
	SetReadDisable(disable bool)

	// ReadEnabled returns whether reading is enabled on the connection.
	ReadEnabled() bool

	// TLS returns a related tls connection.
	TLS() net.Conn

	// SetBufferLimit set the buffer limit.
	SetBufferLimit(limit uint32)

	// BufferLimit returns the buffer limit.
	BufferLimit() uint32

	// SetLocalAddress sets a local address
	SetLocalAddress(localAddress net.Addr, restored bool)

	// SetCollector set read/write mertics collectors
	SetCollector(read, write metrics.Counter)
	// LocalAddressRestored returns whether local address is restored
	// TODO: unsupported now
	LocalAddressRestored() bool

	// GetWriteBuffer is used by network writer filter
	GetWriteBuffer() []buffer.IoBuffer

	// GetReadBuffer is used by network read filter
	GetReadBuffer() buffer.IoBuffer

	// FilterManager returns the FilterManager
	FilterManager() FilterManager

	// RawConn returns the original connections.
	// Caution: raw conn only used in io-loop disable mode
	// TODO: a better way to provide raw conn
	RawConn() net.Conn

	// SetTransferEventListener set a method will be called when connection transfer occur
	SetTransferEventListener(listener func() bool)

	// SetIdleTimeout sets the timeout that will set the connnection to idle. At intervals of readTimeout,
	// Connections can be closed after idle for idleTimeout/readTimeout checks. Mosn close idle connections
	// if no idle timeout set，a zero value for idleTimeout means no idle connections.
	SetIdleTimeout(readTimeout time.Duration, idleTimeout time.Duration)

	// State returns the connection state
	State() ConnState

	// OnRead deals with data not read from doRead process
	OnRead(buffer buffer.IoBuffer)
}

// ConnectionEvent type
type ConnectionEvent string

// ConnectionEvent types
const (
	RemoteClose     ConnectionEvent = "RemoteClose"
	LocalClose      ConnectionEvent = "LocalClose"
	OnReadErrClose  ConnectionEvent = "OnReadErrClose"
	OnWriteErrClose ConnectionEvent = "OnWriteErrClose"
	OnConnect       ConnectionEvent = "OnConnect"
	Connected       ConnectionEvent = "ConnectedFlag"
	ConnectTimeout  ConnectionEvent = "ConnectTimeout"
	ConnectFailed   ConnectionEvent = "ConnectFailed"
	OnReadTimeout   ConnectionEvent = "OnReadTimeout"
	OnWriteTimeout  ConnectionEvent = "OnWriteTimeout"
)

// IsClose represents whether the event is triggered by connection close
func (ce ConnectionEvent) IsClose() bool {
	return ce == LocalClose || ce == RemoteClose ||
		ce == OnReadErrClose || ce == OnWriteErrClose || ce == OnWriteTimeout
}

// ConnectFailure represents whether the event is triggered by connection failure
func (ce ConnectionEvent) ConnectFailure() bool {
	return ce == ConnectFailed || ce == ConnectTimeout
}

// ConnectionEventListener is a network level callbacks that happen on a connection.
type ConnectionEventListener interface {
	// OnEvent is called on ConnectionEvent
	OnEvent(event ConnectionEvent)
}
