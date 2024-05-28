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

package proxy

import (
	"container/list"
	"context"
	"runtime"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/streamfilter"
	mosnsync "mosn.io/mosn/pkg/sync"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var (
	globalStats *Stats

	currProxyID uint32

	pool mosnsync.WorkerPool
)

func init() {
	// register init function with interest of P number
	configmanager.RegisterConfigParsedListener(configmanager.ParseCallbackKeyProcessor, initWorkerPool)
}

func initWorkerPool(data interface{}, endParsing bool) error {
	initGlobalStats()

	poolSize := runtime.NumCPU() * 256

	// set poolSize equal to processor if it was specified
	if pNum, ok := data.(int); ok && pNum > 0 {
		poolSize = pNum * 256
	}

	pool = mosnsync.NewWorkerPool(poolSize)

	return nil
}

func GetWorkerPool() mosnsync.WorkerPool {
	return pool
}

func initGlobalStats() {
	globalStats = newProxyStats(types.GlobalProxyName)
}

// types.ReadFilter
// types.ServerStreamConnectionEventListener
type proxy struct {
	config              *v2.Proxy
	clusterManager      types.ClusterManager
	readCallbacks       api.ReadFilterCallbacks
	downstreamListener  api.ConnectionEventListener
	routersWrapper      types.RouterWrapper // wrapper used to point to the routers instance
	fallback            bool
	serverStreamConn    types.ServerStreamConnection
	context             context.Context
	activeStreams       *list.List // downstream requests
	asMux               sync.RWMutex
	stats               *Stats
	listenerStats       *Stats
	accessLogs          []api.AccessLog
	streamFilterFactory streamfilter.StreamFilterFactory
	routeHandlerFactory router.MakeHandlerFunc

	protocols []api.ProtocolName

	// configure the proxy level worker pool
	// eg. if we want the requests on one connection to keep serial,
	// we only start one worker(goroutine) on this connection
	// request will blocking wait for the previous to finish sending
	workerpool mosnsync.WorkerPool
}

// NewProxy create proxy instance for given v2.Proxy config
func NewProxy(ctx context.Context, config *v2.Proxy) Proxy {
	aclog, _ := variable.Get(ctx, types.VariableAccessLogs)
	proxy := &proxy{
		config:         config,
		clusterManager: cluster.GetClusterMngAdapterInstance().ClusterManager,
		activeStreams:  list.New(),
		stats:          globalStats,
		context:        ctx,
		accessLogs:     aclog.([]api.AccessLog),
	}

	if pi, err := variable.Get(ctx, types.VarProtocolConfig); err == nil {
		if protos, ok := pi.([]api.ProtocolName); ok {
			proxy.protocols = protos
		}
	}

	// proxy level worker pool config
	if config.ConcurrencyNum > 0 {
		proxy.workerpool = mosnsync.NewWorkerPool(config.ConcurrencyNum)
	}
	// proxy level worker pool config end

	lv, _ := variable.Get(ctx, types.VariableListenerName)
	listenerName := lv.(string)
	proxy.listenerStats = newListenerStats(listenerName)

	if routersWrapper := router.GetRoutersMangerInstance().GetRouterWrapperByName(proxy.config.RouterConfigName); routersWrapper != nil {
		proxy.routersWrapper = routersWrapper
	} else {
		log.DefaultLogger.Alertf("proxy.config", "[proxy] RouterConfigName:%s doesn't exit", proxy.config.RouterConfigName)
	}

	proxy.downstreamListener = &downstreamCallbacks{
		proxy: proxy,
	}

	proxy.streamFilterFactory = streamfilter.GetStreamFilterManager().GetStreamFilterFactory(listenerName)
	proxy.routeHandlerFactory = router.GetMakeHandlerFunc(proxy.config.RouterHandlerName)

	return proxy
}

func (p *proxy) OnData(buf buffer.IoBuffer) api.FilterStatus {
	if p.fallback {
		return api.Continue
	}

	if p.serverStreamConn == nil {
		var prot string
		if conn, ok := p.readCallbacks.Connection().RawConn().(*mtls.TLSConn); ok {
			prot = conn.ConnectionState().NegotiatedProtocol
		}

		scopes := p.protocols
		// if protocols is Auto only, match all registered protocols
		if len(scopes) == 1 && scopes[0] == protocol.Auto {
			scopes = nil
		}

		proto, err := stream.SelectStreamFactoryProtocol(p.context, prot, buf.Bytes(), scopes)

		if err == stream.EAGAIN {
			return api.Stop
		}
		if err == stream.FAILED {
			if p.config.FallbackForUnknownProtocol {
				p.fallback = true
				return api.Continue
			}

			var size int
			if buf.Len() > 10 {
				size = 10
			} else {
				size = buf.Len()
			}
			log.DefaultLogger.Alertf("proxy.auto", "[proxy] Protocol Auto error magic :%v", buf.Bytes()[:size])
			p.readCallbacks.Connection().Close(api.NoFlush, api.OnReadErrClose)
			return api.Stop
		}

		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[proxy] Protoctol Auto: %v", proto)
		}

		p.serverStreamConn = stream.CreateServerStreamConnection(p.context, proto, p.readCallbacks.Connection(), p)
	}
	p.serverStreamConn.Dispatch(buf)

	return api.Stop
}

