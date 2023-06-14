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
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"mosn.io/api"
	admin "mosn.io/mosn/pkg/admin/store"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/filter/listener/originaldst"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
	"mosn.io/pkg/variable"
)

// ConnectionHandler
// ClusterConfigFactoryCb
// ClusterHostFactoryCb
type connHandler struct {
	numConnections int64
	listeners      []*activeListener
	clusterManager types.ClusterManager
}

// NewHandler
// create types.ConnectionHandler's implement connHandler
// with cluster manager and logger
func NewHandler(clusterManagerFilter types.ClusterManagerFilter, clMng types.ClusterManager) types.ConnectionHandler {
	ch := &connHandler{
		numConnections: 0,
		clusterManager: clMng,
		listeners:      make([]*activeListener, 0),
	}

	clusterManagerFilter.OnCreated(ch, ch)

	return ch
}

// ClusterConfigFactoryCb
func (ch *connHandler) UpdateClusterConfig(clusters []v2.Cluster) error {

	for _, cluster := range clusters {
		if err := ch.clusterManager.AddOrUpdatePrimaryCluster(cluster); err != nil {
			return fmt.Errorf("UpdateClusterConfig: AddOrUpdatePrimaryCluster failure, cluster name = %s", cluster.Name)
		}
	}

	// TODO: remove cluster

	return nil
}

// ClusterHostFactoryCb
func (ch *connHandler) UpdateClusterHost(cluster string, hosts []v2.Host) error {
	return ch.clusterManager.UpdateClusterHosts(cluster, hosts)
}

// ConnectionHandler
func (ch *connHandler) NumConnections() uint64 {
	return uint64(atomic.LoadInt64(&ch.numConnections))
}

