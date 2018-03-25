package proxy

import (
	"sync"
	"container/list"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/router"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"github.com/rcrowley/go-metrics"
	"fmt"
)

// types.ReadFilter
// types.ServerStreamConnectionCallbacks
type proxy struct {
	config              *v2.Proxy
	clusterManager      types.ClusterManager
	readCallbacks       types.ReadFilterCallbacks
	upstreamConnection  types.ClientConnection
	downstreamCallbacks DownstreamCallbacks

	clusterName  string
	routerConfig types.RouterConfig
	serverCodec  types.ServerStreamConnection

	// downstream requests
	activeSteams *list.List
	asMux        sync.RWMutex

	// stats
	stats *proxyStats

	// access logs
	accessLogs []types.AccessLog
}

func NewProxy(config *v2.Proxy, clusterManager types.ClusterManager) Proxy {
	proxy := &proxy{
		config:         config,
		clusterManager: clusterManager,
		activeSteams:   list.New(),
	}

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
		p.stats.downstreamConnDestroy.Inc(1)
		p.stats.downstreamConnActive.Dec(1)

		p.asMux.RLock()
		defer p.asMux.RUnlock()

		for urEle := p.activeSteams.Front(); urEle != nil; urEle = urEle.Next() {
			ur := urEle.Value.(*upstreamRequest)
			ur.requestEncoder.GetStream().ResetStream(types.StreamLocalReset)
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
	p.initProxyStats(cb)

	p.stats.downstreamConnTotal.Inc(1)
	p.stats.downstreamConnActive.Inc(1)

	p.readCallbacks.Connection().AddConnectionCallbacks(p.downstreamCallbacks)
	p.serverCodec = stream.CreateServerStreamConnection(types.Protocol(p.config.DownstreamProtocol), p.readCallbacks.Connection(), p)
}

func (p *proxy) initProxyStats(cb types.ReadFilterCallbacks) {
	p.stats = &proxyStats{
		downstreamConnTotal:     metrics.GetOrRegisterCounter(proxyStatsName(cb.Connection().Id(), "connection_total"), nil),
		downstreamConnDestroy:   metrics.GetOrRegisterCounter(proxyStatsName(cb.Connection().Id(), "connection_destroy"), nil),
		downstreamConnActive:    metrics.GetOrRegisterCounter(proxyStatsName(cb.Connection().Id(), "connection_active"), nil),
		downstreamRequestTotal:  metrics.GetOrRegisterCounter(proxyStatsName(cb.Connection().Id(), "request_total"), nil),
		downstreamRequestActive: metrics.GetOrRegisterCounter(proxyStatsName(cb.Connection().Id(), "request_active"), nil),
		downstreamRequestReset:  metrics.GetOrRegisterCounter(proxyStatsName(cb.Connection().Id(), "request_reset"), nil),
		downstreamRequestTime:   metrics.GetOrRegisterHistogram(proxyStatsName(cb.Connection().Id(), "request_time"), nil, metrics.NewUniformSample(100)),
	}
}

func proxyStatsName(connId uint64, statName string) string {
	return fmt.Sprintf("%d.proxy.downstream_%s", connId, statName)
}

func (p *proxy) OnGoAway() {}

//由stream层来调用
func (p *proxy) NewStream(streamId uint32, responseEncoder types.StreamEncoder) types.StreamDecoder {
	stream := newActiveStream(p, responseEncoder)

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

	}

	return 0
}

func (p *proxy) deleteActiveStream(s *activeStream) {
	p.asMux.Lock()
	p.activeSteams.Remove(s.element)
	p.asMux.Unlock()
}

// ConnectionCallbacks
type downstreamCallbacks struct {
	proxy *proxy
}

func (dc *downstreamCallbacks) OnEvent(event types.ConnectionEvent) {
	dc.proxy.onDownstreamEvent(event)
}

func (dc *downstreamCallbacks) OnAboveWriteBufferHighWatermark() {
	// TODO
}

func (dc *downstreamCallbacks) OnBelowWriteBufferLowWatermark() {
	// TODO
}

type proxyStats struct {
	downstreamConnTotal      metrics.Counter
	downstreamConnDestroy    metrics.Counter
	downstreamConnActive     metrics.Counter
	downstreamConnBytesTotal metrics.Counter
	downstreamRequestTotal   metrics.Counter
	downstreamRequestActive  metrics.Counter
	downstreamRequestReset   metrics.Counter
	downstreamRequestTime    metrics.Histogram
}

func (s *proxyStats) print() {
	log.DefaultLogger.Printf("downstream conn total %d, conn active %d, conn destroy %d, request total %d, request active %d, request reset %d",
		s.downstreamConnTotal.Count(), s.downstreamConnActive.Count(), s.downstreamConnDestroy.Count(), s.downstreamRequestTotal.Count(), s.downstreamRequestActive.Count(), s.downstreamRequestReset.Count())
}