// rpc realize upstream on event
func (p *proxy) onDownstreamEvent(event api.ConnectionEvent) {
	if event.IsClose() {
		p.stats.DownstreamConnectionDestroy.Inc(1)
		p.stats.DownstreamConnectionActive.Dec(1)
		p.listenerStats.DownstreamConnectionDestroy.Inc(1)
		p.listenerStats.DownstreamConnectionActive.Dec(1)
		var urEleNext *list.Element

		p.asMux.RLock()
		defer p.asMux.RUnlock()

		for urEle := p.activeStreams.Front(); urEle != nil; urEle = urEleNext {
			urEleNext = urEle.Next()

			ds := urEle.Value.(*downStream)
			// don't need to resetStream when upstream already response
			// so that the stream can be reused.
			if ds.upstreamProcessDone.Load() {
				continue
			}
			ds.OnResetStream(types.StreamConnectionTermination)
		}
		return
	}
	if event == api.OnReadTimeout {
		if p.shouldFallback() {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[proxy] wait for fallback timeout, do fallback")
			}

			p.fallback = true
			p.readCallbacks.ContinueReading()
		}
		return
	}
	if event == api.OnShutdown && p.serverStreamConn != nil {
		p.serverStreamConn.GoAway()
		return
	}
}

func (p *proxy) shouldFallback() bool {
	if !p.config.FallbackForUnknownProtocol || p.serverStreamConn != nil {
		return false
	}
	if p.fallback == true {
		return false
	}
	// not yet receive any data, and then timeout happens, should keep waiting for data and not fallback
	if p.readCallbacks.Connection().GetReadBuffer().Len() == 0 {
		return false
	}

	return true
}

func (p *proxy) ReadDisableUpstream(disable bool) {
	// TODO
}

func (p *proxy) ReadDisableDownstream(disable bool) {
	// TODO
}

func (p *proxy) ActiveStreamSize() int {
	if p.activeStreams == nil {
		return 0
	}
	return p.activeStreams.Len()
}

func (p *proxy) InitializeReadFilterCallbacks(cb api.ReadFilterCallbacks) {
	p.readCallbacks = cb

	// bytes total adds all connections data together
	cb.Connection().SetCollector(p.stats.DownstreamBytesReadTotal, p.stats.DownstreamBytesWriteTotal)

	p.stats.DownstreamConnectionTotal.Inc(1)
	p.stats.DownstreamConnectionActive.Inc(1)
	p.listenerStats.DownstreamConnectionTotal.Inc(1)
	p.listenerStats.DownstreamConnectionActive.Inc(1)

	p.readCallbacks.Connection().AddConnectionEventListener(p.downstreamListener)
	if len(p.protocols) == 1 && p.protocols[0] != protocol.Auto {
		p.serverStreamConn = stream.CreateServerStreamConnection(p.context, api.ProtocolName(p.protocols[0]), p.readCallbacks.Connection(), p)
	}
}

func (p *proxy) OnGoAway() {}

func (p *proxy) NewStreamDetect(ctx context.Context, responseSender types.StreamSender, span api.Span) types.StreamReceiveListener {
	stream := newActiveStream(ctx, p, responseSender, span)

	if p.streamFilterFactory != nil {
		p.streamFilterFactory.CreateFilterChain(ctx, stream.getStreamFilterChainRegisterCallback())
	}

	p.asMux.Lock()
	stream.element = p.activeStreams.PushBack(stream)
	p.asMux.Unlock()

	return stream
}

func (p *proxy) OnNewConnection() api.FilterStatus {
	return api.Continue
}

func (p *proxy) streamResetReasonToResponseFlag(reason types.StreamResetReason) api.ResponseFlag {
	switch reason {
	case types.StreamConnectionFailed:
		return api.UpstreamConnectionFailure
	case types.StreamConnectionTermination:
		return api.UpstreamConnectionTermination
	case types.StreamLocalReset:
		return api.UpstreamLocalReset
	case types.StreamOverflow:
		return api.UpstreamOverflow
	case types.StreamRemoteReset:
		return api.UpstreamRemoteReset
	case types.UpstreamGlobalTimeout, types.UpstreamPerTryTimeout:
		return api.UpstreamRequestTimeout
	}

	return 0
}

func (p *proxy) deleteActiveStream(s *downStream) {
	if s.element != nil {
		p.asMux.Lock()
		p.activeStreams.Remove(s.element)
		p.asMux.Unlock()
		s.element = nil
	}
}

// ConnectionEventListener
type downstreamCallbacks struct {
	proxy *proxy
}

func (dc *downstreamCallbacks) OnEvent(event api.ConnectionEvent) {
	dc.proxy.onDownstreamEvent(event)
}
