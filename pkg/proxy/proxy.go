package proxy

import (
	"sync"
	"container/list"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/router"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
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

	return proxy
}

func (p *proxy) OnData(buf types.IoBuffer) types.FilterStatus {
	p.serverCodec.Dispatch(buf)

	return types.StopIteration
}

//rpc realize upstream on event
func (p *proxy) onDownstreamEvent(event types.ConnectionEvent) {
	if event.IsClose() {
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
	p.readCallbacks.Connection().AddConnectionCallbacks(p.downstreamCallbacks)
	p.serverCodec = stream.CreateServerStreamConnection(types.Protocol(p.config.DownstreamProtocol), p.readCallbacks.Connection(), p)

	// TODO: set downstream connection stats
}

func (p *proxy) OnGoAway() {}

func (p *proxy) NewStream(streamId uint32, responseEncoder types.StreamEncoder) types.StreamDecoder {
	stream := &activeStream{
		proxy:           p,
		requestInfo:     network.NewRequestInfo(),
		responseEncoder: responseEncoder,
	}

	stream.responseEncoder.GetStream().AddCallbacks(stream)

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