// AddOrUpdateListener used to add or update listener
// listener name is unique key to represent the listener
// and listener with the same name must have the same configured address
func (ch *connHandler) AddOrUpdateListener(lc *v2.Listener) (types.ListenerEventListener, error) {

	var listenerName string
	if lc.Name == "" {
		listenerName = lc.Addr.String() //utils.GenerateUUID()
		lc.Name = listenerName
	} else {
		listenerName = lc.Name
	}
	// currently, we just support one filter chain
	if len(lc.FilterChains) != 1 {
		return nil, errors.New("error updating listener, listener have filter chains count is not 1")
	}
	// set listener filter , network filter and stream filter
	var listenerFiltersFactories []api.ListenerFilterChainFactory
	var networkFiltersFactories []api.NetworkFilterChainFactory
	listenerFiltersFactories = configmanager.AddOrUpdateListenerFilterFactories(listenerName, lc.ListenerFilters)
	streamfilter.GetStreamFilterManager().AddOrUpdateStreamFilterConfig(listenerName, lc.StreamFilters)
	networkFiltersFactories = configmanager.AddOrUpdateNetworkFilterFactories(listenerName, lc)

	var al *activeListener
	if al = ch.findActiveListenerByName(listenerName); al != nil {
		// listener already exist, update the listener

		// a listener with the same name must have the same configured address
		if al.listener.Addr().String() != lc.Addr.String() ||
			al.listener.Addr().Network() != lc.Addr.Network() {
			return nil, errors.New("error updating listener, listen address and listen name doesn't match")
		}

		rawConfig := al.listener.Config()
		// FIXME: update log level need the pkg/logger support.

		al.listenerFiltersFactories = listenerFiltersFactories
		rawConfig.ListenerFilters = lc.ListenerFilters
		al.networkFiltersFactories = networkFiltersFactories
		rawConfig.FilterChains[0].FilterChainMatch = lc.FilterChains[0].FilterChainMatch
		rawConfig.FilterChains[0].Filters = lc.FilterChains[0].Filters

		rawConfig.StreamFilters = lc.StreamFilters

		// tls update only take effects on new connections
		// config changed
		rawConfig.FilterChains[0].TLSContexts = lc.FilterChains[0].TLSContexts
		rawConfig.FilterChains[0].TLSConfig = lc.FilterChains[0].TLSConfig
		rawConfig.FilterChains[0].TLSConfigs = lc.FilterChains[0].TLSConfigs
		rawConfig.Inspector = lc.Inspector
		mgr, err := mtls.NewTLSServerContextManager(rawConfig)
		if err != nil {
			log.DefaultLogger.Errorf("[server] [conn handler] [update listener] create tls context manager failed, %v", err)
			return nil, err
		}
		// object changed
		al.tlsMng = mgr
		// some simle config update
		rawConfig.PerConnBufferLimitBytes = lc.PerConnBufferLimitBytes
		al.listener.SetPerConnBufferLimitBytes(lc.PerConnBufferLimitBytes)
		rawConfig.ListenerTag = lc.ListenerTag
		al.listener.SetListenerTag(lc.ListenerTag)
		rawConfig.OriginalDst = lc.OriginalDst
		al.listener.SetOriginalDstType(lc.OriginalDst)
		al.idleTimeout = lc.ConnectionIdleTimeout
		al.listener.SetConfig(rawConfig)

		// set update label to true, do not start the listener again
		al.updatedLabel = true
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[server] [conn handler] [update listener] update listener: %s", lc.AddrConfig)
		}

	} else {
		// listener doesn't exist, add the listener
		//TODO: connection level stop-chan usage confirm
		listenerStopChan := make(chan struct{})

		//initialize access log
		var als []api.AccessLog

		for _, alConfig := range lc.AccessLogs {

			//use default listener access log path
			if alConfig.Path == "" {
				alConfig.Path = types.MosnLogBasePath + string(os.PathSeparator) + lc.Name + "_access.log"
			}

			if al, err := log.NewAccessLog(alConfig.Path, alConfig.Format); err == nil {
				als = append(als, al)
			} else {
				return nil, fmt.Errorf("initialize listener access logger %s failed: %v", alConfig.Path, err.Error())
			}
		}

		l := network.GetListenerFactory()(lc)

		var err error
		al, err = newActiveListener(l, lc, als, listenerFiltersFactories, networkFiltersFactories, ch, listenerStopChan)
		if err != nil {
			return al, err
		}
		l.SetListenerCallbacks(al)
		ch.listeners = append(ch.listeners, al)
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[server] [conn handler] [add listener] add listener: %s", lc.Addr.String())
		}

	}

	configmanager.SetListenerConfig(*al.listener.Config())
	return al, nil
}

func (ch *connHandler) StartListener(lctx context.Context, listenerTag uint64) {
	for _, l := range ch.listeners {
		if l.listener.ListenerTag() == listenerTag {
			// TODO: use goroutine pool
			l.GoStart(lctx)
		}
	}
}

func (ch *connHandler) StartListeners(lctx context.Context) {
	for _, l := range ch.listeners {
		// start goroutine
		l.GoStart(lctx)
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
			log.DefaultLogger.Infof("[server] [conn handler] remove listener name: %s", name)
			ch.listeners = append(ch.listeners[:i], ch.listeners[i+1:]...)
		}
	}
}

func (ch *connHandler) GracefulStopListener(lctx context.Context, name string) error {
	var errGlobal error
	for _, l := range ch.listeners {
		if l.listener.Name() == name {
			log.DefaultLogger.Infof("graceful closing listener %v", name)
			if err := l.listener.Shutdown(lctx); err != nil {
				log.DefaultLogger.Errorf("failed to shutdown listener %v: %v", l.listener.Name(), err)
				errGlobal = err
			}
		}
	}
	return errGlobal
}

func (ch *connHandler) GracefulCloseListener(lctx context.Context, name string) error {
	var errGlobal error
	for _, l := range ch.listeners {
		if l.listener.Name() == name {
			log.DefaultLogger.Infof("graceful closing listener %v", name)
			if err := l.listener.Shutdown(lctx); err != nil {
				log.DefaultLogger.Errorf("failed to shutdown listener %v: %v", l.listener.Name(), err)
				errGlobal = err
			}
			if err := l.listener.Close(lctx); err != nil {
				log.DefaultLogger.Errorf("failed to close listener %v: %v", l.listener.Name(), err)
				errGlobal = err
			}
		}
	}
	return errGlobal
}

