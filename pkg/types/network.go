package types

import (
	"context"
	"io"
	"net"

	"github.com/rcrowley/go-metrics"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
)

//
//   The bunch of interfaces are structure skeleton to build a high performance, extensible network framework.
//
//   In mosn, we have 4 layers to build a mesh, net/io layer is the fundamental layer to support upper level's functionality.
//	 -----------------------
//   |        PROXY          |
//    -----------------------
//   |       STREAMING       |
//    -----------------------
//   |        PROTOCOL       |
//    -----------------------
//   |         NET/IO        |
//    -----------------------
//
//   Core model in network layer are listener and connection. Listener listens specified port, waiting for new connections.
//   Both listener and connection have a extension mechanism, implemented as listener and filter chain, which are used to fill in customized logic.
//   Event listeners are used to subscribe important event of Listener and Connection. Method in listener will be called on event occur, but not effect the control flow.
//   Filters are called on event occurs, it also returns a status to effect control flow. Currently 2 states are used: Continue to let it go, StopIteration to stop the control flow.
//   Filter has a callback handler to interactive with core model. For example, ReadFilterCallbacks can be used to continue filter chain in connection, on which is in a stopped state.
//
//   Listener:
//   	- Event listener
// 			- ListenerEventListener
//      - Filter
// 			- ListenerFilter
//   Connection:
//		- Event listener
// 			- ConnectionEventListener
//		- Filter
// 			- ReadFilter
//			- WriteFilter
//
//   Below is the basic relation on listener and connection:
//    --------------------------------------------------
//   |                                      			|
//   | 	  EventListener       EventListener     		|
//   |        *|                   |*          		    |
//   |         |                   |       				|
//   |        1|     1      *      |1          			|
// 	 |	    Listener --------- Connection      			|
//   |        1|      [accpet]     |1          			|
//	 |         |                   |-----------         |
//   |        *|                   |*          |*       |
//   |	 ListenerFilter       ReadFilter  WriteFilter   |
//   |                                                  |
//    --------------------------------------------------
//

const (
	ListenerStatsPrefix = "listener.%d."
)

// listener interface
type Listener interface {
	// Listener name
	Name() string

	// Listener address
	Addr() net.Addr

	// Start starts listener with context
	Start(lctx context.Context)

	// Stop stops listener
	// Accepted connections will not be closed
	Stop()

	// Listener tag,
	ListenerTag() uint64

	// Return a copy a listener fd
	ListenerFD() (uintptr, error)

	// Limit bytes per connection
	PerConnBufferLimitBytes() uint32

	// Set listener event listener
	SetListenerCallbacks(cb ListenerEventListener)

	// Close listener, not closing connections
	Close(lctx context.Context) error
}

// TLS ContextManager
type TLSContextManager interface {
	Conn(c net.Conn) net.Conn
	Enabled() bool
}

