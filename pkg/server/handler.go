package server

import (
	"container/list"
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
    "gitlab.alipay-inc.com/afe/mosn/pkg/filter/accept/original_dst"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// ConnectionHandler
// ClusterConfigFactoryCb
// ClusterHostFactoryCb
type connHandler struct {
	numConnections int64
	listeners      []*activeListener
	clusterManager types.ClusterManager
	logger         log.Logger
}

func NewHandler(clusterManagerFilter types.ClusterManagerFilter,clMng types.ClusterManager, logger log.Logger) types.ConnectionHandler {
	ch := &connHandler{
		numConnections: 0,
		clusterManager: clMng,
		listeners:      make([]*activeListener, 0),
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

func (ch *connHandler) AddListener(lc *v2.ListenerConfig, networkFiltersFactory types.NetworkFilterChainFactory, streamFiltersFactories []types.StreamFilterChainFactory) {
	//TODO: connection level stop-chan usage confirm
	listenerStopChan := make(chan bool)

	//use default listener path
	if lc.LogPath == "" {
		lc.LogPath = MosnLogBasePath + string(os.PathSeparator) + lc.Name + ".log"
	}

	logger, err := log.NewLogger(lc.LogPath, log.LogLevel(lc.LogLevel))
	if err != nil {
		ch.logger.Fatalf("initialize listener logger failed : %v", err)
	}

	//initialize access log
	var als []types.AccessLog

	for _, alConfig := range lc.AccessLogs {

		//use default listener access log path
		if alConfig.Path == "" {
			alConfig.Path = MosnLogBasePath + string(os.PathSeparator) + lc.Name + "_access.log"
		}

		if al, err := log.NewAccessLog(alConfig.Path, nil, alConfig.Format); err == nil {
			als = append(als, al)
		} else {
			log.StartLogger.Fatalln("initialize listener access logger %s failed : %v", alConfig.Path, err)
		}
	}

	l := network.NewListener(lc, logger)

	al := newActiveListener(l, logger, als, networkFiltersFactory, streamFiltersFactories, ch, listenerStopChan, lc.DisableConnIo)
	l.SetListenerCallbacks(al)

	ch.listeners = append(ch.listeners, al)
}

func (ch *connHandler) StartListener(listenerTag uint64, lctx context.Context) {
	for _, l := range ch.listeners {
		if l.listener.ListenerTag() == listenerTag {
			// TODO: use goruntine pool
			go l.listener.Start(nil)
		}
	}
}

func (ch *connHandler) StartListeners(lctx context.Context) {
	for _, l := range ch.listeners {
		// start goruntine
		go l.listener.Start(nil)
	}
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

	for idx, l := range ch.listeners {
		fd, err := l.listener.ListenerFD()
		if err != nil {
			log.DefaultLogger.Errorf("fail to get listener %s file descriptor: %v", l.listener.Name(), err)
			return nil //stop reconfigure
		}
		fds[idx] = fd
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

// ListenerEventListener
type activeListener struct {
	disableConnIo          bool
	listener               types.Listener
	networkFiltersFactory  types.NetworkFilterChainFactory
	streamFiltersFactories []types.StreamFilterChainFactory
	listenIP			   string
	listenPort             int
	statsNamespace         string
	conns                  *list.List
	connsMux               sync.RWMutex
	handler                *connHandler
	stopChan               chan bool
	stats                  *ListenerStats
	logger                 log.Logger
	accessLogs             []types.AccessLog
}

func newActiveListener(listener types.Listener, logger log.Logger, accessLoggers []types.AccessLog,
	networkFiltersFactory types.NetworkFilterChainFactory, streamFiltersFactories []types.StreamFilterChainFactory,
		handler *connHandler, stopChan chan bool, disableConnIo bool) *activeListener {
	al := &activeListener{
		disableConnIo:          disableConnIo,
		listener:               listener,
		networkFiltersFactory:  networkFiltersFactory,
		streamFiltersFactories: streamFiltersFactories,
		conns:      list.New(),
		handler:    handler,
		stopChan:   stopChan,
		logger:     logger,
		accessLogs: accessLoggers,
	}
	handler.listeners = append(handler.listeners, al)

	listenPort := 0
	var listenIP string
	localAddr := al.listener.Addr().String()

	if temps := strings.Split(localAddr, ":"); len(temps) > 0 {
		listenPort, _ = strconv.Atoi(temps[len(temps)-1])
		listenIP = temps[0]
	}

    al.listenIP = listenIP
	al.listenPort = listenPort
	al.statsNamespace = types.ListenerStatsPrefix + strconv.Itoa(listenPort)
	al.stats = newListenerStats(al.statsNamespace)

	return al
}

// ListenerEventListener
func (al *activeListener) OnAccept(rawc net.Conn, handOffRestoredDestinationConnections bool) {
	arc := newActiveRawConn(rawc, al)
	// TODO: create listener filter chain

	if handOffRestoredDestinationConnections {
		arc.acceptedFilters = append(arc.acceptedFilters, original_dst.NewOriginalDst())
		arc.handOffRestoredDestinationConnections = true
		log.DefaultLogger.Infof("accept restored destination connection from:%s", al.listener.Addr().String())
	}else{
		log.DefaultLogger.Infof("accept connection from:%s", al.listener.Addr().String())
	}

	ctx := context.WithValue(context.Background(), types.ContextKeyListenerPort, al.listenPort)
	ctx = context.WithValue(ctx, types.ContextKeyListenerName, al.listener.Name())
	ctx = context.WithValue(ctx, types.ContextKeyListenerStatsNameSpace, al.statsNamespace)
	ctx = context.WithValue(ctx, types.ContextKeyNetworkFilterChainFactory, al.networkFiltersFactory)
	ctx = context.WithValue(ctx, types.ContextKeyStreamFilterChainFactories, al.streamFiltersFactories)
	ctx = context.WithValue(ctx, types.ContextKeyLogger, al.logger)
	ctx = context.WithValue(ctx, types.ContextKeyAccessLogs, al.accessLogs)

	arc.ContinueFilterChain(true, ctx)
}

func (al *activeListener) OnNewConnection(conn types.Connection, ctx context.Context) {
	// todo: this hack is due to http2 protocol process. golang http2 provides a io loop to read/write stream
	if !al.disableConnIo {
		// start conn loops first
		conn.Start(ctx)
	}

	//Register Proxy's Filter
	configFactory := al.networkFiltersFactory.CreateFilterFactory(al.handler.clusterManager, ctx)
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

		al.logger.Debugf("new downstream connection %d accepted", conn.Id())
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

	al.logger.Debugf("close downstream connection, stats: %s", al.stats.String())
}

func (al *activeListener) newConnection(rawc net.Conn, ctx context.Context) {
	conn := network.NewServerConnection(rawc, al.stopChan, al.logger)
	newCtx := context.WithValue(ctx, types.ContextKeyConnectionId, conn.Id())

	conn.SetBufferLimit(al.listener.PerConnBufferLimitBytes())

	al.OnNewConnection(conn, newCtx)
}

type activeRawConn struct {
	rawc               net.Conn
	originalDstIP	    string
	originalDstPort	    int
	handOffRestoredDestinationConnections bool
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

func (arc *activeRawConn)SetOrigingalAddr(ip string, port int){
	arc.originalDstIP = ip
	arc.originalDstPort = port
	log.DefaultLogger.Infof("conn set origin addr:%s:%d", ip, port)
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
		if arc.handOffRestoredDestinationConnections {
			var _lst, _ls2 *activeListener

			for _, lst := range arc.activeListener.handler.listeners {
				if lst.listenIP == arc.originalDstIP && lst.listenPort == arc.originalDstPort {
					_lst = lst
					break
				}

				if lst.listenPort == arc.originalDstPort && lst.listenIP == "0.0.0.0" {
					_ls2 = lst
				}
			}

			if _lst != nil {
				log.DefaultLogger.Infof("original dst:%s:%d", _lst.listenIP, _lst.listenPort)
				_lst.OnAccept(arc.rawc, false)
			} else if _ls2 != nil {
				log.DefaultLogger.Infof("original dst:%s:%d", _ls2.listenIP, _ls2.listenPort)
				_ls2.OnAccept(arc.rawc, false)
			}

		}else{
			arc.activeListener.newConnection(arc.rawc, ctx)
		}
		
	}
}

func (arc *activeRawConn) Conn() net.Conn {
	return arc.rawc
}

// ConnectionEventListener
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
	ac.conn.AddConnectionEventListener(ac)
	ac.conn.AddBytesReadListener(func(bytesRead uint64) {
		listener.stats.DownstreamBytesReadCurrent().Update(int64(bytesRead))

		if bytesRead > 0 {
			listener.stats.DownstreamBytesRead().Inc(int64(bytesRead))
		}
	})
	ac.conn.AddBytesSentListener(func(bytesSent uint64) {
		listener.stats.DownstreamBytesWriteCurrent().Update(int64(bytesSent))

		if bytesSent > 0 {
			listener.stats.DownstreamBytesWrite().Inc(int64(bytesSent))
		}
	})

	return ac
}

// ConnectionEventListener
func (ac *activeConnection) OnEvent(event types.ConnectionEvent) {
	if event.IsClose() {
		ac.listener.removeConnection(ac)
	}
}

// ConnectionEventListener
func (ac *activeConnection) OnAboveWriteBufferHighWatermark() {}

// ConnectionEventListener
func (ac *activeConnection) OnBelowWriteBufferLowWatermark() {}
