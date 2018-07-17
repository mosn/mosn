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
	"sync"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/network/buffer"
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var (
	codecHeadersBufPool types.HeadersBufferPool
	activeStreamPool    types.ObjectBufferPool
	globalStats         *proxyStats
)

func init() {
	globalStats = newProxyStats(types.GlobalStatsNamespace)
	codecHeadersBufPool = buffer.NewHeadersBufferPool(1)
	activeStreamPool = buffer.NewObjectPool(1)
}

// types.ReadFilter
// types.ServerStreamConnectionEventListener
type proxy struct {
	config              *v2.Proxy
	clusterManager      types.ClusterManager
	readCallbacks       types.ReadFilterCallbacks
	upstreamConnection  types.ClientConnection
	downstreamCallbacks DownstreamCallbacks

	clusterName    string
	routers        types.Routers
	serverCodec    types.ServerStreamConnection
	resueCodecMaps bool
	codecPool      types.HeadersBufferPool

	context context.Context

	// downstream requests
	activeSteams *list.List
	asMux        sync.RWMutex

	// stats
	stats *proxyStats

	// listener stats
	listenerStats *listenerStats

	// access logs
	accessLogs []types.AccessLog
}

func NewProxy(config *v2.Proxy, clusterManager types.ClusterManager, ctx context.Context) Proxy {
	ctx = context.WithValue(ctx, types.ContextKeyConnectionCodecMapPool, codecHeadersBufPool)

	proxy := &proxy{
		config:         config,
		clusterManager: clusterManager,
		activeSteams:   list.New(),
		stats:          globalStats,
		resueCodecMaps: true,
		codecPool:      codecHeadersBufPool,
		context:        ctx,
		accessLogs:     ctx.Value(types.ContextKeyAccessLogs).([]types.AccessLog),
	}

	listenStatsNamespace := ctx.Value(types.ContextKeyListenerStatsNameSpace).(string)
	proxy.listenerStats = newListenerStats(listenStatsNamespace)
	proxy.routers, _ = router.CreateRouteConfig(types.Protocol(config.DownstreamProtocol), config)
	proxy.downstreamCallbacks = &downstreamCallbacks{
		proxy: proxy,
	}

	return proxy
}

func (p *proxy) OnData(buf types.IoBuffer) types.FilterStatus {
	p.serverCodec.Dispatch(buf)

	return types.StopIteration
}

//rpc realize upstream on event
func (p *proxy) onDownstreamEvent(event types.ConnectionEvent) {
	if event.IsClose() {
		p.stats.DownstreamConnectionDestroy().Inc(1)
		p.stats.DownstreamConnectionActive().Dec(1)
		var urEleNext *list.Element

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

	cb.Connection().SetStats(&types.ConnectionStats{
		ReadTotal:    p.stats.DownstreamBytesRead(),
		ReadCurrent:  p.stats.DownstreamBytesReadCurrent(),
		WriteTotal:   p.stats.DownstreamBytesWrite(),
		WriteCurrent: p.stats.DownstreamBytesWriteCurrent(),
	})

	p.stats.DownstreamConnectionTotal().Inc(1)
	p.stats.DownstreamConnectionActive().Inc(1)

	p.readCallbacks.Connection().AddConnectionEventListener(p.downstreamCallbacks)
	p.serverCodec = stream.CreateServerStreamConnection(p.context, types.Protocol(p.config.DownstreamProtocol), p.readCallbacks.Connection(), p)
}

func (p *proxy) OnGoAway() {}

func (p *proxy) NewStream(streamId string, responseSender types.StreamSender) types.StreamReceiver {
	stream := newActiveStream(streamId, p, responseSender)

	if ff := p.context.Value(types.ContextKeyStreamFilterChainFactories); ff != nil {
		ffs := ff.([]types.StreamFilterChainFactory)

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
	// reuse decode map
	if p.resueCodecMaps {
		if s.downstreamReqHeaders != nil {
			//p.codecPool.Give(s.downstreamReqHeaders)
		}

		if s.upstreamRequest != nil {
			if s.upstreamRequest.upstreamRespHeaders != nil {
				//p.codecPool.Give(s.upstreamRequest.upstreamRespHeaders)
			}
		}
	}

	p.asMux.Lock()
	p.activeSteams.Remove(s.element)
	p.asMux.Unlock()

	//s.reset()
}

// ConnectionEventListener
type downstreamCallbacks struct {
	proxy *proxy
}

func (dc *downstreamCallbacks) OnEvent(event types.ConnectionEvent) {
	dc.proxy.onDownstreamEvent(event)
}
