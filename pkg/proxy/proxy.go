package proxy

import (
	"container/list"
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/router"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"sync"
)

var globalStats *proxyStats

func init() {
	globalStats = newProxyStats(types.GlobalStatsNamespace)
}

// types.ReadFilter
// types.ServerStreamConnectionEventListener
type proxy struct {
	config              *v2.Proxy
	clusterManager      types.ClusterManager
	readCallbacks       types.ReadFilterCallbacks
	upstreamConnection  types.ClientConnection
	downstreamCallbacks DownstreamCallbacks

	clusterName  string
	routerConfig types.RouterConfig
	serverCodec  types.ServerStreamConnection

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
	proxy := &proxy{
		config:         config,
		clusterManager: clusterManager,
		activeSteams:   list.New(),
		stats:          globalStats,
		context:        ctx,
	}

	listenStatsNamespace := ctx.Value(types.ContextKeyListenerStatsNameSpace).(string)
	proxy.listenerStats = newListenerStats(listenStatsNamespace)

	proxy.routerConfig, _ = router.CreateRouteConfig(types.Protocol(config.DownstreamProtocol), config)
	proxy.downstreamCallbacks = &downstreamCallbacks{
		proxy: proxy,
	}
	for _, alConfig := range config.AccessLogs {
		al, _ := log.NewAccessLog(alConfig.Path, nil, alConfig.Format)
		proxy.accessLogs = append(proxy.accessLogs, al)
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

		p.asMux.RLock()
		defer p.asMux.RUnlock()

		for urEle := p.activeSteams.Front(); urEle != nil; urEle = urEle.Next() {
			ur := urEle.Value.(*upstreamRequest)
			ur.requestEncoder.GetStream().ResetStream(types.StreamConnectionTermination)
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
	p.serverCodec = stream.CreateServerStreamConnection(types.Protocol(p.config.DownstreamProtocol), p.readCallbacks.Connection(), p)
}

func (p *proxy) OnGoAway() {}

func (p *proxy) NewStream(streamId string, responseEncoder types.StreamEncoder) types.StreamDecoder {
	stream := newActiveStream(streamId, p, responseEncoder)

	if ff := p.context.Value(types.ContextKeyStreamFilterChainFactories); ff != nil {
		ffs := ff.([]types.StreamFilterChainFactory)

		for _, f := range ffs {
			f.CreateFilterChain(stream)
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

func (p *proxy) deleteActiveStream(s *activeStream) {
	p.asMux.Lock()
	p.activeSteams.Remove(s.element)
	p.asMux.Unlock()
}

// ConnectionEventListener
type downstreamCallbacks struct {
	proxy *proxy
}

func (dc *downstreamCallbacks) OnEvent(event types.ConnectionEvent) {
	dc.proxy.onDownstreamEvent(event)
}

func (dc *downstreamCallbacks) OnAboveWriteBufferHighWatermark() {
	dc.proxy.serverCodec.OnUnderlyingConnectionAboveWriteBufferHighWatermark()
}

func (dc *downstreamCallbacks) OnBelowWriteBufferLowWatermark() {
	dc.proxy.serverCodec.OnUnderlyingConnectionBelowWriteBufferLowWatermark()
}
