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
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/admin"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/filter/accept/originaldst"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/tls"
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

// GenerateListenerID generates an uuid
// https://tools.ietf.org/html/rfc4122
// crypto.rand use getrandom(2) or /dev/urandom
// It is maybe occur an error due to system error
// panic if an error occurred
func (ch *connHandler) GenerateListenerID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		ch.logger.Fatalf("generate an uuid failed, error: %v", err)
	}
	// see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// AddOrUpdateListener used to add or update listener
// listener name is unique key to represent the listener
// and listener with the same name must have the same configured address
func (ch *connHandler) AddOrUpdateListener(lc *v2.Listener, networkFiltersFactories []types.NetworkFilterChainFactory,
	streamFiltersFactories []types.StreamFilterChainFactory) (types.ListenerEventListener, error) {

	var listenerName string
	if lc.Name == "" {
		listenerName = ch.GenerateListenerID()
		lc.Name = listenerName
	} else {
		listenerName = lc.Name
	}

	var al *activeListener
	if al = ch.findActiveListenerByName(listenerName); al != nil {
		// listener already exist, update the listener

		// a listener with the same name must have the same configured address
		if al.listener.Addr().String() != lc.Addr.String() ||
			al.listener.Addr().Network() != lc.Addr.Network() {
			return nil, fmt.Errorf("error updating listener, listen address and listen name doesn't match")
		}

		equalConfig := reflect.DeepEqual(al.listener.Config(), lc)
		equalNetworkFilter := reflect.DeepEqual(al.networkFiltersFactories, networkFiltersFactories)
		equalStreamFilters := reflect.DeepEqual(al.streamFiltersFactories, streamFiltersFactories)
		// duplicate config does nothing
		if equalConfig && equalNetworkFilter && equalStreamFilters {
			log.DefaultLogger.Debugf("duplicate listener:%s found. no add/update", listenerName)
			return nil, nil
		}

		// update some config, and as Address and Name doesn't change , so need't change *rawl
		al.updatedLabel = true
		if !equalConfig {
			al.disableConnIo = lc.DisableConnIo
			al.listener.SetConfig(lc)
			al.listener.SetePerConnBufferLimitBytes(lc.PerConnBufferLimitBytes)
			al.listener.SetListenerTag(lc.ListenerTag)
			al.listener.SethandOffRestoredDestinationConnections(lc.HandOffRestoredDestinationConnections)
			log.DefaultLogger.Debugf("AddOrUpdateListener: use new listen config = %+v", lc)
		}

		// update network filter
		if !equalNetworkFilter {
			al.networkFiltersFactories = networkFiltersFactories
			log.DefaultLogger.Debugf("AddOrUpdateListener: use new networkFiltersFactories = %+v", networkFiltersFactories)
		}

		// update stream filter
		if !equalStreamFilters {
			al.streamFiltersFactories = streamFiltersFactories
			log.DefaultLogger.Debugf("AddOrUpdateListener: use new streamFiltersFactories = %+v", streamFiltersFactories)
		}
	} else {
		// listener doesn't exist, add the listener
		//TODO: connection level stop-chan usage confirm
		listenerStopChan := make(chan struct{})
		//use default listener path
		if lc.LogPath == "" {
			lc.LogPath = MosnLogBasePath + string(os.PathSeparator) + lc.Name + ".log"
		}

		logger, err := log.NewLogger(lc.LogPath, log.Level(lc.LogLevel))
		if err != nil {
			return nil, fmt.Errorf("initialize listener logger failed : %v", err.Error())
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
				return nil, fmt.Errorf("initialize listener access logger ", alConfig.Path, " failed: ", err.Error())
			}
		}

		l := network.NewListener(lc, logger)

		al, err = newActiveListener(l, lc, logger, als, networkFiltersFactories, streamFiltersFactories, ch, listenerStopChan)
		if err != nil {
			return al, err
		}
		l.SetListenerCallbacks(al)
		ch.listeners = append(ch.listeners, al)
	}

	admin.SetListenerConfig(listenerName, lc)
	return al, nil
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

func (ch *connHandler) FindListenerByName(name string) types.Listener {
	l := ch.findActiveListenerByName(name)

	if l == nil {
		return nil
	}

	return l.listener
}

func (ch *connHandler) RemoveListeners(name string) {
	for i, l := range ch.listeners {
		if l.listener.Name() == name {
			ch.listeners = append(ch.listeners[:i], ch.listeners[i+1:]...)
		}
	}
}

func (ch *connHandler) StopListener(lctx context.Context, name string, close bool) error {
	for _, l := range ch.listeners {
		if l.listener.Name() == name {
			// stop goroutine
			if close {
				return l.listener.Close(lctx)
			}

			return l.listener.Stop()
		}
	}

	return nil
}