// GracefulStopListeners stop accept new connections
// and graceful close all the existing connections.
func (ch *connHandler) GracefulStopListeners(lctx context.Context) error {
	var failed bool
	listeners := ch.listeners
	wg := sync.WaitGroup{}
	wg.Add(len(listeners))
	for _, l := range listeners {
		al := l
		log.DefaultLogger.Infof("graceful shutdown listener %v", al.listener.Name())
		// Shutdown listener in parallel
		utils.GoWithRecover(func() {
			defer wg.Done()
			if err := al.listener.Shutdown(lctx); err != nil {
				log.DefaultLogger.Errorf("failed to shutdown listener %v: %v", al.listener.Name(), err)
				failed = true
			}
		}, nil)
	}
	wg.Wait()

	if failed {
		return errors.New("failed to shutdown listeners")
	}
	return nil
}

// CloseListeners close listeners immediately
func (ch *connHandler) CloseListeners() {
	for _, l := range ch.listeners {
		l.listener.Close(nil)
	}
}

func (ch *connHandler) ListListenersFile(lctx context.Context) []*os.File {
	files := make([]*os.File, 0)
	for _, l := range ch.listeners {
		if !l.listener.IsBindToPort() {
			continue
		}
		file, err := l.listener.ListenerFile()
		if err != nil {
			log.DefaultLogger.Alertf("listener.list", "[server] [conn handler] fail to get listener %s file descriptor: %v", l.listener.Name(), err)
			return nil //stop reconfigure
		}
		files = append(files, file)
	}
	return files
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
		if l.listener != nil && l.listener.Name() == name {
			return l
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
	listener                 types.Listener
	listenerFiltersFactories []api.ListenerFilterChainFactory
	networkFiltersFactories  []api.NetworkFilterChainFactory
	listenIP                 string
	listenPort               int
	defaultReadBufferSize    int
	conns                    *utils.SyncList
	handler                  *connHandler
	stopChan                 chan struct{}
	stats                    *listenerStats
	accessLogs               []api.AccessLog
	updatedLabel             bool
	idleTimeout              *api.DurationConfig
	tlsMng                   types.TLSContextManager
}

func newActiveListener(listener types.Listener, lc *v2.Listener, accessLoggers []api.AccessLog,
	listenerFiltersFactories []api.ListenerFilterChainFactory,
	networkFiltersFactories []api.NetworkFilterChainFactory,
	handler *connHandler, stopChan chan struct{}) (*activeListener, error) {
	al := &activeListener{
		listener:                 listener,
		defaultReadBufferSize:    lc.DefaultReadBufferSize,
		conns:                    utils.NewSyncList(),
		handler:                  handler,
		stopChan:                 stopChan,
		accessLogs:               accessLoggers,
		updatedLabel:             false,
		idleTimeout:              lc.ConnectionIdleTimeout,
		networkFiltersFactories:  networkFiltersFactories,
		listenerFiltersFactories: listenerFiltersFactories,
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
	al.stats = newListenerStats(al.listener.Name())

	mgr, err := mtls.NewTLSServerContextManager(lc)
	if err != nil {
		log.DefaultLogger.Errorf("[server] [new listener] create tls context manager failed, %v", err)
		return nil, err
	}
	al.tlsMng = mgr

	return al, nil
}

func (al *activeListener) GoStart(lctx context.Context) {
	utils.GoWithRecover(func() {
		al.listener.Start(lctx, false)
	}, func(r interface{}) {
		// TODO: add a times limit?
		log.DefaultLogger.Alertf("listener.start", "[network] [listener start] old listener panic")
		al.GoStart(lctx)
	})
}

// ListenerEventListener
func (al *activeListener) OnAccept(rawc net.Conn, useOriginalDst bool, oriRemoteAddr net.Addr, ch chan api.Connection, buf []byte, listeners []api.ConnectionEventListener) {
	var rawf *os.File

	// only store fd and tls conn handshake in final working listener
	if !useOriginalDst {
		if network.UseNetpollMode {
			// store fd for further usage

			switch rawc.LocalAddr().Network() {
			case "udp":
				if tc, ok := rawc.(*net.UDPConn); ok {
					rawf, _ = tc.File()
				}
			case "unix":
				if tc, ok := rawc.(*net.UnixConn); ok {
					rawf, _ = tc.File()
				}
			default:
				if tc, ok := rawc.(*net.TCPConn); ok {
					rawf, _ = tc.File()
				}
			}
		}
		// if ch is not nil, the conn has been initialized in func transferNewConn
		if al.tlsMng != nil && ch == nil {
			conn, err := al.tlsMng.Conn(rawc)
			if err != nil {
				if log.DefaultLogger.GetLogLevel() >= log.INFO {
					log.DefaultLogger.Infof("[server] [listener] accept connection failed, error: %v", err)
				}
				rawc.Close()
				return
			}
			rawc = conn
		}
	}

	arc := newActiveRawConn(rawc, al)

	// listener filter chain.
	for _, lfcf := range al.listenerFiltersFactories {
		arc.acceptedFilters = append(arc.acceptedFilters, lfcf)
	}

	if useOriginalDst {
		arc.useOriginalDst = true
		// TODO remove it when Istio deprecate UseOriginalDst.
		arc.acceptedFilters = append(arc.acceptedFilters, originaldst.NewOriginalDst(arc.activeListener.listener.GetOriginalDstType()))
	}

	// connection context support variables too
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableListenerPort, al.listenPort)
	_ = variable.Set(ctx, types.VariableListenerType, al.listener.Config().Type)
	_ = variable.Set(ctx, types.VariableListenerName, al.listener.Name())
	_ = variable.Set(ctx, types.VariableConnDefaultReadBufferSize, al.defaultReadBufferSize)
	_ = variable.Set(ctx, types.VariableNetworkFilterChainFactories, al.networkFiltersFactories)
	_ = variable.Set(ctx, types.VariableAccessLogs, al.accessLogs)
	if rawf != nil {
		_ = variable.Set(ctx, types.VariableConnectionFd, rawf)
	}
	if ch != nil {
		_ = variable.Set(ctx, types.VariableAcceptChan, ch)
		_ = variable.Set(ctx, types.VariableAcceptBuffer, buf)
	}
	if rawc.LocalAddr().Network() == "udp" {
		_ = variable.Set(ctx, types.VariableAcceptBuffer, buf)
	}
	if oriRemoteAddr != nil {
		_ = variable.Set(ctx, types.VariableOriRemoteAddr, oriRemoteAddr)
	}

	if len(listeners) != 0 {
		_ = variable.Set(ctx, types.VariableConnectionEventListeners, listeners)
	}

	arc.ctx = ctx

	arc.ContinueFilterChain(ctx, true)
}

func (al *activeListener) OnNewConnection(ctx context.Context, conn api.Connection) {
	//Register Proxy's Filter
	filterManager := conn.FilterManager()
	for _, nfcf := range al.networkFiltersFactories {
		nfcf.CreateFilterChain(ctx, filterManager)
	}

	ac := newActiveConnection(al, conn)

	if conn.LocalAddr().Network() == "udp" {
		network.SetUDPProxyMap(network.GetProxyMapKey(conn.LocalAddr().String(), conn.RemoteAddr().String()), conn)
	}

	e := al.conns.PushBack(ac)
	ac.element = e

	atomic.AddInt64(&al.handler.numConnections, 1)

	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[server] [listener] accept connection from %s, condId= %d, remote addr:%s", al.listener.Addr().String(), conn.ID(), conn.RemoteAddr().String())
	}

	filterManager.InitializeReadFilters()

	if len(filterManager.ListReadFilter()) == 0 &&
		len(filterManager.ListWriteFilters()) == 0 {
		// no filter found, close connection
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[server] [listener] accept connection from %s, condId= %d, remote addr:%s, but no filters found, closing it", al.listener.Addr().String(), conn.ID(), conn.RemoteAddr().String())
		}
		conn.Close(api.NoFlush, api.LocalClose)
		return
	}

	// start conn loops first
	conn.Start(ctx)
}

