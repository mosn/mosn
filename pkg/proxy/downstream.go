package proxy

import (
	"container/list"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// types.StreamEventListener
// types.StreamDecoder
// types.FilterChainFactoryCallbacks
type activeStream struct {
	streamId string
	proxy    *proxy
	route    types.Route
	cluster  types.ClusterInfo
	element  *list.Element

	// flow control
	bufferLimit        uint32
	highWatermarkCount int

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
	downstreamRespHeaders  interface{}
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

	watermarkCallbacks types.DownstreamWatermarkEventListener

	// ~~~ filters
	encoderFilters []*activeStreamEncoderFilter
	decoderFilters []*activeStreamDecoderFilter

	logger log.Logger
}

func newActiveStream(streamId string, proxy *proxy, responseEncoder types.StreamEncoder) *activeStream {
	var stream *activeStream

	if buf := activeStreamPool.Take(); buf == nil {
		stream = &activeStream{}
	} else {
		stream = buf.(*activeStream)
	}

	stream.streamId = streamId
	stream.proxy = proxy
	stream.requestInfo = network.NewRequestInfo()
	stream.responseEncoder = responseEncoder
	stream.responseEncoder.GetStream().AddEventListener(stream)

	stream.logger = log.ByContext(proxy.context)

	proxy.stats.DownstreamRequestTotal().Inc(1)
	proxy.stats.DownstreamRequestActive().Inc(1)
	proxy.listenerStats.DownstreamRequestTotal().Inc(1)
	proxy.listenerStats.DownstreamRequestActive().Inc(1)

	return stream
}

// types.StreamEventListener
func (s *activeStream) OnResetStream(reason types.StreamResetReason) {
	s.proxy.stats.DownstreamRequestReset().Inc(1)
	s.proxy.listenerStats.DownstreamRequestReset().Inc(1)
	s.cleanStream()
}

func (s *activeStream) OnAboveWriteBufferHighWatermark() {
	s.callHighWatermarkCallbacks()
}

func (s *activeStream) callHighWatermarkCallbacks() {
	s.upstreamRequest.requestEncoder.GetStream().ReadDisable(true)
	s.highWatermarkCount++

	if s.watermarkCallbacks != nil {
		s.watermarkCallbacks.OnAboveWriteBufferHighWatermark()
	}
}

func (s *activeStream) OnBelowWriteBufferLowWatermark() {
	s.callLowWatermarkCallbacks()
}

func (s *activeStream) callLowWatermarkCallbacks() {
	s.upstreamRequest.requestEncoder.GetStream().ReadDisable(false)
	s.highWatermarkCount--

	if s.watermarkCallbacks != nil {
		s.watermarkCallbacks.OnBelowWriteBufferLowWatermark()
	}
}

// case 1: stream's lifecycle ends normally
// case 2: proxy ends stream in lifecycle
func (s *activeStream) endStream() {
	s.stopTimer()
	var isReset bool
	if s.responseEncoder != nil {
		if !s.downstreamRecvDone || !s.localProcessDone {
			// if downstream req received not done, or local proxy process not done by handle upstream response,
			// just mark it as done and reset stream as a failed case
			s.localProcessDone = true
			s.responseEncoder.GetStream().ResetStream(types.StreamLocalReset)
			isReset = true
		}
	}
	if !isReset {
		s.cleanStream()
	}
	// note: if proxy logic resets the stream, there maybe some underlying data in the conn.
	// we ignore this for now, fix as a todo
}

func (s *activeStream) cleanStream() {
	s.proxy.stats.DownstreamRequestActive().Dec(1)
	s.proxy.listenerStats.DownstreamRequestActive().Dec(1)

	var downstreamRespHeadersMap map[string]string

	if v, ok := s.downstreamRespHeaders.(map[string]string); ok {
		downstreamRespHeadersMap = v
	}

	if s.proxy != nil && s.proxy.accessLogs != nil {

		for _, al := range s.proxy.accessLogs {
			al.Log(s.downstreamReqHeaders, downstreamRespHeadersMap, s.requestInfo)
		}
	}

	s.proxy.deleteActiveStream(s)
}

// types.StreamDecoder
func (s *activeStream) OnDecodeHeaders(headers map[string]string, endStream bool) {
	s.downstreamRecvDone = endStream
	s.downstreamReqHeaders = headers

	s.doDecodeHeaders(nil, headers, endStream)
}

func (s *activeStream) doDecodeHeaders(filter *activeStreamDecoderFilter, headers map[string]string, endStream bool) {
	if s.decodeHeaderFilters(filter, headers, endStream) {
		return
	}

	//Get some route by service name
	route, clusterKey := s.proxy.routerConfig.Route(headers)
	// route,routeKey:= s.proxy.routerConfig.Route(headers)

	if route == nil || route.RouteRule() == nil {
		// no route
		s.requestInfo.SetResponseFlag(types.NoRouteFound)

		s.sendHijackReply(types.RouterUnavailableCode, headers)

		return
	}

	s.route = route
	s.requestInfo.SetRouteEntry(route.RouteRule())

	s.requestInfo.SetDownstreamLocalAddress(s.proxy.readCallbacks.Connection().LocalAddr())
	// todo: detect remote addr
	s.requestInfo.SetDownstreamRemoteAddress(s.proxy.readCallbacks.Connection().RemoteAddr())
	err, pool := s.initializeUpstreamConnectionPool(route.RouteRule().ClusterName(clusterKey))

	if err != nil {
		log.DefaultLogger.Errorf("initialize Upstream Connection Pool error, request can't be proxyed")
		return
	}

	s.timeout = parseProxyTimeout(route, headers)
	s.retryState = newRetryState(route.RouteRule().Policy().RetryPolicy(), headers, s.cluster)

	//Build Request
	s.upstreamRequest = &upstreamRequest{
		activeStream: s,
		proxy:        s.proxy,
		connPool:     pool,
	}

	if endStream {
		s.onUpstreamRequestSent()
	}
	//Call upstream's encode header method to build upstream's request
	s.upstreamRequest.encodeHeaders(headers, endStream)
}

func (s *activeStream) OnDecodeData(data types.IoBuffer, endStream bool) {
	s.requestInfo.SetBytesReceived(s.requestInfo.BytesReceived() + uint64(data.Len()))
	s.downstreamRecvDone = endStream

	s.doDecodeData(nil, data, endStream)
}

func (s *activeStream) doDecodeData(filter *activeStreamDecoderFilter, data types.IoBuffer, endStream bool) {
	// if active stream finished the lifecycle, just ignore further data
	if s.localProcessDone {
		return
	}

	if s.decodeDataFilters(filter, data, endStream) {
		return
	}

	shouldBufData := false
	if s.retryState != nil && s.retryState.retryOn {
		shouldBufData = true

		// todo: set a buf limit
	}

	if endStream {
		s.onUpstreamRequestSent()
	}

	if shouldBufData {
		copied := data.Clone()

		if s.downstreamReqDataBuf != data {
			// not in on decodeData continue decode context
			if s.downstreamReqDataBuf == nil {
				s.downstreamReqDataBuf = buffer.NewIoBuffer(data.Len())
			}

			s.downstreamReqDataBuf.ReadFrom(data)
		}

		// use a copy when we need to reuse buffer later
		s.upstreamRequest.encodeData(copied, endStream)
	} else {
		s.upstreamRequest.encodeData(data, endStream)
	}
}

func (s *activeStream) OnDecodeTrailers(trailers map[string]string) {
	s.downstreamRecvDone = true

	s.doDecodeTrailers(nil, trailers)
}

func (s *activeStream) OnDecodeError(err error,headers map[string]string) {
	// todo: enrich headers' information to do some hijack
	//Check headers' info to do hijack
	switch err.Error() {
	case types.CodecException:
		s.sendHijackReply(types.CodecExceptionCode, headers)
	case types.DeserializeException:
		s.sendHijackReply(types.DeserialExceptionCode, headers)
	default:
		s.sendHijackReply(types.UnknownCode, headers)
	}
	return
}

func (s *activeStream) doDecodeTrailers(filter *activeStreamDecoderFilter, trailers map[string]string) {
	// if active stream finished the lifecycle, just ignore further data
	if s.localProcessDone {
		return
	}

	if s.decodeTrailersFilters(filter, trailers) {
		return
	}

	s.downstreamReqTrailers = trailers
	s.onUpstreamRequestSent()
	s.upstreamRequest.encodeTrailers(trailers)
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
	s.onUpstreamReset(UpstreamPerTryTimeout, types.StreamLocalReset)
}

func (s *activeStream) initializeUpstreamConnectionPool(clusterName string) (error, types.ConnectionPool) {
	clusterSnapshot := s.proxy.clusterManager.Get(clusterName, nil)

	if reflect.ValueOf(clusterSnapshot).IsNil() {
		// no available cluster
		log.DefaultLogger.Errorf("cluster snapshot is nil, cluster name is: %s",clusterName)
		s.requestInfo.SetResponseFlag(types.NoRouteFound)
		s.sendHijackReply(types.RouterUnavailableCode, s.downstreamReqHeaders)

		return errors.New(fmt.Sprintf("unkown cluster %s", clusterName)), nil
	}

	clusterInfo := clusterSnapshot.ClusterInfo()
	s.cluster = clusterInfo
	clusterConnectionResource := clusterInfo.ResourceManager().ConnectionResource()

	if !clusterConnectionResource.CanCreate() {
		log.DefaultLogger.Errorf("cluster Connection Resource can't create, cluster name is %s",clusterName)
		
		s.requestInfo.SetResponseFlag(types.UpstreamOverflow)
		s.sendHijackReply(types.UpstreamOverFlowCode, s.downstreamReqHeaders)

		return errors.New(fmt.Sprintf("upstream overflow in cluster %s", clusterName)), nil
	}

	var connPool types.ConnectionPool

	// todo: refactor
	switch types.Protocol(s.proxy.config.UpstreamProtocol) {
	case protocol.SofaRpc:
		connPool = s.proxy.clusterManager.SofaRpcConnPoolForCluster(clusterName, nil)
	case protocol.Http2:
		connPool = s.proxy.clusterManager.HttpConnPoolForCluster(clusterName, protocol.Http2, nil)
	default:
		connPool = s.proxy.clusterManager.HttpConnPoolForCluster(clusterName, protocol.Http2, nil)
	}

	if connPool == nil {
		s.requestInfo.SetResponseFlag(types.NoHealthyUpstream)
		s.sendHijackReply(types.NoHealthUpstreamCode, s.downstreamReqHeaders)

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

func (s *activeStream) doEncodeHeaders(filter *activeStreamEncoderFilter, headers interface{}, endStream bool) {
	if s.encodeHeaderFilters(filter, headers, endStream) {
		return
	}

	//Currently, just log the error
	if err := s.responseEncoder.EncodeHeaders(headers, endStream); err != nil {
		s.logger.Errorf("[downstream] decode headers error, %s", err)
	}

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
	s.downstreamRespHeaders = headers
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
			code = types.TimeoutExceptionCode
		} else {
			reasonFlag := s.proxy.streamResetReasonToResponseFlag(reason)
			s.requestInfo.SetResponseFlag(reasonFlag)
			code = types.NoHealthUpstreamCode
		}

		s.sendHijackReply(code, s.downstreamReqHeaders)
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
	}

	s.upstreamRequest.encodeHeaders(s.downstreamReqHeaders,
		s.downstreamReqDataBuf != nil && s.downstreamReqTrailers != nil)

	if s.upstreamRequest != nil {
		if s.downstreamReqDataBuf != nil {
			// make a data copy to retry
			copied := s.downstreamReqDataBuf.Clone()
			s.upstreamRequest.encodeData(copied, s.downstreamReqTrailers == nil)
		}

		if s.downstreamReqTrailers != nil {
			s.upstreamRequest.encodeTrailers(s.downstreamReqTrailers)
		}

		// setup per try timeout timer
		s.setupPerTryTimeout()
	}
}

func (s *activeStream) onUpstreamAboveWriteBufferHighWatermark() {
	s.responseEncoder.GetStream().ReadDisable(true)
}

func (s *activeStream) onUpstreamBelowWriteBufferHighWatermark() {
	s.responseEncoder.GetStream().ReadDisable(false)
}

func (s *activeStream) resetStream() {
	s.endStream()
}

func (s *activeStream) sendHijackReply(code int, headers map[string]string) {
	if headers == nil {
		headers = make(map[string]string, 5)
	}
	log.DefaultLogger.Debugf("downstream send hijack reply,with code %d",code)

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
	s.bufferLimit = bufferLimit

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

func (s *activeStream) reset() {
	s.streamId = ""
	s.proxy = nil
	s.route = nil
	s.cluster = nil
	s.element = nil
	s.bufferLimit = 0
	s.highWatermarkCount = 0
	s.timeout = nil
	s.retryState = nil
	s.requestInfo = nil
	s.responseEncoder = nil
	s.upstreamRequest = nil
	s.perRetryTimer = nil
	s.globalRetryTimer = nil
	s.downstreamRespHeaders = nil
	s.downstreamReqDataBuf = nil
	s.downstreamReqTrailers = nil
	s.downstreamRespHeaders = nil
	s.downstreamRespDataBuf = nil
	s.downstreamRespTrailers = nil
	s.downstreamResponseStarted = false
	s.upstreamRequestSent = false
	s.downstreamRecvDone = false
	s.localProcessDone = false
	s.encoderFiltersStreaming = false
	s.decoderFiltersStreaming = false
	s.filterStage = 0
	s.watermarkCallbacks = nil
	s.encoderFilters = s.encoderFilters[:0]
	s.decoderFilters = s.decoderFilters[:0]
}
