package types

import (
	"net"
	"bytes"
	"crypto/tls"
	"context"
)

type Listener interface {
	Name() string

	Addr() net.Addr

	Start(stopChan chan bool, lctx context.Context)

	ListenerTag() uint64

	SetListenerCallbacks(cb ListenerCallbacks)

	Close(lctx context.Context) error
}

// Callbacks invoked by a listener.
type ListenerCallbacks interface {
	OnAccept(rawc net.Conn, handOffRestoredDestinationConnections bool)

	OnNewConnection(conn Connection)

	OnClose()
}

type FilterStatus string

const (
	Continue      FilterStatus = "Continue"
	StopIteration FilterStatus = "StopIteration"
)

type ListenerFilter interface {
	// Called when a raw connection is accepted, but before a Connection is created.
	OnAccept(cb ListenerFilterCallbacks) FilterStatus
}

// called by listener filter to talk to listener
type ListenerFilterCallbacks interface {
	Conn() net.Conn

	ContinueFilterChain(success bool)
}

// Note: unsupport for now
type ListenerFilterManager interface {
	AddListenerFilter(lf *ListenerFilter)
}

type ConnState string

const (
	Open    ConnState = "Open"
	Closing ConnState = "Closing"
	Closed  ConnState = "Closed"
)

type ConnectionCloseType string

const (
	FlushWrite ConnectionCloseType = "FlushWrite"
	NoFlush    ConnectionCloseType = "NoFlush"
)

type Connection interface {
	Id() uint64

	Start(lctx context.Context)

	// only called by other-stream connection's read loop notifying data buf
	Write(buf *bytes.Buffer) error

	Close(ccType ConnectionCloseType) error

	LocalAddr() net.Addr

	RemoteAddr() net.Addr

	AddConnectionCallbacks(cb ConnectionCallbacks)

	AddBytesSentCallback(cb func(bytesSent uint64))

	NextProtocol() string

	SetNoDelay(enable bool)

	SetReadDisable(disable bool)

	ReadEnabled() bool

	Ssl() *tls.Conn

	SetBufferLimit(limit uint32)

	BufferLimit() uint32

	SetLocalAddress(localAddress net.Addr, restored bool)

	LocalAddressRestored() bool

	GetWriteBuffer() *[]byte

	GetReadBuffer() *bytes.Buffer

	AboveHighWatermark() bool

	FilterManager() FilterManager
}

type ClientConnection interface {
	Connection

	// connect to server in a async way
	Connect()
}

type ConnectionEvent string

const (
	RemoteClose     ConnectionEvent = "RemoteClose"
	LocalClose      ConnectionEvent = "LocalClose"
	OnReadErrClose  ConnectionEvent = "OnReadErrClose"
	OnWriteErrClose ConnectionEvent = "OnWriteErrClose"
	OnConnect       ConnectionEvent = "OnConnect"
	Connected       ConnectionEvent = "Connected"
	ConnectTimeout  ConnectionEvent = "ConnectTimeout"
	ConnectFailed   ConnectionEvent = "ConnectFailed"
)

// Network level callbacks that happen on a connection.
// thread-safe required
type ConnectionCallbacks interface {
	OnEvent(event ConnectionEvent)

	OnAboveWriteBufferHighWatermark()

	OnBelowWriteBufferLowWatermark()
}

type ConnectionHandler interface {
	NumConnections() uint64

	StartListener(l Listener)

	FindListenerByAddress(addr net.Addr) Listener

	RemoveListeners(listenerTag uint64)

	StopListener(listenerTag uint64, lctx context.Context)

	StopListeners(lctx context.Context)
}

// only called by conn read loop
type ReadFilter interface {
	OnData(buffer *bytes.Buffer) FilterStatus

	// example: tcp代理可通过此方法在收到downstream请求时生成upstream connection
	OnNewConnection() FilterStatus

	InitializeReadFilterCallbacks(cb ReadFilterCallbacks)
}

// only called by conn accept loop
type WriteFilter interface {
	OnWrite(buffer *[]byte) FilterStatus
}

// called by read filter to talk to connection
type ReadFilterCallbacks interface {
	Connection() Connection

	ContinueReading()

	UpstreamHost() HostInfo

	SetUpstreamHost(upstreamHost HostInfo)
}

type FilterManager interface {
	AddReadFilter(rf ReadFilter)

	AddWriteFilter(wf WriteFilter)

	ListWriteFilters() []WriteFilter

	ListReadFilter() []ReadFilter

	InitializeReadFilters() bool

	// only called by connection read loop
	OnRead()

	OnWrite() FilterStatus
}

type FilterChainFactory interface {
	CreateNetworkFilterChain(conn Connection)

	CreateListenerFilterChain(listener ListenerFilterManager)
}

type ResponseFlag int

const (
	NoHealthyUpstream             ResponseFlag = 0x2
	UpstreamConnectionFailure     ResponseFlag = 0x20
	UpstreamConnectionTermination ResponseFlag = 0x40
	NoRouteFound                  ResponseFlag = 0x100
	UpstreamOverflow              ResponseFlag = 0x80
)

type Addresses []net.Addr

func (as Addresses) Contains(addr net.Addr) bool {
	for _, one := range as {
		// TODO: support port wildcard
		if one.String() == addr.String() {
			return true
		}
	}

	return false
}
