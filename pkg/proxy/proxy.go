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
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/stream"
	mosnsync "github.com/alipay/sofa-mosn/pkg/sync"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/json-iterator/go"
	"github.com/rcrowley/go-metrics"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/mtls"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var (
	globalStats *Stats

	currProxyID uint32
	workerPool  mosnsync.ShardWorkerPool
)

func init() {
	globalStats = newProxyStats(types.GlobalProxyName)

	// default shardsNum is equal to the cpu num
	shardsNum := runtime.NumCPU()
	// use 4096 as chan buffer length
	poolSize := shardsNum * 4096

	workerPool, _ = mosnsync.NewShardWorkerPool(poolSize, shardsNum, eventDispatch)
	workerPool.Init()
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
		accessLogs:     ctx.Value(types.ContextKeyAccessLogs).([]types.AccessLog),
	}

	extJSON, err := json.Marshal(proxy.config.ExtendConfig)
	if err == nil {
		log.DefaultLogger.Tracef("proxy extend config = %v", proxy.config.ExtendConfig)
		var xProxyExtendConfig v2.XProxyExtendConfig
		json.Unmarshal([]byte(extJSON), &xProxyExtendConfig)
		proxy.context = context.WithValue(proxy.context, types.ContextSubProtocol, xProxyExtendConfig.SubProtocol)
		log.DefaultLogger.Tracef("proxy extend config subprotocol = %v", xProxyExtendConfig.SubProtocol)
	} else {
		log.DefaultLogger.Errorf("get proxy extend config fail = %v", err)
	}

	listenerName := ctx.Value(types.ContextKeyListenerName).(string)
	proxy.listenerStats = newListenerStats(listenerName)

	if routersWrapper := router.GetRoutersMangerInstance().GetRouterWrapperByName(proxy.config.RouterConfigName); routersWrapper != nil {
		proxy.routersWrapper = routersWrapper
	} else {
		log.DefaultLogger.Errorf("RouterConfigName:%s doesn't exit", proxy.config.RouterConfigName)
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
			log.DefaultLogger.Errorf("Protocol Auto error magic :%v", buf.Bytes()[:size])
			p.readCallbacks.Connection().Close(types.NoFlush, types.OnReadErrClose)
			return types.Stop
		}
		log.DefaultLogger.Debugf("Protoctol Auto: %v", protocol)
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
		downStreams := make([]*downStream, 0, p.activeSteams.Len())
		for urEle := p.activeSteams.Front(); urEle != nil; urEle = urEleNext {
			urEleNext = urEle.Next()

			ds := urEle.Value.(*downStream)
			downStreams = append(downStreams, ds)
		}
		p.asMux.RUnlock()

		for _, ds := range downStreams {
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

func (p *proxy) NewStreamDetect(ctx context.Context, responseSender types.StreamSender, spanBuilder types.SpanBuilder) types.StreamReceiveListener {
	stream := newActiveStream(ctx, p, responseSender, spanBuilder)

	if ff := p.context.Value(types.ContextKeyStreamFilterChainFactories); ff != nil {
		ffs := ff.([]types.StreamFilterChainFactory)
		log.DefaultLogger.Debugf("there is %d stream filters in config", len(ffs))

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

func (p *proxy) convertProtocol() (dp, up types.Protocol) {
	if p.serverStreamConn == nil {
		dp = types.Protocol(p.config.DownstreamProtocol)
	} else {
		dp = p.serverStreamConn.Protocol()
	}
	up = types.Protocol(p.config.UpstreamProtocol)
	if up == protocol.Auto {
		up = dp
	}
	return
}