func (al *activeListener) activeStreamSize() int {
	listenerName := al.listener.Name()
	s := metrics.NewListenerStats(listenerName)

	return int(s.Counter(metrics.DownstreamRequestActive).Count())
}

var (
	// drain time, default 15 seconds
	drainTime = time.Second * 15
)

func SetDrainTime(time time.Duration) {
	drainTime = time
}

// OnShutdown graceful stop the existing connection and wait all connections to be closed
func (al *activeListener) OnShutdown() {
	utils.GoWithRecover(func() {
		al.conns.VisitSafe(func(v interface{}) {
			conn := v.(*activeConnection).conn
			// TODO 1: conn.OnConnectionEvent may be blocked on connection.Write, need a proper way to not block too long.
			// TODO 2: shutdown connections gradual, it's useful when there are many connections.
			conn.OnConnectionEvent(api.OnShutdown)
		})
	}, nil)

	al.waitConnectionsClose(drainTime)
}

func (al *activeListener) OnClose() {
}

// waitConnectionsClose wait all connections to be closed, wait maxWaitMilliseconds at most.
func (al *activeListener) waitConnectionsClose(maxWaitTime time.Duration) {
	// if there is any stream being processed and without timeout,
	// we try to wait for processing to complete, or wait for a timeout.
	current := time.Now()
	remainStream, waited := al.activeStreamSize(), time.Since(current)
	for ; remainStream > 0 && waited <= maxWaitTime; remainStream, waited =
		al.activeStreamSize(), time.Since(current) {
		// sleep 10ms
		time.Sleep(10 * time.Millisecond)
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[activeListener] listener %s waiting connections close, remaining stream count %d, waited time %dms",
				al.listener.Name(), remainStream, Milliseconds(waited))
		}
	}

	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("[activeListener] listener %s wait connections close complete, remaining stream count %d, waited time %dms",
			al.listener.Name(), remainStream, Milliseconds(waited))
	}
}

