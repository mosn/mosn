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

package server

import (
	"container/list"
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"fmt"
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/filter/accept/originaldst"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/types"
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

// NewHandler
// create types.ConnectionHandler's implement connHandler
// with cluster manager and logger
func NewHandler(clusterManagerFilter types.ClusterManagerFilter, clMng types.ClusterManager,
	logger log.Logger) types.ConnectionHandler {
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
		if !ch.clusterManager.AddOrUpdatePrimaryCluster(cluster) {
			return fmt.Errorf("UpdateClusterConfig: AddOrUpdatePrimaryCluster failure, cluster name = %s", cluster.Name)
		}
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

func (ch *connHandler) AddListener(lc *v2.ListenerConfig, networkFiltersFactory types.NetworkFilterChainFactory,
	streamFiltersFactories []types.StreamFilterChainFactory) types.ListenerEventListener {
	//TODO: connection level stop-chan usage confirm
	listenerStopChan := make(chan struct{})

	//use default listener path
	if lc.LogPath == "" {
		lc.LogPath = MosnLogBasePath + string(os.PathSeparator) + lc.Name + ".log"
	}

	logger, err := log.NewLogger(lc.LogPath, log.Level(lc.LogLevel))
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
			log.StartLogger.Fatalln("initialize listener access logger ", alConfig.Path, " failed: ", err)
		}
	}

	l := network.NewListener(lc, logger)

	al := newActiveListener(l, logger, als, networkFiltersFactory, streamFiltersFactories, ch, listenerStopChan, lc.DisableConnIo)
	l.SetListenerCallbacks(al)

	ch.listeners = append(ch.listeners, al)

	return al
}

func (ch *connHandler) StartListener(lctx context.Context, listenerTag uint64) {
	for _, l := range ch.listeners {
		if l.listener.ListenerTag() == listenerTag {
			// TODO: use goroutine pool
			go l.listener.Start(nil)
		}
	}
}

func (ch *connHandler) StartListeners(lctx context.Context) {
	for _, l := range ch.listeners {
		// start goroutine
		go l.listener.Start(nil)
	}
}

func (ch *connHandler) FindListenerByAddress(addr net.Addr) types.Listener {
	l := ch.findActiveListenerByAddress(addr)

	if l == nil {
		return nil
	}

	return l.listener
}

func (ch *connHandler) RemoveListeners(listenerTag uint64) {
	for i, l := range ch.listeners {
		if l.listener.ListenerTag() == listenerTag {
			ch.listeners = append(ch.listeners[:i], ch.listeners[i+1:]...)
		}
	}
}

func (ch *connHandler) StopListener(lctx context.Context, listenerTag uint64) {
	for _, l := range ch.listeners {
		if l.listener.ListenerTag() == listenerTag {
			// stop goroutine
			l.listener.Stop()
		}
	}
}

