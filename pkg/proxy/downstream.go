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
)

// types.StreamCallbacks
// types.StreamDecoder
type activeStream struct {
	proxy   *proxy
	cluster types.ClusterInfo
	element *list.Element

	// ~~~ control args
	timeout    *ProxyTimeout
	retryState *retrystate

	requestInfo     types.RequestInfo
	responseEncoder types.StreamEncoder
	upstreamRequest *upstreamRequest

	// ~~~ downstream request buf
	downstreamHeaders  map[string]string
	downstreamDataBuf  types.IoBuffer
	downstreamTrailers map[string]string

	// ~~~ state
	// starts to send back downstream response, set on upstream response detected
	downstreamResponseStarted bool
	// upstream req sent
	upstreamRequestSent bool
	// downstream request received done
	downstreamRecvDone bool
	// 1. upstream response received done 2. done by a hijack response
	localProcessDone bool
}

// types.StreamCallbacks
func (s *activeStream) OnResetStream(reason types.StreamResetReason) {
	// todo: logging
	// todo: stats

	s.proxy.deleteActiveStream(s)
}

func (s *activeStream) OnAboveWriteBufferHighWatermark() {}

func (s *activeStream) OnBelowWriteBufferLowWatermark() {}

// case 1: stream's lifecycle ends normally
// case 2: proxy ends stream in lifecycle
func (s *activeStream) endStream() {
	s.clean()

	if s.responseEncoder != nil {
		if !s.downstreamRecvDone || !s.localProcessDone {
			// if downstream req received not done, or local proxy process not done by handle upstream response,
			// just mark it as done and reset stream as a failed case
			s.localProcessDone = true
			s.responseEncoder.GetStream().RemoveCallbacks(s)
			s.responseEncoder.GetStream().ResetStream(types.StreamLocalReset)
		}
	}

	s.proxy.deleteActiveStream(s)

	// note: if proxy logic resets the stream, there maybe some underlying data in the conn.
	// we ignore this for now, fix as a todo
}

// types.StreamDecoder
func (s *activeStream) OnDecodeHeaders(headers map[string]string, endStream bool) {
	s.downstreamRecvDone = endStream
	s.downstreamHeaders = headers

	//do some route by service name
	route := s.proxy.routerConfig.Route(headers)

	if route == nil || route.RouteRule() == nil {
		// no route
		s.requestInfo.SetResponseFlag(types.NoRouteFound)
		s.sendHijackReply(404, nil)

		return
	}

	err, pool := s.initializeUpstreamConnectionPool(route.RouteRule().ClusterName())

	if err != nil {
		// no available host
		s.requestInfo.SetResponseFlag(types.NoHealthyUpstream)
		s.sendHijackReply(500, nil)

		return
	}

	s.timeout = parseProxyTimeout(route, headers)
	s.retryState = newRetryState(route.RouteRule().Policy().RetryPolicy(), headers, s.cluster)

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

	return
}

