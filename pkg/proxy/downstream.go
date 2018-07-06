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
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// types.StreamEventListener
// types.StreamReceiver
// types.FilterChainFactoryCallbacks
// Downstream stream, as a controller to handle downstream and upstream proxy flow
type downStream struct {
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

	requestInfo     types.RequestInfo
	responseSender  types.StreamSender
	upstreamRequest *upstreamRequest
	perRetryTimer   *timer
	responseTimer   *timer

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
	streamRoundTripDone      bool
	senderFiltersStreaming   bool
	receiverFiltersStreaming bool

	filterStage int

	// ~~~ filters
	senderFilters   []*activeStreamSenderFilter
	receiverFilters []*activeStreamReceiverFilter

	// mux for downstream-upstream flow
	mux sync.Mutex

	logger log.Logger
}

func newActiveStream(streamId string, proxy *proxy, responseSender types.StreamSender) *downStream {
	stream := &downStream{}

	stream.streamId = streamId
	stream.proxy = proxy
	stream.requestInfo = network.NewRequestInfo()
	stream.responseSender = responseSender
	stream.responseSender.GetStream().AddEventListener(stream)

	stream.logger = log.ByContext(proxy.context)

	proxy.stats.DownstreamRequestTotal().Inc(1)
	proxy.stats.DownstreamRequestActive().Inc(1)
	proxy.listenerStats.DownstreamRequestTotal().Inc(1)
	proxy.listenerStats.DownstreamRequestActive().Inc(1)

	return stream
}

// case 1: downstream's lifecycle ends normally
// case 2: downstream got reset. See downStream.resetStream for more detail
func (s *downStream) endStream() {
	var isReset bool
	if s.responseSender != nil {
		if !s.downstreamRecvDone || !s.streamRoundTripDone {
			// if downstream req received not done, or local proxy process not done by handle upstream response,
			// just mark it as done and reset stream as a failed case
			s.streamRoundTripDone = true
			s.responseSender.GetStream().ResetStream(types.StreamLocalReset)
			isReset = true
		}
	}

	if !isReset {
		s.cleanStream()
	}
	// note: if proxy logic resets the stream, there maybe some underlying data in the conn.
	// we ignore this for now, fix as a todo
}

