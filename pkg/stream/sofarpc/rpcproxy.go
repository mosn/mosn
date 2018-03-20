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
	"container/list"
)

// 实现 sofa RPC 的 反向代理

// types.ReadFilter
// types.ServerStreamConnectionCallbacks
type rpcproxy struct {
	clusterManager      types.ClusterManager
	readCallbacks       types.ReadFilterCallbacks
	upstreamConnection  types.ClientConnection
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
		protocols:        sofarpc.DefaultProtocols(),
		activeSteams:     make(map[uint32]*activeStream),
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
	s.proxy.onStreamComplete(s.streamId)
}

func (s *activeStream) OnAboveWriteBufferHighWatermark() {}

func (s *activeStream) OnBelowWriteBufferLowWatermark() {}

// types.StreamDecoder
func (s *activeStream) DecodeHeaders(headers map[string]string, endStream bool) {
	//do some route by service name
	route := s.proxy.routerConfig.Route(headers)

	if route == nil || route.RouteRule() == nil {
		// no route
		s.onDataErr(nil)

		return
	}

	err, pool := s.initializeUpstreamConnectionPool(route.RouteRule().ClusterName())

	if err != nil {
		s.onDataErr(err)

		return
	}

	s.proxy.upstreamRequests[s.streamId] = &upstreamRequest{
		streamId:    s.streamId,
		proxy:       s.proxy,
		connPool:    pool,
		requestInfo: network.NewRequestInfo(),
	}

	return
}

func (s *activeStream) DecodeData(data types.IoBuffer, endStream bool) {}

func (s *activeStream) DecodeTrailers(trailers map[string]string) {}

func (s *activeStream) DecodeComplete(buf types.IoBuffer) {
	if buf != nil {
		upstreamReq := s.proxy.upstreamRequests[s.streamId]
		upstreamReq.sendBuf = buf
		upstreamReq.connPool.NewStream(s.streamId, upstreamReq, upstreamReq)
	}
}

func (s *activeStream) initializeUpstreamConnectionPool(clusterName string) (error, types.ConnectionPool) {
	// todo: we just reset downstream stream on route failed, confirm sofa rpc logic
	clusterSnapshot := s.proxy.clusterManager.Get(clusterName, nil)

	if reflect.ValueOf(clusterSnapshot).IsNil() {
		//s.requestInfo.SetResponseFlag(types.NoRouteFound)
		s.onInitFailure(NoRoute)

		return errors.New(fmt.Sprintf("unkown cluster %s", clusterName)), nil
	}

	clusterInfo := clusterSnapshot.ClusterInfo()
	clusterConnectionResource := clusterInfo.ResourceManager().ConnectionResource()

	if !clusterConnectionResource.CanCreate() {
		//p.requestInfo.SetResponseFlag(types.UpstreamOverflow)
		s.onInitFailure(ResourceLimitExceeded)

		return errors.New(fmt.Sprintf("upstream overflow in cluster %s", clusterName)), nil
	}

	connPool := s.proxy.clusterManager.SofaRpcConnPoolForCluster(clusterName, nil)

	if connPool == nil {
		//p.requestInfo.SetResponseFlag(types.NoHealthyUpstream)
		s.onInitFailure(NoHealthyUpstream)

		return errors.New(fmt.Sprintf("no healthy upstream in cluster %s", clusterName)), nil
	}

	// TODO: update upstream stats

	return nil, connPool
}

func (s *activeStream) onInitFailure(reason UpstreamFailureReason) {
	// todo: convert upstream reason to reset reason
	s.responseEncoder.GetStream().ResetStream(types.StreamConnectionFailed)
}

func (s *activeStream) onDataErr(err error) {
	// todo: convert error to reset reason
	s.responseEncoder.GetStream().ResetStream(types.StreamConnectionFailed)
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
	r.proxy.onUpstreamReset(r.streamId, reason)
}

func (r *upstreamRequest) OnAboveWriteBufferHighWatermark() {}

func (r *upstreamRequest) OnBelowWriteBufferLowWatermark() {}

// types.StreamDecoder
func (r *upstreamRequest) DecodeHeaders(headers map[string]string, endStream bool) {}

func (r *upstreamRequest) DecodeData(data types.IoBuffer, endStream bool) {}

func (r *upstreamRequest) DecodeTrailers(trailers map[string]string) {}

func (r *upstreamRequest) DecodeComplete(data types.IoBuffer) {
	r.proxy.readCallbacks.Connection().Write(data)

	r.proxy.onStreamComplete(r.streamId)
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

func (p *rpcproxy) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {
	p.readCallbacks = cb
	p.readCallbacks.Connection().AddConnectionCallbacks(p.downstreamCallbacks)

	p.codec = newServerStreamConnection(p.readCallbacks.Connection(), p)

	// TODO: set downstream connection stats
}

func (p *rpcproxy) OnGoAway() {}

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

func (p *rpcproxy) onUpstreamReset(streamId uint32, reason types.StreamResetReason) {
	p.onStreamComplete(streamId)

	// todo: update stats
}

func (p *rpcproxy) deleteUpstreamRequest(r *upstreamRequest) {
	r.encoder.GetStream().RemoveCallbacks(r)
	delete(p.upstreamRequests, r.streamId)
}

func (p *rpcproxy) deleteActiveStream(s *activeStream) {
	s.responseEncoder.GetStream().RemoveCallbacks(s)
	delete(p.activeSteams, s.streamId)
}

func (p *rpcproxy) onStreamComplete(streamId uint32) {
	if as, ok := p.activeSteams[streamId]; ok {
		p.deleteActiveStream(as)
	}

	if ur, ok := p.upstreamRequests[streamId]; ok {
		p.deleteUpstreamRequest(ur)
	}
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