// PreStopHook used for graceful stop
func (al *activeListener) PreStopHook(ctx context.Context) func() error {
	// before allowing you to stop listener,
	// check that the preconditions are met.
	// for example: whether all request queues are processed ?
	return func() error {
		if ctx != nil {
			shutdownTimeout := ctx.Value(types.GlobalShutdownTimeout)
			if shutdownTimeout != nil {
				if timeout, err := strconv.ParseInt(shutdownTimeout.(string), 10, 64); err == nil {
					al.waitConnectionsClose(time.Millisecond * time.Duration(timeout))
				}
			}
		}
		return nil
	}
}

// compatible with go 1.12.x
func Milliseconds(d time.Duration) int64 { return int64(d) / 1e6 }

func (al *activeListener) removeConnection(ac *activeConnection) {
	al.conns.Remove(ac.element)

	atomic.AddInt64(&al.handler.numConnections, -1)

}

func (al *activeListener) newConnection(ctx context.Context, rawc net.Conn) {
	conn := network.NewServerConnection(ctx, rawc, al.stopChan)
	if al.idleTimeout != nil {
		conn.SetIdleTimeout(types.DefaultConnReadTimeout, al.idleTimeout.Duration)
	} else {
		// a nil idle timeout, we set a default one
		// notice only server side connection set the default value
		switch conn.LocalAddr().Network() {
		case "udp":
			conn.SetIdleTimeout(types.DefaultUDPReadTimeout, types.DefaultUDPIdleTimeout)
		default:
			conn.SetIdleTimeout(types.DefaultConnReadTimeout, types.DefaultIdleTimeout)
		}
	}
	oriRemoteAddr, err := variable.Get(ctx, types.VariableOriRemoteAddr)
	if err == nil && oriRemoteAddr != nil {
		conn.SetRemoteAddr(oriRemoteAddr.(net.Addr))
	}
	listeners, err := variable.Get(ctx, types.VariableConnectionEventListeners)
	if err == nil && listeners != nil {
		for _, listener := range listeners.([]api.ConnectionEventListener) {
			conn.AddConnectionEventListener(listener)
		}
	}
	_ = variable.Set(ctx, types.VariableConnectionID, conn.ID())
	_ = variable.Set(ctx, types.VariableConnection, conn)

	conn.SetBufferLimit(al.listener.PerConnBufferLimitBytes())

	al.OnNewConnection(ctx, conn)
}