// Clean up on the very end of the stream: end stream or reset stream
// Resources to clean up / reset:
// 	+ upstream request
// 	+ all timers
// 	+ all filters
//  + remove stream in proxy context
func (s *downStream) cleanStream() {
	// reset corresponding upstream stream
	if s.upstreamRequest != nil {
		s.upstreamRequest.resetStream()
	}

	// clean up timers
	s.cleanUp()

	// tell filters it's time to destroy
	for _, ef := range s.senderFilters {
		ef.filter.OnDestroy()
	}

	for _, ef := range s.receiverFilters {
		ef.filter.OnDestroy()
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	if s.upstreamRequest == nil || s.streamRoundTripDone {
		// countdown metrics
		s.proxy.stats.DownstreamRequestActive().Dec(1)
		s.proxy.listenerStats.DownstreamRequestActive().Dec(1)

		// access log
		if s.proxy != nil && s.proxy.accessLogs != nil {
			var downstreamRespHeadersMap map[string]string

			if v, ok := s.downstreamRespHeaders.(map[string]string); ok {
				downstreamRespHeadersMap = v
			}

			for _, al := range s.proxy.accessLogs {
				al.Log(s.downstreamReqHeaders, downstreamRespHeadersMap, s.requestInfo)
			}
		}

		// delete stream
		s.proxy.deleteActiveStream(s)
	}
}

// types.StreamEventListener
// Called by stream layer normally
func (s *downStream) OnResetStream(reason types.StreamResetReason) {
	s.proxy.stats.DownstreamRequestReset().Inc(1)
	s.proxy.listenerStats.DownstreamRequestReset().Inc(1)
	s.cleanStream()
}

// types.StreamReceiver
func (s *downStream) OnReceiveHeaders(headers map[string]string, endStream bool) {
	s.downstreamRecvDone = endStream
	s.downstreamReqHeaders = headers

	s.doReceiveHeaders(nil, headers, endStream)
}

func (s *downStream) doReceiveHeaders(filter *activeStreamReceiverFilter, headers map[string]string, endStream bool) {
	if s.runReceiveHeadersFilters(filter, headers, endStream) {
		return
	}

	//Get some route by service name
	log.StartLogger.Tracef("before active stream route")
	route := s.proxy.routers.Route(headers, 1)

	if route == nil || route.RouteRule() == nil {
		// no route
		log.StartLogger.Warnf("no route to init upstream,headers = %v", headers)
		s.requestInfo.SetResponseFlag(types.NoRouteFound)

		s.sendHijackReply(types.RouterUnavailableCode, headers)

		return
	}
	log.StartLogger.Tracef("get route : %v,clusterName=%v", route, route.RouteRule().ClusterName())

	s.route = route

	s.requestInfo.SetRouteEntry(route.RouteRule())
	s.requestInfo.SetDownstreamLocalAddress(s.proxy.readCallbacks.Connection().LocalAddr())
	// todo: detect remote addr
	s.requestInfo.SetDownstreamRemoteAddress(s.proxy.readCallbacks.Connection().RemoteAddr())

	// active realize loadbalancer ctx
	log.StartLogger.Tracef("before initializeUpstreamConnectionPool")
	err, pool := s.initializeUpstreamConnectionPool(route.RouteRule().ClusterName(), s)

	if err != nil {
		log.DefaultLogger.Errorf("initialize Upstream Connection Pool error, request can't be proxyed,error = %v", err)
		return
	}

	log.StartLogger.Tracef("after initializeUpstreamConnectionPool")
	s.timeout = parseProxyTimeout(route, headers)
	s.retryState = newRetryState(route.RouteRule().Policy().RetryPolicy(), headers, s.cluster)

	//Build Request
	s.upstreamRequest = &upstreamRequest{
		downStream: s,
		proxy:      s.proxy,
		connPool:   pool,
	}

	//Call upstream's append header method to build upstream's request
	s.upstreamRequest.appendHeaders(headers, endStream)

	if endStream {
		s.onUpstreamRequestSent()
	}
}

func (s *downStream) OnReceiveData(data types.IoBuffer, endStream bool) {
	s.requestInfo.SetBytesReceived(s.requestInfo.BytesReceived() + uint64(data.Len()))
	s.downstreamRecvDone = endStream

	s.doReceiveData(nil, data, endStream)
}

func (s *downStream) doReceiveData(filter *activeStreamReceiverFilter, data types.IoBuffer, endStream bool) {
	log.StartLogger.Tracef("active stream do decode data")
	// if active stream finished the lifecycle, just ignore further data
	if s.streamRoundTripDone {
		return
	}

	if s.runReceiveDataFilters(filter, data, endStream) {
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
		s.upstreamRequest.appendData(copied, endStream)
	} else {
		s.upstreamRequest.appendData(data, endStream)
	}
}

func (s *downStream) OnReceiveTrailers(trailers map[string]string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	// exit on downstream reset
	if s.element == nil {
		return
	}

	s.downstreamRecvDone = true

	s.doReceiveTrailers(nil, trailers)
}

func (s *downStream) OnDecodeError(err error, headers map[string]string) {
	// todo: enrich headers' information to do some hijack
	// Check headers' info to do hijack
	switch err.Error() {
	case types.CodecException:
		s.sendHijackReply(types.CodecExceptionCode, headers)
	case types.DeserializeException:
		s.sendHijackReply(types.DeserialExceptionCode, headers)
	default:
		s.sendHijackReply(types.UnknownCode, headers)
	}

	s.OnResetStream(types.StreamLocalReset)
}

func (s *downStream) doReceiveTrailers(filter *activeStreamReceiverFilter, trailers map[string]string) {
	// if active stream finished the lifecycle, just ignore further data
	if s.streamRoundTripDone {
		return
	}

	if s.runReceiveTrailersFilters(filter, trailers) {
		return
	}

	s.downstreamReqTrailers = trailers
	s.onUpstreamRequestSent()
	s.upstreamRequest.appendTrailers(trailers)
}

func (s *downStream) onUpstreamRequestSent() {
	s.upstreamRequestSent = true
	s.requestInfo.SetRequestReceivedDuration(time.Now())

	if s.upstreamRequest != nil {
		// setup per req timeout timer
		s.setupPerReqTimeout()

		// setup global timeout timer
		if s.timeout.GlobalTimeout > 0 {
			if s.responseTimer != nil {
				s.responseTimer.stop()
			}

			s.responseTimer = newTimer(s.onResponseTimeout, s.timeout.GlobalTimeout)
			s.responseTimer.start()
		}
	}
}

// Note: global-timer MUST be stopped before active stream got recycled, otherwise resetting stream's properties will cause panic here
func (s *downStream) onResponseTimeout() {
	s.responseTimer = nil
	s.cluster.Stats().UpstreamRequestTimeout.Inc(1)

	if s.upstreamRequest != nil {
		if s.upstreamRequest.host != nil {
			s.upstreamRequest.host.HostStats().UpstreamRequestTimeout.Inc(1)
		}

		s.upstreamRequest.resetStream()
	}

	s.onUpstreamReset(UpstreamGlobalTimeout, types.StreamLocalReset)
}

func (s *downStream) setupPerReqTimeout() {
	timeout := s.timeout

	if timeout.TryTimeout > 0 {
		if s.perRetryTimer != nil {
			s.perRetryTimer.stop()
		}

		s.perRetryTimer = newTimer(s.onPerReqTimeout, timeout.TryTimeout*time.Second)
		s.perRetryTimer.start()
	}
}

// Note: per-try-timer MUST be stopped before active stream got recycled, otherwise resetting stream's properties will cause panic here
func (s *downStream) onPerReqTimeout() {
	if !s.downstreamResponseStarted {
		// handle timeout on response not

		s.perRetryTimer = nil
		s.cluster.Stats().UpstreamRequestTimeout.Inc(1)

		if s.upstreamRequest.host != nil {
			s.upstreamRequest.host.HostStats().UpstreamRequestTimeout.Inc(1)
		}

		s.upstreamRequest.resetStream()
		s.requestInfo.SetResponseFlag(types.UpstreamRequestTimeout)
		s.onUpstreamReset(UpstreamPerTryTimeout, types.StreamLocalReset)
	} else {
		log.DefaultLogger.Debugf("Skip request timeout on getting upstream response")
	}
}

func (s *downStream) initializeUpstreamConnectionPool(clusterName string, lbCtx types.LoadBalancerContext) (error, types.ConnectionPool) {
	clusterSnapshot := s.proxy.clusterManager.Get(nil, clusterName)

	if reflect.ValueOf(clusterSnapshot).IsNil() {
		// no available cluster
		log.DefaultLogger.Errorf("cluster snapshot is nil, cluster name is: %s", clusterName)
		s.requestInfo.SetResponseFlag(types.NoRouteFound)
		s.sendHijackReply(types.RouterUnavailableCode, s.downstreamReqHeaders)

		return errors.New(fmt.Sprintf("unkown cluster %s", clusterName)), nil
	}

	s.cluster = clusterSnapshot.ClusterInfo()
	var connPool types.ConnectionPool

	// todo: refactor
	switch types.Protocol(s.proxy.config.UpstreamProtocol) {
	case protocol.SofaRpc:
		connPool = s.proxy.clusterManager.SofaRpcConnPoolForCluster(clusterName, lbCtx)
	case protocol.Http2:
		connPool = s.proxy.clusterManager.HttpConnPoolForCluster(clusterName, protocol.Http2, lbCtx)
	case protocol.Http1:
		connPool = s.proxy.clusterManager.HttpConnPoolForCluster(clusterName, protocol.Http1, lbCtx)
	case protocol.Xprotocol:
		connPool = s.proxy.clusterManager.XprotocolConnPoolForCluster(clusterName, protocol.Xprotocol, nil)
	default:
		connPool = s.proxy.clusterManager.HttpConnPoolForCluster(clusterName, protocol.Http2, lbCtx)
	}

	if connPool == nil {
		s.requestInfo.SetResponseFlag(types.NoHealthyUpstream)
		s.sendHijackReply(types.NoHealthUpstreamCode, s.downstreamReqHeaders)

		return errors.New(fmt.Sprintf("no healthy upstream in cluster %s", clusterName)), nil
	}

	// TODO: update upstream stats

	return nil, connPool
}

// ~~~ active stream sender wrapper

func (s *downStream) appendHeaders(headers map[string]string, endStream bool) {
	s.streamRoundTripDone = endStream
	s.doAppendHeaders(nil, headers, endStream)
}

func (s *downStream) doAppendHeaders(filter *activeStreamSenderFilter, headers interface{}, endStream bool) {
	if s.runAppendHeaderFilters(filter, headers, endStream) {
		return
	}

	//Currently, just log the error
	if err := s.responseSender.AppendHeaders(headers, endStream); err != nil {
		s.logger.Errorf("[downstream] decode headers error, %s", err)
	}

	if endStream {
		s.endStream()
	}
}

func (s *downStream) appendData(data types.IoBuffer, endStream bool) {
	s.streamRoundTripDone = endStream

	s.doAppendData(nil, data, endStream)
}

func (s *downStream) doAppendData(filter *activeStreamSenderFilter, data types.IoBuffer, endStream bool) {
	if s.runAppendDataFilters(filter, data, endStream) {
		return
	}

	s.responseSender.AppendData(data, endStream)

	s.requestInfo.SetBytesSent(s.requestInfo.BytesSent() + uint64(data.Len()))

	if endStream {
		s.endStream()
	}
}

func (s *downStream) appendTrailers(trailers map[string]string) {
	s.streamRoundTripDone = true

	s.doAppendTrailers(nil, trailers)
}

func (s *downStream) doAppendTrailers(filter *activeStreamSenderFilter, trailers map[string]string) {
	if s.runAppendTrailersFilters(filter, trailers) {
		return
	}

	s.responseSender.AppendTrailers(trailers)
	s.endStream()
}

// ~~~ upstream event handler
func (s *downStream) onUpstreamReset(urtype UpstreamResetType, reason types.StreamResetReason) {
	// todo: update stats
	log.StartLogger.Tracef("on upstream reset invoked")

	// see if we need a retry
	if urtype != UpstreamGlobalTimeout &&
		s.downstreamResponseStarted && s.retryState != nil {
		retryCheck := s.retryState.retry(nil, reason, s.doRetry)

		if retryCheck == types.ShouldRetry && s.setupRetry(true) {
			// setup retry timer and return
			return
		} else if retryCheck == types.RetryOverflow {
			s.requestInfo.SetResponseFlag(types.UpstreamOverflow)
		}
	}

	// clean up all timers
	s.cleanUp()

	if reason == types.StreamOverflow || reason == types.StreamConnectionFailed ||
		reason == types.StreamRemoteReset {
		log.StartLogger.Tracef("on upstream reset reason %v", reason)
		s.upstreamRequest.connPool.Close()
		s.proxy.readCallbacks.Connection().RawConn().Close()
		s.resetStream()
		return
	}

	// If we have not yet sent anything downstream, send a response with an appropriate status code.
	// Otherwise just reset the ongoing response.
	if s.downstreamResponseStarted {
		s.resetStream()
	} else {
		// send err response if response not started
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

func (s *downStream) onUpstreamHeaders(headers map[string]string, endStream bool) {
	s.downstreamRespHeaders = headers

	// check retry
	if s.retryState != nil {
		retryCheck := s.retryState.retry(headers, "", s.doRetry)

		if retryCheck == types.ShouldRetry && s.setupRetry(endStream) {
			return
		} else if retryCheck == types.RetryOverflow {
			s.requestInfo.SetResponseFlag(types.UpstreamOverflow)
		}

		s.retryState.reset()
	}

	s.requestInfo.SetResponseReceivedDuration(time.Now())

	s.downstreamResponseStarted = true

	if endStream {
		s.onUpstreamResponseRecvFinished()
	}

	// todo: insert proxy headers
	s.appendHeaders(headers, endStream)
}

func (s *downStream) onUpstreamData(data types.IoBuffer, endStream bool) {
	if endStream {
		s.onUpstreamResponseRecvFinished()
	}

	s.appendData(data, endStream)
}

func (s *downStream) onUpstreamTrailers(trailers map[string]string) {
	s.onUpstreamResponseRecvFinished()

	s.appendTrailers(trailers)
}

func (s *downStream) onUpstreamResponseRecvFinished() {
	if !s.upstreamRequestSent {
		s.upstreamRequest.resetStream()
	}

	// todo: stats
	// todo: logs

	s.cleanUp()
}

func (s *downStream) setupRetry(endStream bool) bool {
	if !s.upstreamRequestSent {
		return false
	}

	if !endStream {
		s.upstreamRequest.resetStream()
	}

	s.upstreamRequest.requestSender = nil

	// reset per req timer
	if s.perRetryTimer != nil {
		s.perRetryTimer.stop()
		s.perRetryTimer = nil
	}

	return true
}

// Note: retry-timer MUST be stopped before active stream got recycled, otherwise resetting stream's properties will cause panic here
func (s *downStream) doRetry() {
	err, pool := s.initializeUpstreamConnectionPool(s.cluster.Name(), nil)

	if err != nil {
		s.sendHijackReply(types.NoHealthUpstreamCode, s.downstreamReqHeaders)
		s.cleanUp()
		return
	}

	s.upstreamRequest = &upstreamRequest{
		downStream: s,
		proxy:      s.proxy,
		connPool:   pool,
	}

	s.upstreamRequest.appendHeaders(s.downstreamReqHeaders,
		s.downstreamReqDataBuf != nil && s.downstreamReqTrailers != nil)

	if s.upstreamRequest != nil {
		if s.downstreamReqDataBuf != nil {
			// make a data copy to retry
			copied := s.downstreamReqDataBuf.Clone()
			s.upstreamRequest.appendData(copied, s.downstreamReqTrailers == nil)
		}

		if s.downstreamReqTrailers != nil {
			s.upstreamRequest.appendTrailers(s.downstreamReqTrailers)
		}

		// setup per try timeout timer
		s.setupPerReqTimeout()
	}
}

func (s *downStream) onUpstreamAboveWriteBufferHighWatermark() {
	s.responseSender.GetStream().ReadDisable(true)
}

func (s *downStream) onUpstreamBelowWriteBufferHighWatermark() {
	s.responseSender.GetStream().ReadDisable(false)
}

// Downstream got reset in proxy context on scenario below:
// 1. downstream filter reset downstream
// 2. corresponding upstream got reset
func (s *downStream) resetStream() {
	s.endStream()
}

func (s *downStream) sendHijackReply(code int, headers map[string]string) {
	if headers == nil {
		headers = make(map[string]string, 5)
	}

	headers[types.HeaderStatus] = strconv.Itoa(code)
	s.appendHeaders(headers, true)
}

func (s *downStream) cleanUp() {
	// reset upstream request
	// if a downstream filter ends downstream before send to upstream, upstreamRequest will be nil
	if s.upstreamRequest != nil {
		s.upstreamRequest.requestSender = nil
	}

	// reset retry state
	// if  a downstream filter ends downstream before send to upstream, retryState will be nil
	if s.retryState != nil {
		s.retryState.reset()
	}

	// reset pertry timer
	if s.perRetryTimer != nil {
		s.perRetryTimer.stop()
		s.perRetryTimer = nil
	}

	// reset response timer
	if s.responseTimer != nil {
		s.responseTimer.stop()
		s.responseTimer = nil
	}
}

func (s *downStream) setBufferLimit(bufferLimit uint32) {
	s.bufferLimit = bufferLimit

	// todo
}

func (s *downStream) AddStreamReceiverFilter(filter types.StreamReceiverFilter) {
	sf := newActiveStreamReceiverFilter(len(s.receiverFilters), s, filter)
	s.receiverFilters = append(s.receiverFilters, sf)
}

func (s *downStream) AddStreamSenderFilter(filter types.StreamSenderFilter) {
	sf := newActiveStreamSenderFilter(len(s.senderFilters), s, filter)
	s.senderFilters = append(s.senderFilters, sf)
}

func (s *downStream) reset() {
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
	s.responseSender = nil
	s.upstreamRequest.downStream = nil
	s.upstreamRequest.proxy = nil
	s.upstreamRequest.upstreamRespHeaders = nil
	s.upstreamRequest = nil
	s.perRetryTimer = nil
	s.responseTimer = nil
	s.downstreamRespHeaders = nil
	s.downstreamReqDataBuf = nil
	s.downstreamReqTrailers = nil
	s.downstreamRespHeaders = nil
	s.downstreamRespDataBuf = nil
	s.downstreamRespTrailers = nil
	s.downstreamResponseStarted = false
	s.upstreamRequestSent = false
	s.downstreamRecvDone = false
	s.streamRoundTripDone = false
	s.senderFiltersStreaming = false
	s.receiverFiltersStreaming = false
	s.filterStage = 0
	s.senderFilters = s.senderFilters[:0]
	s.receiverFilters = s.receiverFilters[:0]
}

// types.LoadBalancerContext
// no use currently
func (s *downStream) ComputeHashKey() types.HashedValue {
	return [16]byte{}
}

func (s *downStream) MetadataMatchCriteria() types.MetadataMatchCriteria {
	if nil != s.requestInfo.RouteEntry() {
		return s.requestInfo.RouteEntry().MetadataMatchCriteria()
	} else {
		return nil
	}
}

func (s *downStream) DownstreamConnection() net.Conn {
	return s.proxy.readCallbacks.Connection().RawConn()
}

func (s *downStream) DownstreamHeaders() map[string]string {
	return s.downstreamReqHeaders
}
