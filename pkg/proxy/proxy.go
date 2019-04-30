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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	mosnctx "github.com/alipay/sofa-mosn/pkg/context"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/mtls"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/stream"
	mosnsync "github.com/alipay/sofa-mosn/pkg/sync"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/json-iterator/go"
	"github.com/rcrowley/go-metrics"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var (
	globalStats *Stats

	currProxyID uint32

	pool mosnsync.WorkerPool
)

func init() {
	// register init function with interest of P number
	config.RegisterConfigParsedListener(config.ParseCallbackKeyProcessor, initWorkerPool)
}

func initWorkerPool(data interface{}, endParsing bool) error {
	initGlobalStats()

	poolSize := runtime.NumCPU() * 1024

	// set poolSize equal to processor if it was specified
	if pNum, ok := data.(int); ok && pNum > 0 {
		poolSize = pNum * 1024
	}

	pool = mosnsync.NewWorkerPool(poolSize)

	return nil
}

func initGlobalStats() {
	globalStats = newProxyStats(types.GlobalProxyName)
}

// types.ReadFilter
// types.ServerStreamConnectionEventListener
type proxy struct {
	config             *v2.Proxy
	clusterManager     types.ClusterManager
	readCallbacks      types.ReadFilterCallbacks
	upstreamConnection types.ClientConnection
	downstreamListener types.ConnectionEventListener
	clusterName        string
	routersWrapper     types.RouterWrapper // wrapper used to point to the routers instance
	serverStreamConn   types.ServerStreamConnection
	context            context.Context
	activeSteams       *list.List // downstream requests
	asMux              sync.RWMutex
	stats              *Stats
	listenerStats      *Stats
	accessLogs         []types.AccessLog
}

// NewProxy create proxy instance for given v2.Proxy config
func NewProxy(ctx context.Context, config *v2.Proxy, clusterManager types.ClusterManager) Proxy {
	proxy := &proxy{
		config:         config,
		clusterManager: clusterManager,
		activeSteams:   list.New(),
		stats:          globalStats,
		context:        ctx,
		accessLogs:     mosnctx.Get(ctx, types.ContextKeyAccessLogs).([]types.AccessLog),
	}

	extJSON, err := json.Marshal(proxy.config.ExtendConfig)
	if err == nil {
		log.DefaultLogger.Tracef("[proxy] extend config = %v", proxy.config.ExtendConfig)
		var xProxyExtendConfig v2.XProxyExtendConfig
		json.Unmarshal([]byte(extJSON), &xProxyExtendConfig)
		proxy.context = mosnctx.WithValue(proxy.context, types.ContextSubProtocol, xProxyExtendConfig.SubProtocol)
		log.DefaultLogger.Tracef("[proxy] extend config subprotocol = %v", xProxyExtendConfig.SubProtocol)
	} else {
		log.DefaultLogger.Errorf("[proxy] get proxy extend config fail = %v", err)
	}

	listenerName := mosnctx.Get(ctx, types.ContextKeyListenerName).(string)
	proxy.listenerStats = newListenerStats(listenerName)

	if routersWrapper := router.GetRoutersMangerInstance().GetRouterWrapperByName(proxy.config.RouterConfigName); routersWrapper != nil {
		proxy.routersWrapper = routersWrapper
	} else {
		log.DefaultLogger.Errorf("[proxy] RouterConfigName:%s doesn't exit", proxy.config.RouterConfigName)
	}

	proxy.downstreamListener = &downstreamCallbacks{
		proxy: proxy,
	}

	return proxy
}

func (p *proxy) OnData(buf types.IoBuffer) types.FilterStatus {
	if p.serverStreamConn == nil {
		var prot string
		if conn, ok := p.readCallbacks.Connection().RawConn().(*mtls.TLSConn); ok {
			prot = conn.ConnectionState().NegotiatedProtocol
		}
		protocol, err := stream.SelectStreamFactoryProtocol(prot, buf.Bytes())
		if err == stream.EAGAIN {
			return types.Stop
		} else if err == stream.FAILED {
			var size int
			if buf.Len() > 10 {
				size = 10
			} else {
				size = buf.Len()
			}
			log.DefaultLogger.Errorf("[proxy] Protocol Auto error magic :%v", buf.Bytes()[:size])
			p.readCallbacks.Connection().Close(types.NoFlush, types.OnReadErrClose)
			return types.Stop
		}
		log.DefaultLogger.Debugf("[proxy] Protoctol Auto: %v", protocol)
		p.serverStreamConn = stream.CreateServerStreamConnection(p.context, protocol, p.readCallbacks.Connection(), p)
	}
	p.serverStreamConn.Dispatch(buf)

	return types.Stop
}