type activeRawConn struct {
	rawc                net.Conn
	rawf                *os.File
	ctx                 context.Context
	originalDstIP       string
	originalDstPort     int
	oriRemoteAddr       net.Addr
	useOriginalDst      bool
	rawcElement         *list.Element
	activeListener      *activeListener
	acceptedFilters     []api.ListenerFilterChainFactory
	acceptedFilterIndex int
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
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("[server] [conn] conn set origin addr:%s:%d", ip, port)
	}
}

func init() {
	variable.Register(
		variable.NewStringVariable(types.VarListenerMatchFallbackIP, nil, nil, variable.DefaultStringSetter, 0),
	)
}

const fallback_any = "0.0.0.0"

func (arc *activeRawConn) UseOriginalDst(ctx context.Context) {
	var listener, localListener *activeListener

	// if listener match ip address, we will use a fallback listener.
	// default action is fallback to a listener which listen ip 0.0.0.0
	fallbackip := fallback_any
	if v, err := variable.GetString(arc.ctx, types.VarListenerMatchFallbackIP); err == nil {
		fallbackip = v
	}

	for _, lst := range arc.activeListener.handler.listeners {
		if lst.listenIP == arc.originalDstIP && lst.listenPort == arc.originalDstPort {
			listener = lst
			break
		}

		if lst.listenPort == arc.originalDstPort && lst.listenIP == fallbackip {
			localListener = lst
		}
	}

	var ch chan api.Connection
	var buf []byte
	if val, err := variable.Get(ctx, types.VariableAcceptChan); err == nil && val != nil {
		ch = val.(chan api.Connection)
		if val, err := variable.Get(ctx, types.VariableAcceptBuffer); err == nil && val != nil {
			buf = val.([]byte)
		}
	}

	if listener != nil {
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[server] [conn] found original dest listener :%s:%d", listener.listenIP, listener.listenPort)
		}
		listener.OnAccept(arc.rawc, false, arc.oriRemoteAddr, ch, buf, nil)
		return
	}

	if localListener != nil {
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[server] [conn] use fallback listener for original dest:%s:%d", localListener.listenIP, localListener.listenPort)
		}
		localListener.OnAccept(arc.rawc, false, arc.oriRemoteAddr, ch, buf, nil)
		return
	}

	// If it can’t find any matching listeners and should use the self listener.
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("[server] [conn] no listener found for original dest, fallback to listener filter: %s:%d", arc.activeListener.listenIP, arc.activeListener.listenPort)
	}
	arc.activeListener.OnAccept(arc.rawc, false, arc.oriRemoteAddr, ch, buf, nil)
}

func (arc *activeRawConn) ContinueFilterChain(ctx context.Context, success bool) {

	if !success {
		return
	}

	for ; arc.acceptedFilterIndex < len(arc.acceptedFilters); arc.acceptedFilterIndex++ {
		filterStatus := arc.acceptedFilters[arc.acceptedFilterIndex].OnAccept(arc)
		if filterStatus == api.Stop {
			return
		}
	}

	arc.activeListener.newConnection(ctx, arc.rawc)

}

