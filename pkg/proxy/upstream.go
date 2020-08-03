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
	"context"
	"sync/atomic"
	"time"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
)

// types.StreamEventListener
// types.StreamReceiveListener
// types.PoolEventListener
type upstreamRequest struct {
	proxy         *proxy
	downStream    *downStream
	protocol      types.ProtocolName
	host          types.Host
	requestSender types.StreamSender
	connPool      types.ConnectionPool

	// ~~~ upstream response buf
	upstreamRespHeaders types.HeaderMap

	//~~~ state
	sendComplete bool
	dataSent     bool
	trailerSent  bool
	setupRetry   bool

	// time at send upstream request
	startTime time.Time

	// list element
	element *list.Element
}

// reset upstream request in proxy context
// 1. downstream cleanup
// 2. on upstream global timeout
// 3. on upstream per req timeout
// 4. on upstream response receive error
// 5. before a retry
func (r *upstreamRequest) resetStream() {
	if r.requestSender != nil {
		r.requestSender.GetStream().RemoveEventListener(r)
		r.requestSender.GetStream().ResetStream(types.StreamLocalReset)
	}
}

// types.StreamEventListener
// Called by stream layer normally
func (r *upstreamRequest) OnResetStream(reason types.StreamResetReason) {
	if r.setupRetry {
		return
	}
	// todo: check if we get a reset on encode request headers. e.g. send failed
	if !atomic.CompareAndSwapUint32(&r.downStream.upstreamReset, 0, 1) {
		return
	}

	r.downStream.resetReason = reason
	r.downStream.sendNotify()
}

func (r *upstreamRequest) OnDestroyStream() {}

func (r *upstreamRequest) endStream() {
	upstreamResponseDurationNs := time.Now().Sub(r.startTime).Nanoseconds()
	r.host.HostStats().UpstreamRequestDuration.Update(upstreamResponseDurationNs)
	r.host.HostStats().UpstreamRequestDurationTotal.Inc(upstreamResponseDurationNs)
	r.host.ClusterInfo().Stats().UpstreamRequestDuration.Update(upstreamResponseDurationNs)
	r.host.ClusterInfo().Stats().UpstreamRequestDurationTotal.Inc(upstreamResponseDurationNs)

	// todo: record upstream process time in request info
}

// types.StreamReceiveListener
// Method to decode upstream's response message
func (r *upstreamRequest) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	if r.downStream.processDone() || r.setupRetry {
		return
	}

	r.endStream()

	if code, err := protocol.MappingHeaderStatusCode(r.downStream.context, r.protocol, headers); err == nil {
		r.downStream.requestInfo.SetResponseCode(code)
	}

	r.downStream.requestInfo.SetResponseReceivedDuration(time.Now())
	r.downStream.downstreamRespHeaders = headers
	r.downStream.downstreamRespDataBuf = data
	r.downStream.downstreamRespTrailers = trailers

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(r.downStream.context, "[proxy] [upstream] OnReceive headers: %+v, data: %+v, trailers: %+v", headers, data, trailers)
	}

	r.downStream.sendNotify()
}

func (r *upstreamRequest) receiveHeaders(endStream bool) {
	if r.downStream.processDone() || r.setupRetry {
		return
	}

	r.downStream.onUpstreamHeaders(endStream)
}

func (r *upstreamRequest) receiveData(endStream bool) {
	if r.downStream.processDone() || r.setupRetry {
		return
	}

	r.downStream.onUpstreamData(endStream)
}

func (r *upstreamRequest) receiveTrailers() {
	if r.downStream.processDone() || r.setupRetry {
		return
	}

	r.downStream.onUpstreamTrailers()

}

func (r *upstreamRequest) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
	log.Proxy.Errorf(r.downStream.context, "[proxy] [upstream] OnDecodeError error: %+v", err)

	r.OnResetStream(types.StreamLocalReset)
}

// ~~~ send request wrapper
func (r *upstreamRequest) appendHeaders(endStream bool) {
	if r.downStream.processDone() {
		return
	}
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(r.downStream.context, "[proxy] [upstream] append headers: %+v", r.downStream.downstreamReqHeaders)
	}
	r.sendComplete = endStream

	if r.downStream.oneway {
		r.connPool.NewStream(r.downStream.context, nil, r)
	} else {
		r.connPool.NewStream(r.downStream.context, r, r)
	}
}

