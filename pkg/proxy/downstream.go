package proxy

import (
	"strconv"
	"fmt"
	"time"
	"errors"
	"reflect"
	"container/list"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

// types.StreamCallbacks
// types.StreamDecoder
// types.FilterChainFactoryCallbacks
type activeStream struct {
	streamId uint32
	proxy    *proxy
	route    types.Route
	cluster  types.ClusterInfo
	element  *list.Element

	// ~~~ control args
	timeout    *ProxyTimeout
	retryState *retryState

	requestInfo      types.RequestInfo
	responseEncoder  types.StreamEncoder
	upstreamRequest  *upstreamRequest
	perRetryTimer    *timer
	globalRetryTimer *timer

	// ~~~ downstream request buf
	downstreamReqHeaders  map[string]string
	downstreamReqDataBuf  types.IoBuffer
	downstreamReqTrailers map[string]string

	// ~~~ downstream response buf
	downstreamRespHeaders  map[string]string
	downstreamRespDataBuf  types.IoBuffer
	downstreamRespTrailers map[string]string

	// ~~~ state
	// starts to send back downstream response, set on upstream response detected
	downstreamResponseStarted bool
	// upstream req sent
	upstreamRequestSent bool
	// downstream request received done
	downstreamRecvDone bool
	// 1. upstream response send done 2. done by a hijack response
	localProcessDone bool

	encoderFiltersStreaming bool

	decoderFiltersStreaming bool

	filterStage int

	// ~~~ filters
	encoderFilters []*activeStreamEncoderFilter
	decoderFilters []*activeStreamDecoderFilter
}

func newActiveStream(streamId uint32, proxy *proxy, responseEncoder types.StreamEncoder) *activeStream {
	stream := &activeStream{
		streamId:        streamId,
		proxy:           proxy,
		requestInfo:     network.NewRequestInfo(),
		responseEncoder: responseEncoder,
	}

	stream.responseEncoder.GetStream().AddCallbacks(stream)

	proxy.stats.DownstreamRequestTotal().Inc(1)
	proxy.stats.DownstreamRequestActive().Inc(1)
	proxy.listenerStats.DownstreamRequestTotal().Inc(1)
	proxy.listenerStats.DownstreamRequestActive().Inc(1)

	return stream
}

// types.StreamCallbacks
func (s *activeStream) OnResetStream(reason types.StreamResetReason) {
	s.proxy.stats.DownstreamRequestReset().Inc(1)
	s.proxy.listenerStats.DownstreamRequestReset().Inc(1)
	s.cleanStream()
}

func (s *activeStream) OnAboveWriteBufferHighWatermark() {}

func (s *activeStream) OnBelowWriteBufferLowWatermark() {}

// case 1: stream's lifecycle ends normally
// case 2: proxy ends stream in lifecycle
func (s *activeStream) endStream() {
	s.stopTimer()

	if s.responseEncoder != nil {
		if !s.downstreamRecvDone || !s.localProcessDone {
			// if downstream req received not done, or local proxy process not done by handle upstream response,
			// just mark it as done and reset stream as a failed case
			s.localProcessDone = true
			s.responseEncoder.GetStream().ResetStream(types.StreamLocalReset)
		}
	}

	s.cleanStream()

	// note: if proxy logic resets the stream, there maybe some underlying data in the conn.
	// we ignore this for now, fix as a todo
}

func (s *activeStream) cleanStream() {
	s.proxy.stats.DownstreamRequestActive().Dec(1)
	s.proxy.listenerStats.DownstreamRequestActive().Dec(1)

	for _, al := range s.proxy.accessLogs {
		al.Log(s.downstreamReqHeaders, nil, s.requestInfo)
	}

	log.DefaultLogger.Debugf(s.proxy.stats.String())
	log.DefaultLogger.Debugf(s.proxy.listenerStats.String())
	s.proxy.deleteActiveStream(s)
}

// types.StreamDecoder
func (s *activeStream) OnDecodeHeaders(headers map[string]string, endStream bool) {
	s.downstreamRecvDone = endStream
	s.downstreamReqHeaders = headers

	// todo: validate headers

	s.doDecodeHeaders(nil, headers, endStream)
}

func (s *activeStream) doDecodeHeaders(filter *activeStreamDecoderFilter, headers map[string]string, endStream bool) {
	if s.decodeHeaderFilters(filter, headers, endStream) {
		return
	}

	//do some route by service name
	route := s.proxy.routerConfig.Route(headers)

	if route == nil || route.RouteRule() == nil {
		// no route
		s.requestInfo.SetResponseFlag(types.NoRouteFound)
		s.sendHijackReply(404, nil)

		return
	}

	s.route = route
	s.requestInfo.SetRouteEntry(route.RouteRule())

	s.requestInfo.SetDownstreamLocalAddress(s.proxy.readCallbacks.Connection().LocalAddr())
	// todo: detect remote addr
	s.requestInfo.SetDownstreamRemoteAddress(s.proxy.readCallbacks.Connection().RemoteAddr())
	err, pool := s.initializeUpstreamConnectionPool(route.RouteRule().ClusterName())

	if err != nil {
		return
	}

	s.timeout = parseProxyTimeout(route, headers)
	s.retryState = newRetryState(route.RouteRule().Policy().RetryPolicy(), headers, s.cluster)

	//Build Request
	s.upstreamRequest = &upstreamRequest{
		activeStream: s,
		proxy:        s.proxy,
		connPool:     pool,
		requestInfo:  network.NewRequestInfo(),
	}

	s.upstreamRequest.encodeHeaders(headers, endStream)

	if endStream {
		s.onUpstreamRequestSent()
	}
}

func (s *activeStream) OnDecodeData(data types.IoBuffer, endStream bool) {
	s.requestInfo.SetBytesReceived(s.requestInfo.BytesReceived() + uint64(data.Len()))
	s.downstreamRecvDone = endStream

	// if active stream finished the lifecycle, just ignore further data
	if s.localProcessDone {
		return
	}

	s.doDecodeData(nil, data, endStream)
}

func (s *activeStream) doDecodeData(filter *activeStreamDecoderFilter, data types.IoBuffer, endStream bool) {
	if s.decodeDataFilters(filter, data, endStream) {
		return
	}

	shouldBufData := false
	if s.retryState != nil && s.retryState.retryOn {
		shouldBufData = true

		// todo: set a buf limit
	}

	if shouldBufData {
		buf := buffer.NewIoBuffer(data.Len())
		// todo: change to ReadAll @wugou
		buf.ReadFrom(data)

		if s.downstreamReqDataBuf == nil {
			s.downstreamReqDataBuf = buffer.NewIoBuffer(data.Len())
		}

		s.downstreamReqDataBuf.Append(buf.Bytes())
		s.upstreamRequest.encodeData(s.downstreamReqDataBuf, endStream)
	} else {
		s.upstreamRequest.encodeData(data, endStream)
	}

	if endStream {
		s.onUpstreamRequestSent()
	}
}

func (s *activeStream) OnDecodeTrailers(trailers map[string]string) {
	s.downstreamRecvDone = true

	// if active stream finished the lifecycle, just ignore further data
	if s.localProcessDone {
		return
	}

	s.doDecodeTrailers(nil, trailers)
}

func (s *activeStream) doDecodeTrailers(filter *activeStreamDecoderFilter, trailers map[string]string) {
	if s.decodeTrailersFilters(filter, trailers) {
		return
	}

	s.downstreamReqTrailers = trailers
	s.upstreamRequest.encodeTrailers(trailers)
	s.onUpstreamRequestSent()
}

func (s *activeStream) onUpstreamRequestSent() {
	s.upstreamRequestSent = true
	s.requestInfo.SetRequestReceivedDuration(time.Now())

	if s.upstreamRequest != nil {
		// setup per try timeout timer
		s.setupPerTryTimeout()

		// setup global timeout timer
		if s.timeout.GlobalTimeout > 0 {
			if s.globalRetryTimer != nil {
				s.globalRetryTimer.stop()
			}

			s.globalRetryTimer = newTimer(s.onResponseTimeout, s.timeout.GlobalTimeout)
		}
	}
}

func (s *activeStream) onResponseTimeout() {
	s.globalRetryTimer = nil

	if s.upstreamRequest != nil {
		s.upstreamRequest.resetStream()
	}

	s.onUpstreamReset(UpstreamGlobalTimeout, types.StreamLocalReset)
}

func (s *activeStream) setupPerTryTimeout() {
	timeout := s.timeout

	if timeout.TryTimeout > 0 {
		if s.perRetryTimer != nil {
			s.perRetryTimer.stop()
		}

		s.perRetryTimer = newTimer(s.onPerTryTimeout, timeout.TryTimeout*time.Second)
		s.perRetryTimer.start()
	}
}

func (s *activeStream) onPerTryTimeout() {
	s.perRetryTimer = nil

	s.upstreamRequest.resetStream()
	s.upstreamRequest.requestInfo.SetResponseFlag(types.UpstreamRequestTimeout)

	s.onUpstreamReset(UpstreamPerTryTimeout, types.StreamLocalReset)
}

func (s *activeStream) initializeUpstreamConnectionPool(clusterName string) (error, types.ConnectionPool) {
	clusterSnapshot := s.proxy.clusterManager.Get(clusterName, nil)

	if reflect.ValueOf(clusterSnapshot).IsNil() {
		// no available cluster
		s.requestInfo.SetResponseFlag(types.NoRouteFound)
		s.sendHijackReply(404, nil)

		return errors.New(fmt.Sprintf("unkown cluster %s", clusterName)), nil
	}

	clusterInfo := clusterSnapshot.ClusterInfo()
	s.cluster = clusterInfo
	clusterConnectionResource := clusterInfo.ResourceManager().ConnectionResource()

	if !clusterConnectionResource.CanCreate() {
		s.requestInfo.SetResponseFlag(types.UpstreamOverflow)
		s.sendHijackReply(503, nil)

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
		s.requestInfo.SetResponseFlag(types.NoHealthyUpstream)
		s.sendHijackReply(500, nil)

		return errors.New(fmt.Sprintf("no healthy upstream in cluster %s", clusterName)), nil
	}

	// TODO: update upstream stats

	return nil, connPool
}

// ~~~ active stream encoder wrapper

func (s *activeStream) encodeHeaders(headers map[string]string, endStream bool) {
	s.localProcessDone = endStream

	s.doEncodeHeaders(nil, headers, endStream)
}

func (s *activeStream) doEncodeHeaders(filter *activeStreamEncoderFilter, headers map[string]string, endStream bool) {
	if s.encodeHeaderFilters(filter, headers, endStream) {
		return
	}

	s.responseEncoder.EncodeHeaders(headers, endStream)

	if endStream {
		s.endStream()
	}
}

func (s *activeStream) encodeData(data types.IoBuffer, endStream bool) {
	s.localProcessDone = endStream

	s.doEncodeData(nil, data, endStream)
}

func (s *activeStream) doEncodeData(filter *activeStreamEncoderFilter, data types.IoBuffer, endStream bool) {
	if s.encodeDataFilters(filter, data, endStream) {
		return
	}

	s.responseEncoder.EncodeData(data, endStream)

	s.requestInfo.SetBytesSent(s.requestInfo.BytesSent() + uint64(data.Len()))

	if endStream {
		s.endStream()
	}
}

func (s *activeStream) encodeTrailers(trailers map[string]string) {
	s.localProcessDone = true

	s.doEncodeTrailers(nil, trailers)
}

func (s *activeStream) doEncodeTrailers(filter *activeStreamEncoderFilter, trailers map[string]string) {
	if s.encodeTrailersFilters(filter, trailers) {
		return
	}

	s.responseEncoder.EncodeTrailers(trailers)
	s.endStream()
}

// ~~~ upstream event handler

func (s *activeStream) onUpstreamHeaders(headers map[string]string, endStream bool) {
	// check retry
	if s.retryState != nil {
		shouldRetry := s.retryState.shouldRetry(headers, "")
		if shouldRetry && s.setupRetry(endStream) {
			if s.perRetryTimer != nil {
				s.perRetryTimer.stop()
			}

			s.perRetryTimer = s.retryState.scheduleRetry(s.doRetry)
			return
		}

		s.retryState = nil
	}

	s.requestInfo.SetResponseReceivedDuration(time.Now())

	s.downstreamResponseStarted = true

	if endStream {
		s.onUpstreamResponseRecvDone()
	}

	// todo: insert proxy headers

	s.encodeHeaders(headers, endStream)
}

func (s *activeStream) onUpstreamData(data types.IoBuffer, endStream bool) {
	if endStream {
		s.onUpstreamResponseRecvDone()
	}

	s.encodeData(data, endStream)
}

func (s *activeStream) onUpstreamTrailers(trailers map[string]string) {
	s.onUpstreamResponseRecvDone()

	s.encodeTrailers(trailers)
}

func (s *activeStream) onUpstreamResponseRecvDone() {
	if !s.upstreamRequestSent {
		s.upstreamRequest.resetStream()
	}

	// todo: stats
	// todo: logs

	s.stopTimer()
}

func (s *activeStream) onUpstreamReset(urtype UpstreamResetType, reason types.StreamResetReason) {
	// todo: update stats

	// see if we need a retry
	if urtype != UpstreamGlobalTimeout &&
		s.downstreamResponseStarted && s.retryState != nil {
		if s.retryState.shouldRetry(nil, reason) && s.setupRetry(true) {
			if s.perRetryTimer != nil {
				s.perRetryTimer.stop()
			}

			s.perRetryTimer = s.retryState.scheduleRetry(s.doRetry)
			return
		}
	}

	// If we have not yet sent anything downstream, send a response with an appropriate status code.
	// Otherwise just reset the ongoing response.
	if s.downstreamResponseStarted {
		s.stopTimer()
		s.resetStream()
	} else {
		s.stopTimer()

		var code int
		if urtype == UpstreamGlobalTimeout || urtype == UpstreamPerTryTimeout {
			s.requestInfo.SetResponseFlag(types.UpstreamRequestTimeout)
			code = 504
		} else {
			reasonFlag := s.proxy.streamResetReasonToResponseFlag(reason)
			s.requestInfo.SetResponseFlag(reasonFlag)
			code = 500
		}

		s.sendHijackReply(code, nil)
	}
}

func (s *activeStream) setupRetry(endStream bool) bool {
	if !s.upstreamRequestSent {
		return false
	}

	if !endStream {
		s.upstreamRequest.resetStream()
	}

	s.upstreamRequest = nil

	return true
}

func (s *activeStream) doRetry() {
	err, pool := s.initializeUpstreamConnectionPool(s.cluster.Name())

	if err != nil {
		return
	}

	s.upstreamRequest = &upstreamRequest{
		activeStream: s,
		proxy:        s.proxy,
		connPool:     pool,
		requestInfo:  network.NewRequestInfo(),
	}

	s.upstreamRequest.encodeHeaders(nil,
		s.downstreamReqDataBuf != nil && s.downstreamReqTrailers != nil)

	if s.upstreamRequest != nil {
		if s.downstreamReqDataBuf != nil {
			buf := buffer.NewIoBuffer(s.downstreamReqDataBuf.Len())
			buf.ReadFrom(s.downstreamReqDataBuf)
			s.upstreamRequest.encodeData(buf, s.downstreamReqTrailers == nil)
		}

		if s.downstreamReqTrailers != nil {
			s.upstreamRequest.encodeTrailers(s.downstreamReqTrailers)
		}

		// setup per try timeout timer
		s.setupPerTryTimeout()
	}
}

func (s *activeStream) resetStream() {
	s.endStream()
}

func (s *activeStream) sendHijackReply(code int, headers map[string]string) {
	if headers == nil {
		headers = make(map[string]string)
	}

	headers[types.HeaderStatus] = strconv.Itoa(code)
	s.encodeHeaders(headers, true)
}

func (s *activeStream) stopTimer() {
	if s.perRetryTimer != nil {
		s.perRetryTimer.stop()
		s.perRetryTimer = nil
	}

	if s.globalRetryTimer != nil {
		s.globalRetryTimer.stop()
		s.globalRetryTimer = nil
	}
}

func (s *activeStream) setBufferLimit(bufferLimit uint32) {
	// todo
}

func (s *activeStream) AddStreamDecoderFilter(filter types.StreamDecoderFilter) {
	sf := newActiveStreamDecoderFilter(len(s.decoderFilters), s, filter)
	s.decoderFilters = append(s.decoderFilters, sf)
}

func (s *activeStream) AddStreamEncoderFilter(filter types.StreamEncoderFilter) {
	sf := newActiveStreamEncoderFilter(len(s.encoderFilters), s, filter)
	s.encoderFilters = append(s.encoderFilters, sf)
}