func (arc *activeRawConn) Conn() net.Conn {
	return arc.rawc
}

func (arc *activeRawConn) GetOriContext() context.Context {
	return arc.ctx
}

func (arc *activeRawConn) SetUseOriginalDst(flag bool) {
	arc.useOriginalDst = flag
}

func (arc *activeRawConn) GetUseOriginalDst() bool {
	return arc.useOriginalDst
}

// ConnectionEventListener
// ListenerFilterManager note:unsupported now
// ListenerFilterCallbacks note:unsupported now
type activeConnection struct {
	element  *list.Element
	listener *activeListener
	conn     api.Connection
}

func newActiveConnection(listener *activeListener, conn api.Connection) *activeConnection {
	ac := &activeConnection{
		conn:     conn,
		listener: listener,
	}

	ac.conn.SetNoDelay(true)
	ac.conn.AddConnectionEventListener(ac)
	ac.conn.AddBytesReadListener(func(bytesRead uint64) {

		if bytesRead > 0 {
			listener.stats.DownstreamBytesReadTotal.Inc(int64(bytesRead))
		}
	})
	ac.conn.AddBytesSentListener(func(bytesSent uint64) {

		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("update listener write bytes: %d", bytesSent)
		}
		if bytesSent > 0 {
			listener.stats.DownstreamBytesWriteTotal.Inc(int64(bytesSent))
		}
	})

	return ac
}

// ConnectionEventListener
func (ac *activeConnection) OnEvent(event api.ConnectionEvent) {
	if event.IsClose() {
		ac.listener.removeConnection(ac)
	}
}

func sendInheritListeners() (net.Conn, error) {
	lf := ListListenersFile()
	if lf == nil {
		return nil, errors.New("ListListenersFile() error")
	}

	lsf, lerr := admin.ListServiceListenersFile()
	if lerr != nil {
		return nil, errors.New("ListServiceListenersFile() error")
	}

	var files []*os.File
	files = append(files, lf...)
	files = append(files, lsf...)

	if len(files) > 100 {
		log.DefaultLogger.Errorf("[server] InheritListener fd too many :%d", len(files))
		return nil, errors.New("InheritListeners too many")
	}
	fds := make([]int, len(files))
	for i, f := range files {
		fds[i] = int(f.Fd())
		log.DefaultLogger.Debugf("[server] InheritListener fd: %d", f.Fd())
		defer f.Close()
	}

	var unixConn net.Conn
	var err error
	// retry 10 time
	for i := 0; i < 10; i++ {
		unixConn, err = net.DialTimeout("unix", types.TransferListenDomainSocket, 1*time.Second)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.DefaultLogger.Errorf("[server] sendInheritListeners Dial unix failed %v", err)
		return nil, err
	}

	uc := unixConn.(*net.UnixConn)
	buf := make([]byte, 1)
	rights := syscall.UnixRights(fds...)
	n, oobn, err := uc.WriteMsgUnix(buf, rights, nil)
	if err != nil {
		log.DefaultLogger.Errorf("[server] WriteMsgUnix: %v", err)
		return nil, err
	}
	if n != len(buf) || oobn != len(rights) {
		log.DefaultLogger.Errorf("[server] WriteMsgUnix = %d, %d; want 1, %d", n, oobn, len(rights))
		return nil, err
	}

	return uc, nil
}

// SendInheritConfig send to new mosn using uinx dowmain socket
func SendInheritConfig() error {
	var unixConn net.Conn
	var err error
	// retry 10 time
	for i := 0; i < 10; i++ {
		unixConn, err = net.DialTimeout("unix", types.TransferMosnconfigDomainSocket, 1*time.Second)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.DefaultLogger.Errorf("[server] SendInheritConfig Dial unix failed %v", err)
		return err
	}

	configData, err := configmanager.InheritMosnconfig()
	if err != nil {
		return err
	}

	uc := unixConn.(*net.UnixConn)
	defer uc.Close()

	n, err := uc.Write(configData)
	if err != nil {
		log.DefaultLogger.Errorf("[server] Write: %v", err)
		return err
	}
	if n != len(configData) {
		log.DefaultLogger.Errorf("[server] Write = %d, want %d", n, len(configData))
		return errors.New("write mosnconfig data length error")
	}

	return nil
}