func (r *upstreamRequest) convertHeader(headers types.HeaderMap) types.HeaderMap {
	if r.downStream.noConvert {
		return headers
	}

	dp, up := r.downStream.convertProtocol()

	// need protocol convert
	if dp != up {
		if convHeader, err := protocol.ConvertHeader(r.downStream.context, dp, up, headers); err == nil {
			return convHeader
		} else {
			log.Proxy.Warnf(r.downStream.context, "[proxy] [upstream] convert header from %s to %s failed, %s", dp, up, err.Error())
		}
	}
	return headers
}

func (r *upstreamRequest) appendData(endStream bool) {
	if r.downStream.processDone() {
		return
	}
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(r.downStream.context, "[proxy] [upstream] append data:% +v", r.downStream.downstreamReqDataBuf)
	}

	data := r.downStream.downstreamReqDataBuf
	r.sendComplete = endStream
	r.dataSent = true
	r.requestSender.AppendData(r.downStream.context, r.convertData(data), endStream)
}

func (r *upstreamRequest) convertData(data types.IoBuffer) types.IoBuffer {
	if r.downStream.noConvert {
		return data
	}

	dp, up := r.downStream.convertProtocol()

	// need protocol convert
	if dp != up {
		if convData, err := protocol.ConvertData(r.downStream.context, dp, up, data); err == nil {
			return convData
		} else {
			log.Proxy.Warnf(r.downStream.context, "[proxy] [upstream] convert data from %s to %s failed, %s", dp, up, err.Error())
		}
	}
	return data
}

func (r *upstreamRequest) appendTrailers() {
	if r.downStream.processDone() {
		return
	}
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(r.downStream.context, "[proxy] [upstream] append trailers:%+v", r.downStream.downstreamReqTrailers)
	}
	trailers := r.downStream.downstreamReqTrailers
	r.sendComplete = true
	r.trailerSent = true
	r.requestSender.AppendTrailers(r.downStream.context, trailers)
}

func (r *upstreamRequest) convertTrailer(trailers types.HeaderMap) types.HeaderMap {
	if r.downStream.noConvert {
		return trailers
	}

	dp, up := r.downStream.convertProtocol()

	// need protocol convert
	if dp != up {
		if convTrailer, err := protocol.ConvertTrailer(r.downStream.context, dp, up, trailers); err == nil {
			return convTrailer
		} else {
			log.Proxy.Warnf(r.downStream.context, "[proxy] [upstream] convert header from %s to %s failed, %s", dp, up, err.Error())
		}
	}
	return trailers
}

// types.PoolEventListener
func (r *upstreamRequest) OnFailure(reason types.PoolFailureReason, host types.Host) {
	var resetReason types.StreamResetReason

	log.Proxy.Errorf(r.downStream.context, "[proxy] [upstream] OnFailure host:%s, reason:%v", host.AddressString(), reason)

	switch reason {
	case types.Overflow:
		resetReason = types.StreamOverflow
	case types.ConnectionFailure:
		resetReason = types.StreamConnectionFailed
	}

	r.host = host
	r.OnResetStream(resetReason)
}

func (r *upstreamRequest) OnReady(sender types.StreamSender, host types.Host) {
	// debug message for upstream
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(r.downStream.context, "[proxy] [upstream] connPool ready, proxyId = %v, host = %s", r.downStream.ID, host.AddressString())
	}

	r.requestSender = sender
	r.host = host
	r.requestSender.GetStream().AddEventListener(r)
	// start a upstream send
	r.startTime = time.Now()

	r.downStream.requestInfo.OnUpstreamHostSelected(host)
	r.downStream.requestInfo.SetUpstreamLocalAddress(host.AddressString())

	if trace.IsEnabled() {
		span := trace.SpanFromContext(r.downStream.context)
		if span != nil {
			span.InjectContext(r.downStream.downstreamReqHeaders, r.downStream.requestInfo)
		}
	}

	endStream := r.sendComplete && !r.dataSent && !r.trailerSent
	r.requestSender.AppendHeaders(r.downStream.context, r.convertHeader(r.downStream.downstreamReqHeaders), endStream)

	// todo: check if we get a reset on send headers
}
