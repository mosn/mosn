package proxy

import (
	"fmt"
	"errors"
	"reflect"
	"container/list"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/router"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"sync"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
)

// types.ReadFilter
// types.ServerStreamConnectionCallbacks
type proxy struct {
	config              *v2.Proxy
	clusterManager      types.ClusterManager
	readCallbacks       types.ReadFilterCallbacks
	upstreamConnection  types.ClientConnection
	downstreamCallbacks DownstreamCallbacks

	routerConfig types.RouterConfig
	clusterName  string

	serverCodec types.ServerStreamConnection
	// downstream requests
	activeSteams *list.List
	// upstream requests
	upstreamRequests *list.List
	streamMux        sync.RWMutex
}

func NewProxy(config *v2.Proxy, clusterManager types.ClusterManager) Proxy {
	proxy := &proxy{
		config:           config,
		clusterManager:   clusterManager,
		activeSteams:     list.New(),
		upstreamRequests: list.New(),
	}

	proxy.routerConfig, _ = router.CreateRouteConfig(types.Protocol(config.DownstreamProtocol), config)
	proxy.downstreamCallbacks = &downstreamCallbacks{
		proxy: proxy,
	}

	return proxy
}

// types.StreamCallbacks
// types.StreamDecoder
type activeStream struct {
	element         *list.Element
	proxy           *proxy
	responseEncoder types.StreamEncoder
	upstreamRequest *upstreamRequest
}

// types.StreamCallbacks
func (s *activeStream) OnResetStream(reason types.StreamResetReason) {
	s.cleanup()
}

func (s *activeStream) OnAboveWriteBufferHighWatermark() {}

func (s *activeStream) OnBelowWriteBufferLowWatermark() {}

func (s *activeStream) cleanup() {
	s.proxy.streamMux.Lock()
	defer s.proxy.streamMux.Unlock()

	s.proxy.deleteActiveStream(s)
	s.proxy.deleteUpstreamRequest(s.upstreamRequest)
}

// types.StreamDecoder
func (s *activeStream) OnDecodeHeaders(headers map[string]string, endStream bool) {
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

	req := &upstreamRequest{
		activeStream: s,
		proxy:        s.proxy,
		connPool:     pool,
		requestInfo:  network.NewRequestInfo(),
	}
	req.element = s.proxy.upstreamRequests.PushBack(req)
	s.upstreamRequest = req

	req.connPool.NewStream(0, req, req)

	// todo: move this to pool ready
	// most simple case: just encode headers to upstream
	req.requestEncoder.EncodeHeaders(headers, endStream)

	return
}

func (s *activeStream) OnDecodeData(data types.IoBuffer, endStream bool) {
	s.upstreamRequest.requestEncoder.EncodeData(data, endStream)
}

func (s *activeStream) OnDecodeTrailers(trailers map[string]string) {
	s.upstreamRequest.requestEncoder.EncodeTrailers(trailers)
}

func (s *activeStream) OnDecodeComplete(buf types.IoBuffer) {}

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

	var connPool types.ConnectionPool

	// todo: refactor
	switch types.Protocol(s.proxy.config.UpstreamProtocol) {
	case protocol.SofaRpc:
		connPool = s.proxy.clusterManager.SofaRpcConnPoolForCluster(clusterName, nil)
	case protocol.Http2:
		connPool = s.proxy.clusterManager.HttpConnPoolForCluster(clusterName, 0, protocol.Http2, nil)
	default:
		connPool = s.proxy.clusterManager.HttpConnPoolForCluster(clusterName, 0, protocol.Http2, nil)
	}

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
	proxy          *proxy
	element        *list.Element
	activeStream   *activeStream
	host           types.Host
	requestInfo    types.RequestInfo
	connPool       types.ConnectionPool
	requestEncoder types.StreamEncoder
	sendBuf        types.IoBuffer
}

// types.StreamCallbacks
func (r *upstreamRequest) OnResetStream(reason types.StreamResetReason) {
	r.requestInfo.SetResponseFlag(r.proxy.streamResetReasonToResponseFlag(reason))
	r.cleanup()
	r.proxy.onUpstreamReset(reason)
}

func (r *upstreamRequest) OnAboveWriteBufferHighWatermark() {}

func (r *upstreamRequest) OnBelowWriteBufferLowWatermark() {}

// types.StreamDecoder
func (r *upstreamRequest) OnDecodeHeaders(headers map[string]string, endStream bool) {
	// most simple case: just encode headers to upstream
	r.activeStream.responseEncoder.EncodeHeaders(headers, endStream)

	if endStream {
		r.cleanup()
	}
}

func (r *upstreamRequest) OnDecodeData(data types.IoBuffer, endStream bool) {
	r.activeStream.responseEncoder.EncodeData(data, endStream)

	if endStream {
		r.cleanup()
	}
}

func (r *upstreamRequest) OnDecodeTrailers(trailers map[string]string) {
	r.activeStream.responseEncoder.EncodeTrailers(trailers)
	r.cleanup()
}

func (r *upstreamRequest) OnDecodeComplete(data types.IoBuffer) {}

func (r *upstreamRequest) cleanup() {
	r.proxy.streamMux.Lock()
	defer r.proxy.streamMux.Unlock()

	r.proxy.deleteActiveStream(r.activeStream)
	r.proxy.deleteUpstreamRequest(r)
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
	r.requestEncoder = encoder

	r.requestInfo.OnUpstreamHostSelected(host)
	r.requestEncoder.GetStream().AddCallbacks(r)
}

func (p *proxy) OnData(buf types.IoBuffer) types.FilterStatus {
	p.serverCodec.Dispatch(buf)

	return types.StopIteration
}

//rpc realize upstream on event
func (p *proxy) onDownstreamEvent(event types.ConnectionEvent) {
	if event.IsClose() {
		for urEle := p.upstreamRequests.Front(); urEle != nil; urEle = urEle.Next() {
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
		responseEncoder: responseEncoder,
	}

	stream.responseEncoder.GetStream().AddCallbacks(stream)
	stream.element = p.activeSteams.PushBack(stream)

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

func (p *proxy) onUpstreamReset(reason types.StreamResetReason) {
	// todo: update stats
}

func (p *proxy) deleteUpstreamRequest(r *upstreamRequest) {
	if r.requestEncoder != nil {
		r.requestEncoder.GetStream().RemoveCallbacks(r)
	}

	p.upstreamRequests.Remove(r.element)
}

func (p *proxy) deleteActiveStream(s *activeStream) {
	if s.responseEncoder != nil {
		s.responseEncoder.GetStream().RemoveCallbacks(s)
	}

	p.activeSteams.Remove(s.element)
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
