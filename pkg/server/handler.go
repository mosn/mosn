package server

import (
	"sync"
	"container/list"
	"net"
	"fmt"
	"strconv"
	"strings"
	"context"
	"sync/atomic"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

// ConnectionHandler
// ClusterConfigFactoryCb
// ClusterHostFactoryCb
type connHandler struct {
	disableConnIo          bool
	numConnections         int64
	listeners              []*activeListener
	clusterManager         types.ClusterManager
	networkFiltersFactory  NetworkFilterChainFactory
	streamFiltersFactories []types.StreamFilterChainFactory
	logger                 log.Logger
}

func NewHandler(networkFiltersFactory NetworkFilterChainFactory, streamFiltersFactories []types.StreamFilterChainFactory,
	clusterManagerFilter ClusterManagerFilter, logger log.Logger, DisableConnIo bool) types.ConnectionHandler {
	ch := &connHandler{
		disableConnIo:         DisableConnIo,
		numConnections:        0,
		clusterManager:        cluster.NewClusterManager(nil),
		listeners:             make([]*activeListener, 0),
		networkFiltersFactory: networkFiltersFactory,
		streamFiltersFactories:  streamFiltersFactories,
		logger:                logger,
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
			l.listener.Stop()
		}
	}
}

func (ch *connHandler) StopListeners(lctx context.Context) {
	for _, l := range ch.listeners {
		// stop goruntine
		l.listener.Stop()
	}
}

func (ch *connHandler) ListListenersFD(lctx context.Context) []uintptr {
	fds := make([]uintptr, len(ch.listeners))

	for _, l := range ch.listeners {
		fd, err := l.listener.ListenerFD()
		if err != nil {
			log.DefaultLogger.Fatalln("fail to get listener", l.listener.Name() ," file descriptor:", err)
		}
		fds = append(fds, fd)
	}
	return fds
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
	listener       types.Listener
	listenPort     int
	statsNamespace string
	conns          *list.List
	connsMux       sync.RWMutex
	handler        *connHandler
	stopChan       chan bool
	stats          *ListenerStats
}

func newActiveListener(listener types.Listener, handler *connHandler, stopChan chan bool) *activeListener {
	al := &activeListener{
		listener: listener,
		conns:    list.New(),
		handler:  handler,
		stopChan: stopChan,
	}

	listenPort := 0
	localAddr := al.listener.Addr().String()

	if temps := strings.Split(localAddr, ":"); len(temps) > 0 {
		listenPort, _ = strconv.Atoi(temps[len(temps)-1])
	}

	al.listenPort = listenPort
	al.statsNamespace = fmt.Sprintf(types.ListenerStatsPrefix, listenPort)
	al.stats = newListenerStats(al.statsNamespace)

	return al
}

// ListenerCallbacks
func (al *activeListener) OnAccept(rawc net.Conn, handOffRestoredDestinationConnections bool) {
	arc := newActiveRawConn(rawc, al)
	// TODO: create listener filter chain

	ctx := context.WithValue(context.Background(), types.ContextKeyListenerPort, al.listenPort)
	ctx = context.WithValue(ctx, types.ContextKeyListenerName, al.listener.Name())
	ctx = context.WithValue(ctx, types.ContextKeyListenerStatsNameSpace, al.statsNamespace)
	ctx = context.WithValue(ctx, types.ContextKeyNetworkFilterChainFactory, al.handler.networkFiltersFactory)
	ctx = context.WithValue(ctx, types.ContextKeyStreamFilterChainFactories, al.handler.streamFiltersFactories)

	arc.ContinueFilterChain(true, ctx)
}

func (al *activeListener) OnNewConnection(conn types.Connection, ctx context.Context) {
	// todo: this hack is due to http2 protocol process. golang http2 provides a io loop to read/write stream
	if !al.handler.disableConnIo {
		// start conn loops first
		conn.Start(ctx)
	}

	configFactory := al.handler.networkFiltersFactory.CreateFilterFactory(al.handler.clusterManager, ctx)
	buildFilterChain(conn.FilterManager(), configFactory)

	filterManager := conn.FilterManager()

	if len(filterManager.ListReadFilter()) == 0 &&
		len(filterManager.ListWriteFilters()) == 0 {
		// no filter found, close connection
		conn.Close(types.NoFlush, types.LocalClose)
	} else {
		ac := newActiveConnection(al, conn)

		al.connsMux.Lock()
		e := al.conns.PushBack(ac)
		al.connsMux.Unlock()
		ac.element = e

		al.stats.DownstreamConnectionActive().Inc(1)
		al.stats.DownstreamConnectionTotal().Inc(1)
		atomic.AddInt64(&al.handler.numConnections, 1)

		al.handler.logger.Debugf("new downstream connection %d accepted", conn.Id())
	}
}

func (al *activeListener) OnClose() {}

func (al *activeListener) removeConnection(ac *activeConnection) {
	al.connsMux.Lock()
	al.conns.Remove(ac.element)
	al.connsMux.Unlock()

	al.stats.DownstreamConnectionActive().Dec(1)
	al.stats.DownstreamConnectionDestroy().Inc(1)
	atomic.AddInt64(&al.handler.numConnections, -1)

	al.handler.logger.Debugf("close downstream connection, stats: %s", al.stats.String())
}

func (al *activeListener) newConnection(rawc net.Conn, ctx context.Context) {
	conn := network.NewServerConnection(rawc, al.stopChan, al.handler.logger)
	newCtx := context.WithValue(ctx, types.ContextKeyConnectionId, conn.Id())

	conn.SetBufferLimit(al.listener.PerConnBufferLimitBytes())

	al.OnNewConnection(conn, newCtx)
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

func (arc *activeRawConn) ContinueFilterChain(success bool, ctx context.Context) {
	if success {
		for ; arc.accptedFilterIndex < len(arc.acceptedFilters); arc.accptedFilterIndex++ {
			filterStatus := arc.acceptedFilters[arc.accptedFilterIndex].OnAccept(arc)

			if filterStatus == types.StopIteration {
				return
			}
		}

		// TODO: handle hand_off_restored_destination_connections logic

		arc.activeListener.newConnection(arc.rawc, ctx)
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
		listener.stats.DownstreamBytesReadCurrent().Update(int64(bytesRead))

		if bytesRead > 0 {
			listener.stats.DownstreamBytesRead().Inc(int64(bytesRead))
		}
	})
	ac.conn.AddBytesSentCallback(func(bytesSent uint64) {
		listener.stats.DownstreamBytesWriteCurrent().Update(int64(bytesSent))

		if bytesSent > 0 {
			listener.stats.DownstreamBytesWrite().Inc(int64(bytesSent))
		}
	})

	return ac
}

// ConnectionCallbacks
func (ac *activeConnection) OnEvent(event types.ConnectionEvent) {
	if event.IsClose() {
		ac.listener.removeConnection(ac)
	}
}

// ConnectionCallbacks
func (ac *activeConnection) OnAboveWriteBufferHighWatermark() {}

// ConnectionCallbacks
func (ac *activeConnection) OnBelowWriteBufferLowWatermark() {}