func (s *activeStream) OnDecodeData(data types.IoBuffer, endStream bool) {
	s.downstreamRecvDone = endStream

	// if active stream finished the lifecycle, just ignore further data
	if s.localProcessDone {
		return
	}

	shouldBufData := false
	if s.retryState != nil && s.retryState.retryOn {
		shouldBufData = true

		// todo: set a buf limit
	}

	if shouldBufData {
		buf := buffer.NewIoBuffer(data.Len())
		// todo
		buf.ReadFrom(data)

		if s.downstreamDataBuf == nil {
			s.downstreamDataBuf = buffer.NewIoBuffer(data.Len())
		}

		s.downstreamDataBuf.Append(buf.Bytes())
		s.upstreamRequest.encodeData(s.downstreamDataBuf, endStream)
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

	s.downstreamTrailers = trailers
	s.upstreamRequest.encodeTrailers(trailers)
	s.onUpstreamRequestSent()
}

func (s *activeStream) OnDecodeComplete(buf types.IoBuffer) {}

func (s *activeStream) onUpstreamRequestSent() {
	s.upstreamRequestSent = true

	if s.upstreamRequest != nil {
		// setup per try timeout timer
		s.upstreamRequest.setupPerTryTimeout()

		// setup global timeout timer
		if s.timeout.GlobalTimeout > 0 {
			go func() {
				// todo: support cancel
				select {
				case <-time.After(s.timeout.GlobalTimeout):
					s.onResponseTimeout()
				}
			}()
		}
	}
}

func (s *activeStream) onResponseTimeout() {
	if s.upstreamRequest != nil {
		s.upstreamRequest.resetStream()
	}

	s.onUpstreamReset(UpstreamGlobalTimeout, types.StreamLocalReset)
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
	s.cluster = clusterInfo
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

// ~~~ active stream encoder wrapper

func (s *activeStream) encodeHeaders(headers map[string]string, endStream bool) {
	s.localProcessDone = endStream
	s.responseEncoder.EncodeHeaders(headers, endStream)

	if endStream {
		s.endStream()
	}
}

func (s *activeStream) encodeData(data types.IoBuffer, endStream bool) {
	s.localProcessDone = endStream
	s.responseEncoder.EncodeData(data, endStream)

	if endStream {
		s.endStream()
	}
}

func (s *activeStream) encodeTrailers(trailers map[string]string) {
	s.localProcessDone = true
	s.responseEncoder.EncodeTrailers(trailers)
	s.endStream()
}

// ~~~ upstream event handler

func (s *activeStream) onUpstreamHeaders(headers map[string]string, endStream bool) {
	// check retry
	if s.retryState != nil {
		shouldRetry := s.retryState.shouldRetry(headers, "")
		if shouldRetry && s.setupRetry(endStream) {
			s.retryState.scheduleRetry(s.doRetry)
			return
		}

		s.retryState = nil
	}

	s.downstreamResponseStarted = true

	if endStream {
		s.onUpstreamResponseComplete()
	}

	// todo: insert proxy headers

	s.encodeHeaders(headers, endStream)
}

func (s *activeStream) onUpstreamData(data types.IoBuffer, endStream bool) {
	if endStream {
		s.onUpstreamResponseComplete()
	}

	s.encodeData(data, endStream)
}

func (s *activeStream) onUpstreamTrailers(trailers map[string]string) {
	s.onUpstreamResponseComplete()

	s.encodeTrailers(trailers)
}

func (s *activeStream) onUpstreamResponseComplete() {
	if !s.upstreamRequestSent {
		s.upstreamRequest.resetStream()
	}

	// todo: stats
	// todo: logs

	s.upstreamRequest.clean()
}

func (s *activeStream) onUpstreamReset(urtype UpstreamResetType, reason types.StreamResetReason) {
	// todo: update stats

	// see if we need a retry
	if urtype != UpstreamGlobalTimeout &&
		s.downstreamResponseStarted && s.retryState != nil {
		if s.retryState.shouldRetry(nil, reason) && s.setupRetry(true) {
			s.retryState.scheduleRetry(s.doRetry)
			return
		}
	}

	// If we have not yet sent anything downstream, send a response with an appropriate status code.
	// Otherwise just reset the ongoing response.
	if s.downstreamResponseStarted {
		s.upstreamRequest.clean()
		s.resetStream()
	} else {
		s.upstreamRequest.clean()

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

	if err != nil || pool != nil {
		// no available host
		s.requestInfo.SetResponseFlag(types.NoHealthyUpstream)
		s.sendHijackReply(500, nil)

		return
	}

	s.upstreamRequest = &upstreamRequest{
		activeStream: s,
		proxy:        s.proxy,
		connPool:     pool,
		requestInfo:  network.NewRequestInfo(),
	}

	s.upstreamRequest.connPool.NewStream(0, s.upstreamRequest, s.upstreamRequest)

	if s.upstreamRequest != nil {
		if s.downstreamDataBuf != nil {
			buf := buffer.NewIoBuffer(s.downstreamDataBuf.Len())
			buf.ReadFrom(s.downstreamDataBuf)
			s.upstreamRequest.requestEncoder.EncodeData(buf, s.downstreamTrailers == nil)
		}

		if s.downstreamTrailers != nil {
			s.upstreamRequest.requestEncoder.EncodeTrailers(s.downstreamTrailers)
		}
	}

	// setup per try timeout timer
	s.upstreamRequest.setupPerTryTimeout()
}

func (s *activeStream) resetStream() {
	s.endStream()
}

func (s *activeStream) sendHijackReply(code int, headers map[string]string) {
	if headers == nil {
		headers = make(map[string]string)
	}

	// todo
	headers["Status"] = strconv.Itoa(code)
	s.encodeHeaders(headers, true)
}

func (s *activeStream) clean() {
	// todo: clean as timer
}