func GetInheritListeners() ([]net.Listener, []net.PacketConn, net.Conn, error) {
	syscall.Unlink(types.TransferListenDomainSocket)

	l, err := net.Listen("unix", types.TransferListenDomainSocket)
	if err != nil {
		log.StartLogger.Errorf("[server] InheritListeners net listen error: %v", err)
		return nil, nil, nil, err
	}
	defer l.Close()

	log.StartLogger.Infof("[server] Get InheritListeners start")

	ul := l.(*net.UnixListener)
	ul.SetDeadline(time.Now().Add(time.Second * 10))
	uc, err := ul.AcceptUnix()
	if err != nil {
		log.StartLogger.Errorf("[server] InheritListeners Accept error :%v", err)
		return nil, nil, nil, err
	}
	log.StartLogger.Infof("[server] Get InheritListeners Accept")

	buf := make([]byte, 1)
	oob := make([]byte, 1024)
	_, oobn, _, _, err := uc.ReadMsgUnix(buf, oob)
	if err != nil {
		return nil, nil, nil, err
	}
	scms, err := unix.ParseSocketControlMessage(oob[0:oobn])
	if err != nil {
		log.StartLogger.Errorf("[server] ParseSocketControlMessage: %v", err)
		return nil, nil, nil, err
	}
	if len(scms) != 1 {
		log.StartLogger.Errorf("[server] expected 1 SocketControlMessage; got scms = %#v", scms)
		return nil, nil, nil, err
	}
	gotFds, err := unix.ParseUnixRights(&scms[0])
	if err != nil {
		log.StartLogger.Errorf("[server] unix.ParseUnixRights: %v", err)
		return nil, nil, nil, err
	}

	var listeners []net.Listener
	var packetConn []net.PacketConn
	for i := 0; i < len(gotFds); i++ {
		fd := uintptr(gotFds[i])
		file := os.NewFile(fd, "")
		if file == nil {
			log.StartLogger.Errorf("[server] create new file from fd %d failed", fd)
			return nil, nil, nil, err
		}
		defer file.Close()

		fileListener, err := net.FileListener(file)
		if err != nil {
			pc, err := net.FilePacketConn(file)
			if err == nil {
				packetConn = append(packetConn, pc)
			} else {

				log.StartLogger.Errorf("[server] recover listener from fd %d failed: %s", fd, err)
				return nil, nil, nil, err
			}
		} else {
			// for tcp or unix listener
			listeners = append(listeners, fileListener)
		}
	}

	return listeners, packetConn, uc, nil
}

func GetInheritConfig() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			log.StartLogger.Errorf("[server] GetInheritConfig panic %v", r)
		}
	}()

	syscall.Unlink(types.TransferMosnconfigDomainSocket)

	l, err := net.Listen("unix", types.TransferMosnconfigDomainSocket)
	if err != nil {
		log.StartLogger.Errorf("[server] GetInheritConfig net listen error: %v", err)
		return nil, err
	}
	defer l.Close()

	log.StartLogger.Infof("[server] Get GetInheritConfig start")

	ul := l.(*net.UnixListener)
	ul.SetDeadline(time.Now().Add(time.Second * 10))
	uc, err := ul.AcceptUnix()
	if err != nil {
		log.StartLogger.Errorf("[server] GetInheritConfig Accept error :%v", err)
		return nil, err
	}
	defer uc.Close()
	log.StartLogger.Infof("[server] Get GetInheritConfig Accept")
	configData := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := uc.Read(buf)
		configData = append(configData, buf[:n]...)
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}

	// log.StartLogger.Infof("[server] inherit mosn config data: %v", string(configData))

	return configData, nil
}