// Callbacks invoked by a listener.
type ListenerEventListener interface {
	// Called on new connection accepted
	OnAccept(rawc net.Conn, handOffRestoredDestinationConnections bool)

	// Called on new mosn connection created
	OnNewConnection(conn Connection, ctx context.Context)

	// Called on listener close
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

// A callback handler called by listener filter to talk to listener
type ListenerFilterCallbacks interface {
	// Connection reference used in callback handler
	Conn() net.Conn

	// Continue filter chain
	ContinueFilterChain(success bool, ctx context.Context)

	// Set original addr
	SetOrigingalAddr(ip string, port int)
}

// Note: unsupport for now
type ListenerFilterManager interface {
	AddListenerFilter(lf *ListenerFilter)
}

type IoBuffer interface {
	// Read reads the next len(p) bytes from the buffer or until the buffer
	// is drained. The return value n is the number of bytes read. If the
	// buffer has no data to return, err is io.EOF (unless len(p) is zero);
	// otherwise it is nil.
	Read(p []byte) (n int, err error)

	// ReadOnce make a one-shot read and appends it to the buffer, growing
	// the buffer as needed. The return value n is the number of bytes read. Any
	// error except io.EOF encountered during the read is also returned. If the
	// buffer becomes too large, ReadFrom will panic with ErrTooLarge.
	ReadOnce(r io.Reader) (n int64, err error)

	// ReadFrom reads data from r until EOF and appends it to the buffer, growing
	// the buffer as needed. The return value n is the number of bytes read. Any
	// error except io.EOF encountered during the read is also returned. If the
	// buffer becomes too large, ReadFrom will panic with ErrTooLarge.
	ReadFrom(r io.Reader) (n int64, err error)

	// Write appends the contents of p to the buffer, growing the buffer as
	// needed. The return value n is the length of p; err is always nil. If the
	// buffer becomes too large, Write will panic with ErrTooLarge.
	Write(p []byte) (n int, err error)

	// WriteTo writes data to w until the buffer is drained or an error occurs.
	// The return value n is the number of bytes written; it always fits into an
	// int, but it is int64 to match the io.WriterTo interface. Any error
	// encountered during the write is also returned.
	WriteTo(w io.Writer) (n int64, err error)

	// Peek returns n bytes from buffer, without draining any buffered data.
	// If n > readable buffer, nil will be returned.
	// It can be used in codec to check first-n-bytes magic bytes
	// Note: do not change content in return bytes, use write instead
	Peek(n int) []byte

	// Bytes returns all bytes from buffer, without draining any buffered data.
	// It can be used to get fixed-length content, such as headers, body.
	// Note: do not change content in return bytes, use write instead
	Bytes() []byte

	// Drain drains a offset length of bytes in buffer.
	// It can be used with Bytes(), after consuming a fixed-length of data
	Drain(offset int)

	// Len returns the number of bytes of the unread portion of the buffer;
	// b.Len() == len(b.Bytes()).
	Len() int

	// Cap returns the capacity of the buffer's underlying byte slice, that is, the
	// total space allocated for the buffer's data.
	Cap() int

	// Reset resets the buffer to be empty,
	// but it retains the underlying storage for use by future writes.
	Reset()

	// Clone makes a copy of IoBuffer struct
	Clone() IoBuffer

	// String returns the contents of the unread portion of the buffer
	// as a string. If the Buffer is a nil pointer, it returns "<nil>".
	String() string
}

type BufferWatermarkListener interface {
	OnHighWatermark()

	OnLowWatermark()
}

// A buffer pool to reuse header map.
// Normally recreate a map has minor cpu/memory cost, however in a high concurrent scenario, a buffer pool is needed to recycle map
type HeadersBufferPool interface {
	// Take finds and returns a map from buffer pool. If no buffer available, create a new one with capacity.
	Take(capacity int) map[string]string

	// Give returns a map to buffer pool
	Give(amap map[string]string)
}

type GenericMapPool interface {
	Take(defaultSize int) (amap map[string]interface{})

	Give(amap map[string]interface{})
}

type ObjectBufferPool interface {
	Take() interface{}

	Give(object interface{})
}

// Connection status
type ConnState string

const (
	Open    ConnState = "Open"
	Closing ConnState = "Closing"
	Closed  ConnState = "Closed"
)

// Connection close type
type ConnectionCloseType string

const (
	// Flush write buffer to underlying io then close connection
	FlushWrite ConnectionCloseType = "FlushWrite"
	// Close connection without flushing buffer
	NoFlush ConnectionCloseType = "NoFlush"
)

// Connection interface
type Connection interface {
	// Unique connection id
	Id() uint64

	// Start starts connection with context.
	// See context.go to get available keys in context
	Start(lctx context.Context)

	// Write writes data to the connection.
	// Called by other-side stream connection's read loop. Will loop through stream filters with the buffer if any are installed.
	Write(buf ...IoBuffer) error

	// Close closes connection with connection type and event type.
	// ConnectionCloseType - how to close to connection
	// 	- FlushWrite: connection will be closed after buffer flushed to underlying io
	//	- NoFlush: close connection asap
	// ConnectionEvent - why to close the connection
	// 	- RemoteClose
	//  - LocalClose
	// 	- OnReadErrClose
	//  - OnWriteErrClose
	//  - OnConnect
	//  - Connected:
	//	- ConnectTimeout
	//	- ConnectFailed
	Close(ccType ConnectionCloseType, eventType ConnectionEvent) error

	// LocalAddr returns the local address of the connection.
	// For client connection, this is the origin address
	// For server connection, this is the proxy's address
	// TODO: support get local address in redirected request
	// TODO: support transparent mode
	LocalAddr() net.Addr

	// RemoteAddr returns the remote address of the connection.
	RemoteAddr() net.Addr

	// Add connection level event listener, listener method will be called when connection event occur.
	AddConnectionEventListener(cb ConnectionEventListener)

	// Add io bytes read listener method, method will be called everytime bytes read
	AddBytesReadListener(cb func(bytesRead uint64))

	// Add io bytes write listener method, method will be called everytime bytes write
	AddBytesSentListener(cb func(bytesSent uint64))

	// Network level negotiation, such as ALPN. Returns empty string if not supported.
	NextProtocol() string

	// Enable/disable tcp no delay
	SetNoDelay(enable bool)

	// Enable/disable read on the connection.
	// If reads are enabled after disable, connection continues to read and data will be dispatched to read filter chains
	SetReadDisable(disable bool)

	// Whether reading is enabled on the connection
	ReadEnabled() bool

	// Returns a related tls connection
	TLS() net.Conn

	// Set buffer limit
	SetBufferLimit(limit uint32)

	// Return buffer limit
	BufferLimit() uint32

	// Set a local address
	SetLocalAddress(localAddress net.Addr, restored bool)

	// Inject a connection stats
	SetStats(stats *ConnectionStats)

	// todo: unsupported for now
	LocalAddressRestored() bool

	// Get write buffer, used by network writer filter
	GetWriteBuffer() []IoBuffer

	// Get read buffer, used by network read filter
	GetReadBuffer() IoBuffer

	AboveHighWatermark() bool

	FilterManager() FilterManager

	// Caution: raw conn only used in io-loop disable mode
	// todo: a better way to provide raw conn
	RawConn() net.Conn
}

type ConnectionStats struct {
	ReadTotal    metrics.Counter
	ReadCurrent  metrics.Gauge
	WriteTotal   metrics.Counter
	WriteCurrent metrics.Gauge
}

type ClientConnection interface {
	Connection

	// connect to server in a async way
	Connect(ioEnabled bool) error
}

type ConnectionEvent string

const (
	RemoteClose     ConnectionEvent = "RemoteClose"
	LocalClose      ConnectionEvent = "LocalClose"
	OnReadErrClose  ConnectionEvent = "OnReadErrClose"
	OnWriteErrClose ConnectionEvent = "OnWriteErrClose"
	OnConnect       ConnectionEvent = "OnConnect"
	Connected       ConnectionEvent = "ConnectedFlag"
	ConnectTimeout  ConnectionEvent = "ConnectTimeout"
	ConnectFailed   ConnectionEvent = "ConnectFailed"
)

func (ce ConnectionEvent) IsClose() bool {
	return ce == LocalClose || ce == RemoteClose ||
		ce == OnReadErrClose || ce == OnWriteErrClose
}

func (ce ConnectionEvent) ConnectFailure() bool {
	return ce == ConnectFailed || ce == ConnectTimeout
}

// Network level callbacks that happen on a connection.
type ConnectionEventListener interface {
	// Called on ConnectionEvent
	OnEvent(event ConnectionEvent)

	// Called on write buffer's data run above write buffer
	OnAboveWriteBufferHighWatermark()

	// Called on write buffer's data run below write buffer
	OnBelowWriteBufferLowWatermark()
}

type ConnectionHandler interface {
	// Num of connections
	NumConnections() uint64

	// Add a listener
	AddListener(lc *v2.ListenerConfig, networkFiltersFactory NetworkFilterChainFactory, streamFiltersFactories []StreamFilterChainFactory)

	// Start a listener by tag
	StartListener(listenerTag uint64, lctx context.Context)

	// Start all listeners
	StartListeners(lctx context.Context)

	// Find a listener by address
	FindListenerByAddress(addr net.Addr) Listener

	// Remove listener by tag
	RemoveListeners(listenerTag uint64)

	// Stop listener by tag
	StopListener(listenerTag uint64, lctx context.Context)

	// Stop all listener
	StopListeners(lctx context.Context)

	// List all listeners' fd
	ListListenersFD(lctx context.Context) []uintptr
}

// Connection binary read filter
// Registered by FilterManager.AddReadFilter
type ReadFilter interface {
	// Called everytime bytes is read from the connection
	OnData(buffer IoBuffer) FilterStatus

	// Called on new connection is created
	OnNewConnection() FilterStatus

	// Initial read filter callbacks. It used by init read filter
	InitializeReadFilterCallbacks(cb ReadFilterCallbacks)
}

// Connection binary write filter
// only called by conn accept loop
type WriteFilter interface {
	// Called before data write to raw connection
	OnWrite(buffer []IoBuffer) FilterStatus
}

// Called by read filter to talk to connection
type ReadFilterCallbacks interface {
	// Get connection
	Connection() Connection

	// Continue reading filter iteration on filter stopped, next filter will be called with current read buffer
	ContinueReading()

	// Return current selected upstream host.
	UpstreamHost() HostInfo

	// Set currently selected upstream host.
	SetUpstreamHost(upstreamHost HostInfo)
}

type FilterManager interface {
	// Add a read filter
	AddReadFilter(rf ReadFilter)

	// Add a write filter
	AddWriteFilter(wf WriteFilter)

	// List read filters
	ListReadFilter() []ReadFilter

	// List write filters
	ListWriteFilters() []WriteFilter

	// Init read filters
	InitializeReadFilters() bool

	// Called on data read
	OnRead()

	// Called before data write
	OnWrite() FilterStatus
}

type FilterChainFactory interface {
	CreateNetworkFilterChain(conn Connection)

	CreateListenerFilterChain(listener ListenerFilterManager)
}

type NetworkFilterFactoryCb func(manager FilterManager)

type NetworkFilterChainFactory interface {
	CreateFilterFactory(clusterManager ClusterManager, context context.Context) NetworkFilterFactoryCb
}

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
