package server

import (
	"container/list"
	"net"
	"context"
	"sync/atomic"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
)

type connHandler struct {
	numConnections int64
	listeners      []*activeListener
	clusterManager types.ClusterManager
	filterFactory  NetworkFilterConfigFactory
}

func NewHandler(filterFactory NetworkFilterConfigFactory) types.ConnectionHandler {
	return newConnHandler(filterFactory)
}

func newConnHandler(filterFactory NetworkFilterConfigFactory) *connHandler {
	ch := &connHandler{
		numConnections: 0,
		clusterManager: cluster.NewClusterManager(nil),
		listeners:      make([]*activeListener, 0),
		filterFactory:  filterFactory,
	}

	return ch
}

func (ch *connHandler) NumConnections() uint64 {
	return uint64(atomic.LoadInt64(&ch.numConnections))
}

func (ch *connHandler) StartListener(l types.Listener) {
	listenerStopChan := make(chan bool)

	al := newActiveListener(l, ch, listenerStopChan)
	l.SetListenerCallbacks(al)

	ch.listeners = append(ch.listeners, al)

	// TODO: use goruntine pool
	go l.Start(al.stopChan, nil)
}

func (ch *connHandler) FindListenerByAddress(addr net.Addr) types.Listener {
	l := ch.findActiveListenerByAddress(addr)

	if l == nil {
		return nil
	} else {
		return l.listener
	}
}

func (ch *connHandler) RemoveListeners(listenerTag uint64) {
	for i, l := range ch.listeners {
		if l.listener.ListenerTag() == listenerTag {
			ch.listeners = append(ch.listeners[:i], ch.listeners[i+1:]...)
		}
	}
}

func (ch *connHandler) StopListener(listenerTag uint64, lctx context.Context) {
	for _, l := range ch.listeners {
		if l.listener.ListenerTag() == listenerTag {
			// stop goruntine
			l.stopChan <- true
		}
	}
}

func (ch *connHandler) StopListeners(lctx context.Context) {
	for _, l := range ch.listeners {
		// stop goruntine
		l.stopChan <- true
	}
}

func (ch *connHandler) findActiveListenerByAddress(addr net.Addr) *activeListener {
	for _, l := range ch.listeners {
		if l.listener != nil {
			if l.listener.Addr().Network() == addr.Network() &&
				l.listener.Addr().String() == addr.String() {
				return l
			}
		}
	}

	return nil
}

// ListenerCallbacks
type activeListener struct {
	listener types.Listener
	conns    *list.List
	handler  *connHandler
	stopChan chan bool
}

func newActiveListener(listener types.Listener, handler *connHandler, stopChan chan bool) *activeListener {
	al := &activeListener{
		listener: listener,
		conns:    list.New(),
		handler:  handler,
		stopChan: stopChan,
	}

	return al
}

// ListenerCallbacks
func (al *activeListener) OnAccept(rawc net.Conn, handOffRestoredDestinationConnections bool) {
	arc := newActiveRawConn(rawc, al)
	// TODO: create listener filter chain

	arc.ContinueFilterChain(true)
}

func (al *activeListener) OnNewConnection(conn types.Connection) {
	configFactory := al.handler.filterFactory.CreateFilterFactory(al.handler.clusterManager)
	buildFilterChain(conn.FilterManager(), configFactory)

	filterManager := conn.FilterManager()
	if len(filterManager.ListReadFilter()) == 0 &&
		len(filterManager.ListWriteFilters()) == 0 {
		// no filter found, close connection
		conn.Close(types.NoFlush)
	} else {
		ac := newActiveConnection(al, conn)

		conn.AddConnectionCallbacks(ac)
		e := al.conns.PushBack(ac)
		ac.element = e

		atomic.AddInt64(&al.handler.numConnections, 1)

		ac.conn.Start(nil)
	}
}

func (al *activeListener) OnClose() {}

func (al *activeListener) removeConnection(ac *activeConnection) {
	al.conns.Remove(ac.element)
	atomic.AddInt64(&al.handler.numConnections, -1)
}

func (al *activeListener) newConnection(rawc net.Conn) {
	conn := network.NewServerConnection(rawc, al.stopChan)

	var limit uint32
	// TODO: read from config.perConnectionBufferLimitBytes()
	limit = 4 * 1024
	conn.SetBufferLimit(limit)

	al.OnNewConnection(conn)
}

type activeRawConn struct {
	rawc               net.Conn
	rawcElement        *list.Element
	activeListener     *activeListener
	acceptedFilters    []types.ListenerFilter
	accptedFilterIndex int
}

func newActiveRawConn(rawc net.Conn, activeListener *activeListener) *activeRawConn {
	return &activeRawConn{
		rawc:           rawc,
		activeListener: activeListener,
	}
}

func (arc *activeRawConn) ContinueFilterChain(success bool) {
	if success {
		for ; arc.accptedFilterIndex < len(arc.acceptedFilters); arc.accptedFilterIndex++ {
			filterStatus := arc.acceptedFilters[arc.accptedFilterIndex].OnAccept(arc)

			if filterStatus == types.StopIteration {
				return
			}
		}

		// TODO: handle hand_off_restored_destination_connections logic

		arc.activeListener.newConnection(arc.rawc)
	}
}

func (arc *activeRawConn) Conn() net.Conn {
	return arc.rawc
}

// ConnectionCallbacks
// ListenerFilterManager note:unsupported now
// ListenerFilterCallbacks note:unsupported now
type activeConnection struct {
	element  *list.Element
	listener *activeListener
	conn     types.Connection
}

func newActiveConnection(listener *activeListener, conn types.Connection) *activeConnection {
	ac := &activeConnection{
		conn:     conn,
		listener: listener,
	}

	ac.conn.SetNoDelay(true)
	ac.conn.AddConnectionCallbacks(ac)

	// TODO: downstream stats inc

	return ac
}

// ConnectionCallbacks
func (ac *activeConnection) OnEvent(event types.ConnectionEvent) {
	if event == types.LocalClose || event == types.RemoteClose {
		ac.listener.removeConnection(ac)
	}
}

// ConnectionCallbacks
func (ac *activeConnection) OnAboveWriteBufferHighWatermark() {}

// ConnectionCallbacks
func (ac *activeConnection) OnBelowWriteBufferLowWatermark() {}