func (ch *connHandler) StopListeners(lctx context.Context, close bool) error {
	var errGlobal error
	for _, l := range ch.listeners {
		// stop goroutine
		if close {
			if err := l.listener.Close(lctx); err != nil {
				errGlobal = err
			}
		} else {
			if err := l.listener.Stop(); err != nil {
				errGlobal = err
			}
		}
	}

	return errGlobal
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

func (ch *connHandler) findActiveListenerByName(name string) *activeListener {
	for _, l := range ch.listeners {
		if l.listener != nil {
			if l.listener.Name() == name {
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
	disableConnIo           bool
	listener                types.Listener
	networkFiltersFactories []types.NetworkFilterChainFactory
	streamFiltersFactories  []types.StreamFilterChainFactory
	listenIP                string
	listenPort              int
	statsNamespace          string
	conns                   *list.List
	connsMux                sync.RWMutex
	handler                 *connHandler
	stopChan                chan struct{}
	stats                   *ListenerStats
	logger                  log.Logger
	accessLogs              []types.AccessLog
	updatedLabel            bool
	tlsMng                  types.TLSContextManager
}

func newActiveListener(listener types.Listener, lc *v2.Listener, logger log.Logger, accessLoggers []types.AccessLog,
	networkFiltersFactories []types.NetworkFilterChainFactory, streamFiltersFactories []types.StreamFilterChainFactory,
	handler *connHandler, stopChan chan struct{}) (*activeListener, error) {
	al := &activeListener{
		disableConnIo:           lc.DisableConnIo,
		listener:                listener,
		networkFiltersFactories: networkFiltersFactories,
		streamFiltersFactories:  streamFiltersFactories,
		conns:                   list.New(),
		handler:                 handler,
		stopChan:                stopChan,
		logger:                  logger,
		accessLogs:              accessLoggers,
		updatedLabel:            false,
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

	mgr, err := tls.NewTLSServerContextManager(lc, listener, logger)
	if err != nil {
		logger.Errorf("create tls context manager failed, %v", err)
		return nil, err
	}
	al.tlsMng = mgr

	return al, nil
}

// ListenerEventListener
func (al *activeListener) OnAccept(rawc net.Conn, handOffRestoredDestinationConnections bool, oriRemoteAddr net.Addr, ch chan types.Connection, buf []byte) {
	var rawf *os.File

	// only store fd and tls conn handshake in final working listener
	if !handOffRestoredDestinationConnections {
		if !al.disableConnIo && network.UseNetpollMode {
			// store fd for further usage
			if tc, ok := rawc.(*net.TCPConn); ok {
				rawf, _ = tc.File()
			}
		}
		if al.tlsMng != nil && al.tlsMng.Enabled() {
			rawc = al.tlsMng.Conn(rawc)
		}
	}

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
	ctx = context.WithValue(ctx, types.ContextKeyNetworkFilterChainFactories, al.networkFiltersFactories)
	ctx = context.WithValue(ctx, types.ContextKeyStreamFilterChainFactories, al.streamFiltersFactories)
	ctx = context.WithValue(ctx, types.ContextKeyLogger, al.logger)
	ctx = context.WithValue(ctx, types.ContextKeyAccessLogs, al.accessLogs)
	if rawf != nil {
		ctx = context.WithValue(ctx, types.ContextKeyConnectionFd, rawf)
	}
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
	filterManager := conn.FilterManager()
	for _, nfcf := range al.networkFiltersFactories {
		nfcf.CreateFilterChain(ctx, al.handler.clusterManager, filterManager)
	}
	filterManager.InitializeReadFilters()

	if len(filterManager.ListReadFilter()) == 0 &&
		len(filterManager.ListWriteFilters()) == 0 {
		// no filter found, close connection
		conn.Close(types.NoFlush, types.LocalClose)
		return
	}
	ac := newActiveConnection(al, conn)

	al.connsMux.Lock()
	e := al.conns.PushBack(ac)
	al.connsMux.Unlock()
	ac.element = e

	al.stats.DownstreamConnectionActive().Inc(1)
	al.stats.DownstreamConnectionTotal().Inc(1)
	atomic.AddInt64(&al.handler.numConnections, 1)

	al.logger.Debugf("new downstream connection %d accepted", conn.ID())

	// todo: this hack is due to http2 protocol process. golang http2 provides a io loop to read/write stream
	if !al.disableConnIo {
		// start conn loops first
		conn.Start(ctx)
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
	rawf                                  *os.File
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
