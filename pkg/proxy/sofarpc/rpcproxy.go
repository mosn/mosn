package sofarpc

import (
	"fmt"
	"reflect"
	"errors"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/router"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"container/list"
)

// 实现 sofa RPC 的 反向代理

// types.ReadFilter
// types.ServerStreamConnectionCallbacks
type rpcproxy struct {
	clusterManager      types.ClusterManager
	readCallbacks       types.ReadFilterCallbacks
	upstreamConnection  types.ClientConnection
	requestInfo         types.RequestInfo
	downstreamCallbacks DownstreamCallbacks

	protocols    types.Protocols
	routerConfig types.RouterConfig
	clusterName  string

	codec types.ServerStreamConnection
	// downstream requests
	activeSteams map[uint32]*activeStream
	// upstream requests
	upstreamRequests map[uint32]*upstreamRequest
}

func NewRPCProxy(config *v2.RpcProxy, clusterManager types.ClusterManager) RpcProxy {
	proxy := &rpcproxy{
		clusterManager:   clusterManager,
		requestInfo:      network.NewRequestInfo(),
		protocols:        sofarpc.DefaultProtocols(),
		upstreamRequests: make(map[uint32]*upstreamRequest),
	}

	proxy.routerConfig, _ = router.CreateRouteConfig(protocol.SofaRpc, config)

	proxy.downstreamCallbacks = &downstreamCallbacks{
		proxy: proxy,
	}

	return proxy
}

// types.StreamCallbacks
// types.StreamDecoder
type activeStream struct {
	streamId        uint32
	element         *list.Element
	proxy           *rpcproxy
	responseEncoder types.StreamEncoder
}

// types.StreamCallbacks
func (s *activeStream) OnResetStream(reason types.StreamResetReason) {
	s.responseEncoder.GetStream().RemoveCallbacks(s)
	delete(s.proxy.activeSteams, s.streamId)
}

func (s *activeStream) OnAboveWriteBufferHighWatermark() {}

func (s *activeStream) OnBelowWriteBufferLowWatermark() {}

// types.StreamDecoder
func (s *activeStream) DecodeHeaders(headers map[string]string, endStream bool) {
	//do some route by service name
	route := s.proxy.routerConfig.Route(headers)

	if route == nil || route.RouteRule() == nil {
		// no route
		s.proxy.onDataErr()

		return
	}

	err, pool := s.proxy.initializeUpstreamConnectionPool(route.RouteRule().ClusterName())

	if err != nil {
		s.proxy.onDataErr()

		return
	}

	s.proxy.upstreamRequests[s.streamId] = &upstreamRequest{
		streamId:    s.streamId,
		connPool:    pool,
		requestInfo: network.NewRequestInfo(),
	}

	return
}

func (s *activeStream) DecodeData(data types.IoBuffer, endStream bool) {}

func (s *activeStream) DecodeTrailers(trailers map[string]string) {}

func (s *activeStream) DecodeComplete(buf types.IoBuffer) {
	upstreamReq := s.proxy.upstreamRequests[s.streamId]

	upstreamReq.sendBuf = buf
	upstreamReq.connPool.NewStream(s.streamId, upstreamReq, upstreamReq)
}

// types.StreamCallbacks
// types.StreamDecoder
// types.PoolCallbacks
type upstreamRequest struct {
	streamId    uint32
	proxy       *rpcproxy
	host        types.Host
	requestInfo types.RequestInfo
	connPool    types.ConnectionPool
	encoder     types.StreamEncoder
	sendBuf     types.IoBuffer
}

// types.StreamCallbacks
func (r *upstreamRequest) OnResetStream(reason types.StreamResetReason) {
	r.requestInfo.SetResponseFlag(r.proxy.streamResetReasonToResponseFlag(reason))
	delete(r.proxy.upstreamRequests, r.streamId)
	r.proxy.onUpstreamReset(reason)
}

func (r *upstreamRequest) OnAboveWriteBufferHighWatermark() {}

func (r *upstreamRequest) OnBelowWriteBufferLowWatermark() {}

// types.StreamDecoder
func (r *upstreamRequest) DecodeHeaders(headers map[string]string, endStream bool) {}

func (r *upstreamRequest) DecodeData(data types.IoBuffer, endStream bool) {}

func (r *upstreamRequest) DecodeTrailers(trailers map[string]string) {}

func (r *upstreamRequest) DecodeComplete(data types.IoBuffer) {
	r.proxy.readCallbacks.Connection().Write(data)
	r.proxy.responseDecodeComplete(r)
}

func (r *upstreamRequest) responseDecodeComplete() {}