//rpc realize upstream on event
func (p *proxy) onDownstreamEvent(event types.ConnectionEvent) {
	if event.IsClose() {
		p.stats.DownstreamConnectionDestroy.Inc(1)
		p.stats.DownstreamConnectionActive.Dec(1)
		p.listenerStats.DownstreamConnectionDestroy.Inc(1)
		p.listenerStats.DownstreamConnectionActive.Dec(1)
		var urEleNext *list.Element

		p.asMux.RLock()
		defer p.asMux.RUnlock()

		for urEle := p.activeSteams.Front(); urEle != nil; urEle = urEleNext {
			urEleNext = urEle.Next()

			ds := urEle.Value.(*downStream)
			ds.OnResetStream(types.StreamConnectionTermination)
		}
	}
}

func (p *proxy) ReadDisableUpstream(disable bool) {
	// TODO
}

func (p *proxy) ReadDisableDownstream(disable bool) {
	// TODO
}

func (p *proxy) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {
	p.readCallbacks = cb

	// bytes total adds all connections data together, but buffered data not
	cb.Connection().SetStats(&types.ConnectionStats{
		ReadTotal:     p.stats.DownstreamBytesReadTotal,
		ReadBuffered:  metrics.NewGauge(),
		WriteTotal:    p.stats.DownstreamBytesWriteTotal,
		WriteBuffered: metrics.NewGauge(),
	})

	p.stats.DownstreamConnectionTotal.Inc(1)
	p.stats.DownstreamConnectionActive.Inc(1)
	p.listenerStats.DownstreamConnectionTotal.Inc(1)
	p.listenerStats.DownstreamConnectionActive.Inc(1)

	p.readCallbacks.Connection().AddConnectionEventListener(p.downstreamListener)
	if p.config.DownstreamProtocol != string(protocol.Auto) {
		p.serverStreamConn = stream.CreateServerStreamConnection(p.context, types.Protocol(p.config.DownstreamProtocol), p.readCallbacks.Connection(), p)
	}
}

func (p *proxy) OnGoAway() {}

func (p *proxy) NewStreamDetect(ctx context.Context, responseSender types.StreamSender, span types.Span) types.StreamReceiveListener {
	stream := newActiveStream(ctx, p, responseSender, span)

	if ff := mosnctx.Get(p.context, types.ContextKeyStreamFilterChainFactories); ff != nil {
		ffs := ff.([]types.StreamFilterChainFactory)

		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(stream.context, "[proxy][downstream] %d stream filters in config", len(ffs))
		}

		for _, f := range ffs {
			f.CreateFilterChain(p.context, stream)
		}
	}

	p.asMux.Lock()
	stream.element = p.activeSteams.PushBack(stream)
	p.asMux.Unlock()

	return stream
}

func (p *proxy) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (p *proxy) streamResetReasonToResponseFlag(reason types.StreamResetReason) types.ResponseFlag {
	switch reason {
	case types.StreamConnectionFailed:
		return types.UpstreamConnectionFailure
	case types.StreamConnectionTermination:
		return types.UpstreamConnectionTermination
	case types.StreamLocalReset:
		return types.UpstreamLocalReset
	case types.StreamOverflow:
		return types.UpstreamOverflow
	case types.StreamRemoteReset:
		return types.UpstreamRemoteReset
	}

	return 0
}

func (p *proxy) deleteActiveStream(s *downStream) {
	if s.element != nil {
		p.asMux.Lock()
		p.activeSteams.Remove(s.element)
		p.asMux.Unlock()
	}
}

// ConnectionEventListener
type downstreamCallbacks struct {
	proxy *proxy
}

func (dc *downstreamCallbacks) OnEvent(event types.ConnectionEvent) {
	dc.proxy.onDownstreamEvent(event)
}