func (ch *connHandler) StopListeners(lctx context.Context, close bool) {
	for _, l := range ch.listeners {
		// stop goroutine
		if close {
			l.listener.Close(lctx)
		} else {
			l.listener.Stop()
		}
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

func (ch *connHandler) StopConnection() {
	for _, l := range ch.listeners {
		close(l.stopChan)
	}
}

// ListenerEventListener
type activeListener struct {
	disableConnIo          bool
	listener               types.Listener
	networkFiltersFactory  types.NetworkFilterChainFactory
	streamFiltersFactories []types.StreamFilterChainFactory
	listenIP               string
	listenPort             int
	statsNamespace         string
	conns                  *list.List
	connsMux               sync.RWMutex
	handler                *connHandler
	stopChan               chan struct{}
	stats                  *ListenerStats
	logger                 log.Logger
	accessLogs             []types.AccessLog
}

func newActiveListener(listener types.Listener, logger log.Logger, accessLoggers []types.AccessLog,
	networkFiltersFactory types.NetworkFilterChainFactory, streamFiltersFactories []types.StreamFilterChainFactory,
	handler *connHandler, stopChan chan struct{}, disableConnIo bool) *activeListener {
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
func (al *activeListener) OnAccept(rawc net.Conn, handOffRestoredDestinationConnections bool, oriRemoteAddr net.Addr, ch chan types.Connection, buf []byte) {
	arc := newActiveRawConn(rawc, al)
	// TODO: create listener filter chain

	if handOffRestoredDestinationConnections {
		arc.acceptedFilters = append(arc.acceptedFilters, originaldst.NewOriginalDst())
		arc.handOffRestoredDestinationConnections = true
		log.DefaultLogger.Infof("accept restored destination connection from:%s", al.listener.Addr().String())
	} else {
		log.DefaultLogger.Infof("accept connection from:%s", al.listener.Addr().String())
	}

	ctx := context.WithValue(context.Background(), types.ContextKeyListenerPort, al.listenPort)
	ctx = context.WithValue(ctx, types.ContextKeyListenerName, al.listener.Name())
	ctx = context.WithValue(ctx, types.ContextKeyListenerStatsNameSpace, al.statsNamespace)
	ctx = context.WithValue(ctx, types.ContextKeyNetworkFilterChainFactory, al.networkFiltersFactory)
	ctx = context.WithValue(ctx, types.ContextKeyStreamFilterChainFactories, al.streamFiltersFactories)
	ctx = context.WithValue(ctx, types.ContextKeyLogger, al.logger)
	ctx = context.WithValue(ctx, types.ContextKeyAccessLogs, al.accessLogs)
	if ch != nil {
		ctx = context.WithValue(ctx, types.ContextKeyAcceptChan, ch)
		ctx = context.WithValue(ctx, types.ContextKeyAcceptBuffer, buf)
	}
	if oriRemoteAddr != nil {
		ctx = context.WithValue(ctx, types.ContextOriRemoteAddr, oriRemoteAddr)
	}
	arc.ContinueFilterChain(ctx, true)
}

func (al *activeListener) OnNewConnection(ctx context.Context, conn types.Connection) {
	//Register Proxy's Filter
	configFactory := al.networkFiltersFactory.CreateFilterFactory(ctx, al.handler.clusterManager)
	buildFilterChain(conn.FilterManager(), configFactory)

	// todo: this hack is due to http2 protocol process. golang http2 provides a io loop to read/write stream
	if !al.disableConnIo {
		// start conn loops first
		conn.Start(ctx)
	}

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

		al.logger.Debugf("new downstream connection %d accepted", conn.ID())
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

func (al *activeListener) newConnection(ctx context.Context, rawc net.Conn) {
	conn := network.NewServerConnection(ctx, rawc, al.stopChan, al.logger)
	oriRemoteAddr := ctx.Value(types.ContextOriRemoteAddr)
	if oriRemoteAddr != nil {
		conn.SetRemoteAddr(oriRemoteAddr.(net.Addr))
	}
	newCtx := context.WithValue(ctx, types.ContextKeyConnectionID, conn.ID())

	conn.SetBufferLimit(al.listener.PerConnBufferLimitBytes())

	al.OnNewConnection(newCtx, conn)
}

type activeRawConn struct {
	rawc                                  net.Conn
	originalDstIP                         string
	originalDstPort                       int
	oriRemoteAddr                         net.Addr
	handOffRestoredDestinationConnections bool
	rawcElement                           *list.Element
	activeListener                        *activeListener
	acceptedFilters                       []types.ListenerFilter
	acceptedFilterIndex                   int
}

func newActiveRawConn(rawc net.Conn, activeListener *activeListener) *activeRawConn {
	return &activeRawConn{
		rawc:           rawc,
		activeListener: activeListener,
	}
}

func (arc *activeRawConn) SetOriginalAddr(ip string, port int) {
	arc.originalDstIP = ip
	arc.originalDstPort = port
	arc.oriRemoteAddr, _ = net.ResolveTCPAddr("", ip+":"+strconv.Itoa(port))
	log.DefaultLogger.Infof("conn set origin addr:%s:%d", ip, port)
}

func (arc *activeRawConn) HandOffRestoredDestinationConnectionsHandler() {
	var listener, localListener *activeListener

	for _, lst := range arc.activeListener.handler.listeners {
		if lst.listenIP == arc.originalDstIP && lst.listenPort == arc.originalDstPort {
			listener = lst
			break
		}

		if lst.listenPort == arc.originalDstPort && lst.listenIP == "0.0.0.0" {
			localListener = lst
		}
	}

	if listener != nil {
		log.DefaultLogger.Infof("original dst:%s:%d", listener.listenIP, listener.listenPort)
		listener.OnAccept(arc.rawc, false, arc.oriRemoteAddr, nil, nil)
	}
	if localListener != nil {
		log.DefaultLogger.Infof("original dst:%s:%d", localListener.listenIP, localListener.listenPort)
		localListener.OnAccept(arc.rawc, false, arc.oriRemoteAddr, nil, nil)
	}
}

func (arc *activeRawConn) ContinueFilterChain(ctx context.Context, success bool) {

	if !success {
		return
	}

	for ; arc.acceptedFilterIndex < len(arc.acceptedFilters); arc.acceptedFilterIndex++ {
		filterStatus := arc.acceptedFilters[arc.acceptedFilterIndex].OnAccept(arc)
		if filterStatus == types.StopIteration {
			return
		}
	}

	// TODO: handle hand_off_restored_destination_connections logic
	if arc.handOffRestoredDestinationConnections {
		arc.HandOffRestoredDestinationConnectionsHandler()
	} else {
		arc.activeListener.newConnection(ctx, arc.rawc)
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