// types.PoolCallbacks
func (r *upstreamRequest) OnPoolFailure(streamId uint32, reason types.PoolFailureReason, host types.Host) {
	var resetReason types.StreamResetReason

	switch reason {
	case types.Overflow:
		resetReason = types.StreamOverflow
	case types.ConnectionFailure:
		resetReason = types.StreamConnectionFailed
	}

	r.OnResetStream(resetReason)
}

func (r *upstreamRequest) OnPoolReady(streamId uint32, encoder types.StreamEncoder, host types.Host) {
	r.encoder = encoder

	r.requestInfo.OnUpstreamHostSelected(host)
	r.encoder.GetStream().AddCallbacks(r)
	r.encoder.EncodeData(r.sendBuf, false)
}

func (p *rpcproxy) OnData(buf types.IoBuffer) types.FilterStatus {
	p.codec.Dispatch(buf)

	return types.StopIteration
}

func (p *rpcproxy) onDataErr() {
	if p.upstreamConnection != nil {
		p.upstreamConnection.Close(types.NoFlush, types.LocalClose)
	}

	if p.readCallbacks.Connection() != nil {
		p.readCallbacks.Connection().Close(types.NoFlush, types.LocalClose)
	}
}

//rpc realize upstream on event
func (p *rpcproxy) onDownstreamEvent(event types.ConnectionEvent) {
	if event.IsClose() {
		for _, upstreamRequest := range p.upstreamRequests {
			upstreamRequest.encoder.GetStream().ResetStream(types.StreamLocalReset)
		}
	}
}

func (p *rpcproxy) ReadDisableUpstream(disable bool) {
	// TODO
}

func (p *rpcproxy) ReadDisableDownstream(disable bool) {
	// TODO
}

func (p *rpcproxy) initializeUpstreamConnectionPool(clusterName string) (error, types.ConnectionPool) {
	clusterSnapshot := p.clusterManager.Get(clusterName, nil)

	if reflect.ValueOf(clusterSnapshot).IsNil() {
		p.requestInfo.SetResponseFlag(types.NoRouteFound)
		p.onInitFailure(NoRoute)

		return errors.New(fmt.Sprintf("unkown cluster %s", clusterName)), nil
	}

	clusterInfo := clusterSnapshot.ClusterInfo()
	clusterConnectionResource := clusterInfo.ResourceManager().ConnectionResource()

	if !clusterConnectionResource.CanCreate() {
		p.requestInfo.SetResponseFlag(types.UpstreamOverflow)
		p.onInitFailure(ResourceLimitExceeded)

		return errors.New(fmt.Sprintf("upstream overflow in cluster %s", clusterName)), nil
	}

	connPool := p.clusterManager.SofaRpcConnPoolForCluster(clusterName, nil)

	if connPool == nil {
		p.requestInfo.SetResponseFlag(types.NoHealthyUpstream)
		p.onInitFailure(NoHealthyUpstream)

		return errors.New(fmt.Sprintf("no healthy upstream in cluster %s", clusterName)), nil
	}

	// TODO: update upstream stats

	return nil, connPool
}

func (p *rpcproxy) onConnectionSuccess() {
	log.DefaultLogger.Debugf("new upstream connection %d created", p.upstreamConnection.Id())
}

func (p *rpcproxy) onInitFailure(reason UpstreamFailureReason) {
	p.readCallbacks.Connection().Close(types.NoFlush, types.LocalClose)
}

func (p *rpcproxy) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {
	p.readCallbacks = cb
	p.readCallbacks.Connection().AddConnectionCallbacks(p.downstreamCallbacks)

	p.codec = newServerStreamConnection(p.readCallbacks.Connection(), p)

	// TODO: set downstream connection stats
}

func (p *rpcproxy) NewStream(streamId uint32, responseEncoder types.StreamEncoder) types.StreamDecoder {
	stream := &activeStream{
		streamId:        streamId,
		proxy:           p,
		responseEncoder: responseEncoder,
	}

	stream.responseEncoder.GetStream().AddCallbacks(stream)
	p.activeSteams[streamId] = stream

	return stream
}

func (p *rpcproxy) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (p *rpcproxy) streamResetReasonToResponseFlag(reason types.StreamResetReason) types.ResponseFlag {
	switch reason {
	case types.StreamConnectionFailed:
		return types.UpstreamConnectionFailure

	}

	return 0
}

func (p *rpcproxy) onUpstreamReset(reason types.StreamResetReason) {

}

func (p *rpcproxy) responseDecodeComplete(r *upstreamRequest) {
	r.encoder.GetStream().RemoveCallbacks(r)
	delete(p.upstreamRequests, r.streamId)
}

// ConnectionCallbacks
type downstreamCallbacks struct {
	proxy *rpcproxy
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
