package server

import (
	"sync"
	"container/list"
	"net"
	"context"
	"sync/atomic"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"github.com/rcrowley/go-metrics"
	"fmt"
)

// ConnectionHandler
// ClusterConfigFactoryCb
// ClusterHostFactoryCb
type connHandler struct {
	numConnections int64
	listeners      []*activeListener
	clusterManager types.ClusterManager
	filterFactory  NetworkFilterConfigFactory
	logger         log.Logger
}

func NewHandler(filterFactory NetworkFilterConfigFactory, clusterManagerFilter ClusterManagerFilter, logger log.Logger) types.ConnectionHandler {
	ch := &connHandler{
		numConnections: 0,
		clusterManager: cluster.NewClusterManager(nil),
		listeners:      make([]*activeListener, 0),
		filterFactory:  filterFactory,
		logger:         logger,
	}

	clusterManagerFilter.OnCreated(ch, ch)

	return ch
}

// ClusterConfigFactoryCb
func (ch *connHandler) UpdateClusterConfig(clusters []v2.Cluster) error {

	for _, cluster := range clusters {
		ch.clusterManager.AddOrUpdatePrimaryCluster(cluster)
	}

	// TODO: remove cluster

	return nil
}

// ClusterHostFactoryCb
func (ch *connHandler) UpdateClusterHost(cluster string, priority uint32, hosts []v2.Host) error {
	return ch.clusterManager.UpdateClusterHosts(cluster, priority, hosts)
}

// ConnectionHandler
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
	connsMux sync.RWMutex
	handler  *connHandler
	stopChan chan bool
	stats    *ListenerStats
}

type ListenerStats struct {
	downstreamTotal             metrics.Counter
	downstreamDestroy           metrics.Counter
	downstreamActive            metrics.Gauge
	downstreamBytesRead         metrics.Counter
	downstreamBytesReadCurrent  metrics.Gauge
	downstreamBytesWrite        metrics.Counter
	downstreamBytesWriteCurrent metrics.Gauge
}

func newActiveListener(listener types.Listener, handler *connHandler, stopChan chan bool) *activeListener {
	al := &activeListener{
		listener: listener,
		conns:    list.New(),
		handler:  handler,
		stopChan: stopChan,
		stats:    newListenerStats(listener),
	}

	return al
}

func newListenerStats(listener types.Listener) *ListenerStats {
	addr := listener.Addr().String()

	return &ListenerStats{
		downstreamTotal:             metrics.GetOrRegisterCounter(listenerStatsName("connection_total", addr), nil),
		downstreamDestroy:           metrics.GetOrRegisterCounter(listenerStatsName("connection_destroy", addr), nil),
		downstreamActive:            metrics.GetOrRegisterGauge(listenerStatsName("connection_active", addr), nil),
		downstreamBytesRead:         metrics.GetOrRegisterCounter(listenerStatsName("bytes_read", addr), nil),
		downstreamBytesReadCurrent:  metrics.GetOrRegisterGauge(listenerStatsName("bytes_read_current", addr), nil),
		downstreamBytesWrite:        metrics.GetOrRegisterCounter(listenerStatsName("bytes_write", addr), nil),
		downstreamBytesWriteCurrent: metrics.GetOrRegisterGauge(listenerStatsName("bytes_write_current", addr), nil),
	}
}

func listenerStatsName(addr string, statName string) string {
	return fmt.Sprintf("listener.%s.downstream_%s", addr, statName)
}

// ListenerCallbacks
func (al *activeListener) OnAccept(rawc net.Conn, handOffRestoredDestinationConnections bool) {
	arc := newActiveRawConn(rawc, al)
	// TODO: create listener filter chain

	arc.ContinueFilterChain(true)
}

func (al *activeListener) OnNewConnection(conn types.Connection) {
	al.handler.logger.Debugf("New connection %d accepted", conn.Id())

	// start conn loops first
	conn.Start(nil)

	configFactory := al.handler.filterFactory.CreateFilterFactory(al.handler.clusterManager)
	buildFilterChain(conn.FilterManager(), configFactory)

	filterManager := conn.FilterManager()

	if len(filterManager.ListReadFilter()) == 0 &&
		len(filterManager.ListWriteFilters()) == 0 {
		// no filter found, close connection
		conn.Close(types.NoFlush, types.LocalClose)
	} else {
		ac := newActiveConnection(al, conn)

		conn.AddConnectionCallbacks(ac)
		al.connsMux.Lock()
		e := al.conns.PushBack(ac)
		al.connsMux.Unlock()
		ac.element = e

		atomic.AddInt64(&al.handler.numConnections, 1)
	}
}

func (al *activeListener) OnClose() {}

func (al *activeListener) removeConnection(ac *activeConnection) {
	al.connsMux.Lock()
	al.conns.Remove(ac.element)
	al.connsMux.Unlock()

	al.stats.downstreamActive.Update(al.stats.downstreamActive.Value() - 1)
	al.stats.downstreamDestroy.Inc(1)
	atomic.AddInt64(&al.handler.numConnections, -1)

	al.handler.logger.Debugf("close downstream connection, total %d, active %d",
		al.stats.downstreamTotal.Count(), al.stats.downstreamActive.Value())
}

func (al *activeListener) newConnection(rawc net.Conn) {
	conn := network.NewServerConnection(rawc, al.stopChan, al.handler.logger)

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
	ac.conn.AddBytesReadCallback(func(bytesRead uint64) {
		listener.stats.downstreamBytesReadCurrent.Update(int64(bytesRead))

		if bytesRead > 0 {
			listener.stats.downstreamBytesRead.Inc(int64(bytesRead))
		}
	})
	ac.conn.AddBytesSentCallback(func(bytesSent uint64) {
		listener.stats.downstreamBytesWriteCurrent.Update(int64(bytesSent))

		if bytesSent > 0 {
			listener.stats.downstreamBytesWrite.Inc(int64(bytesSent))
		}
	})

	listener.stats.downstreamActive.Update(listener.stats.downstreamActive.Value() + 1)
	listener.stats.downstreamTotal.Inc(1)

	return ac
}

// ConnectionCallbacks
func (ac *activeConnection) OnEvent(event types.ConnectionEvent) {
	if event == types.LocalClose || event == types.RemoteClose ||
		event == types.OnReadErrClose || event == types.OnWriteErrClose {
		ac.listener.removeConnection(ac)
	}
}

// ConnectionCallbacks
func (ac *activeConnection) OnAboveWriteBufferHighWatermark() {}

// ConnectionCallbacks
func (ac *activeConnection) OnBelowWriteBufferLowWatermark() {}
