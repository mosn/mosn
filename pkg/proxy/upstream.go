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
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
	"sync/atomic"
)

// types.StreamEventListener
// types.StreamReceiveListener
// types.PoolEventListener
type upstreamRequest struct {
	proxy         *proxy
	downStream    *downStream
	protocol      types.Protocol
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
	/*
		workerPool.Offer(&event{
			id:  r.downStream.ID,
			dir: upstream,
			evt: reset,
			handle: func() {
				r.ResetStream(reason)
			},
		}, false)
	*/

	/*
		r.requestSender = nil
	*/

	if !r.setupRetry {
		// todo: check if we get a reset on encode request headers. e.g. send failed
		if !atomic.CompareAndSwapUint32(&r.downStream.upstreamReset, 0, 1) {
			return
		}

		r.downStream.resetReason = reason
		//r.downStream.onUpstreamReset(reason)
		r.downStream.sendNotify()
	}
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

func (r *upstreamRequest) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	if r.downStream.processDone() {
		return
	}

	r.endStream()

	if code, err := protocol.MappingHeaderStatusCode(r.protocol, headers); err == nil {
		r.downStream.requestInfo.SetResponseCode(code)
	}

	r.downStream.requestInfo.SetResponseReceivedDuration(time.Now())
	r.downStream.downstreamRespHeaders = headers

	if data != nil {
		r.downStream.downstreamRespDataBuf = data.Clone()
		data.Drain(data.Len())
	}

	r.downStream.downstreamRespTrailers = trailers

	r.downStream.logger.Debugf("upstreamRequest OnReceive %+v", headers)

	r.downStream.sendNotify()
}

// types.StreamReceiveListener
// Method to decode upstream's response message
func (r *upstreamRequest) OnReceiveHeaders(ctx context.Context, headers types.HeaderMap, endStream bool) {
	// we convert a status to http standard code, and log it
	if code, err := protocol.MappingHeaderStatusCode(r.protocol, headers); err == nil {
		r.downStream.requestInfo.SetResponseCode(code)
	}

	r.downStream.requestInfo.SetResponseReceivedDuration(time.Now())

	if endStream {
		r.endStream()
	}

	workerPool.Offer(&event{
		id:  r.downStream.ID,
		dir: upstream,
		evt: recvHeader,
		handle: func() {
			r.ReceiveHeaders(headers, endStream)
		},
	}, true)
}

func (r *upstreamRequest) ReceiveHeaders(headers types.HeaderMap, endStream bool) {
	if r.downStream.processDone() {
		return
	}

	r.upstreamRespHeaders = headers
	r.downStream.onUpstreamHeaders(headers, endStream)
}

func (r *upstreamRequest) OnReceiveData(context context.Context, data types.IoBuffer, endStream bool) {
	r.downStream.downstreamRespDataBuf = data.Clone()
	data.Drain(data.Len())

	if endStream {
		r.endStream()
	}

	workerPool.Offer(&event{
		id:  r.downStream.ID,
		dir: upstream,
		evt: recvData,
		handle: func() {
			r.ReceiveData(r.downStream.downstreamRespDataBuf, endStream)
		},
	}, true)
}

func (r *upstreamRequest) ReceiveData(data types.IoBuffer, endStream bool) {
	if r.downStream.processDone() {
		return
	}

	if !r.setupRetry {
		r.downStream.onUpstreamData(data, endStream)
	}
}

func (r *upstreamRequest) OnReceiveTrailers(context context.Context, trailers types.HeaderMap) {
	r.endStream()

	workerPool.Offer(&event{
		id:  r.downStream.ID,
		dir: upstream,
		evt: recvTrailer,
		handle: func() {
			r.ReceiveTrailers(trailers)
		},
	}, true)
}

func (r *upstreamRequest) ReceiveTrailers(trailers types.HeaderMap) {
	if r.downStream.processDone() {
		return
	}

	if !r.setupRetry {
		r.downStream.onUpstreamTrailers(trailers)
	}
}

func (r *upstreamRequest) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
	r.OnResetStream(types.StreamLocalReset)
}

// ~~~ send request wrapper
func (r *upstreamRequest) appendHeaders(headers types.HeaderMap, endStream bool) {
	if r.downStream.processDone() {
		return
	}
	log.StartLogger.Tracef("upstream request encode headers")
	r.sendComplete = endStream

	log.StartLogger.Tracef("upstream request before conn pool new stream")
	r.connPool.NewStream(r.downStream.context, r, r)
}

func (r *upstreamRequest) convertHeader(headers types.HeaderMap) types.HeaderMap {
	dp, up := r.downStream.convertProtocol()

	// need protocol convert
	if dp != up {
		if convHeader, err := protocol.ConvertHeader(r.downStream.context, dp, up, headers); err == nil {
			return convHeader
		} else {
			r.downStream.logger.Errorf("convert header from %s to %s failed, %s", dp, up, err.Error())
		}
	}
	return headers
}

func (r *upstreamRequest) appendData(data types.IoBuffer, endStream bool) {
	if r.downStream.processDone() {
		return
	}
	log.DefaultLogger.Debugf("upstream request encode data")
	r.sendComplete = endStream
	r.dataSent = true
	r.requestSender.AppendData(r.downStream.context, r.convertData(data), endStream)
}

func (r *upstreamRequest) convertData(data types.IoBuffer) types.IoBuffer {
	dp, up := r.downStream.convertProtocol()

	// need protocol convert
	if dp != up {
		if convData, err := protocol.ConvertData(r.downStream.context, dp, up, data); err == nil {
			return convData
		} else {
			r.downStream.logger.Errorf("convert data from %s to %s failed, %s", dp, up, err.Error())
		}
	}
	return data
}

func (r *upstreamRequest) appendTrailers(trailers types.HeaderMap) {
	if r.downStream.processDone() {
		return
	}
	log.DefaultLogger.Debugf("upstream request encode trailers")
	r.sendComplete = true
	r.trailerSent = true
	r.requestSender.AppendTrailers(r.downStream.context, trailers)
}

func (r *upstreamRequest) convertTrailer(trailers types.HeaderMap) types.HeaderMap {
	dp, up := r.downStream.convertProtocol()

	// need protocol convert
	if dp != up {
		if convTrailer, err := protocol.ConvertTrailer(r.downStream.context, dp, up, trailers); err == nil {
			return convTrailer
		} else {
			r.downStream.logger.Errorf("convert header from %s to %s failed, %s", dp, up, err.Error())
		}
	}
	return trailers
}

// types.PoolEventListener
func (r *upstreamRequest) OnFailure(reason types.PoolFailureReason, host types.Host) {
	var resetReason types.StreamResetReason

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
	r.requestSender = sender
	r.host = host
	r.requestSender.GetStream().AddEventListener(r)
	// start a upstream send
	r.startTime = time.Now()

	endStream := r.sendComplete && !r.dataSent && !r.trailerSent
	r.requestSender.AppendHeaders(r.downStream.context, r.convertHeader(r.downStream.downstreamReqHeaders), endStream)

	r.downStream.requestInfo.OnUpstreamHostSelected(host)
	r.downStream.requestInfo.SetUpstreamLocalAddress(host.Address())

	// debug message for upstream
	r.downStream.logger.Debugf("client conn id %d, proxy id %d, upstream id %d", r.proxy.readCallbacks.Connection().ID(), r.downStream.ID, sender.GetStream().ID())

	// todo: check if we get a reset on send headers
}
