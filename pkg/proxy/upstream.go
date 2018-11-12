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
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// types.StreamEventListener
// types.StreamReceiver
// types.PoolEventListener
type upstreamRequest struct {
	proxy         *proxy
	element       *list.Element
	downStream    *downStream
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
}

// reset upstream request in proxy context
// 1. downstream cleanup
// 2. on upstream global timeout
// 3. on upstream per req timeout
// 4. on upstream response receive error
// 5. before a retry
func (r *upstreamRequest) resetStream() {
	// only reset a alive request sender stream
	if r.requestSender != nil {
		r.requestSender.GetStream().RemoveEventListener(r)
		r.requestSender.GetStream().ResetStream(types.StreamLocalReset)
	}
}

// types.StreamEventListener
// Called by stream layer normally
func (r *upstreamRequest) OnResetStream(reason types.StreamResetReason) {
	workerPool.Offer(&resetEvent{
		streamEvent: streamEvent{
			direction: Upstream,
			streamID:  r.downStream.streamID,
			stream:    r.downStream,
		},
		reason: reason,
	})
}

func (r *upstreamRequest) ResetStream(reason types.StreamResetReason) {
	r.requestSender = nil

	if !r.setupRetry {
		// todo: check if we get a reset on encode request headers. e.g. send failed
		r.downStream.onUpstreamReset(UpstreamReset, reason)
	}
}

// types.StreamReceiver
// Method to decode upstream's response message
func (r *upstreamRequest) OnReceiveHeaders(context context.Context, headers types.HeaderMap, endStream bool) {
	// save response code
	if status, ok := headers.Get(protocol.MosnResponseStatusCode); ok {
		if code, err := strconv.Atoi(status); err == nil {
			r.downStream.requestInfo.SetResponseCode(uint32(code))
		}
		headers.Del(protocol.MosnResponseStatusCode)
	}

	buffer.TransmitBufferPoolContext(r.downStream.context, context)

	workerPool.Offer(&receiveHeadersEvent{
		streamEvent: streamEvent{
			direction: Upstream,
			streamID:  r.downStream.streamID,
			stream:    r.downStream,
		},
		headers:   headers,
		endStream: endStream,
	})
}

func (r *upstreamRequest) ReceiveHeaders(headers types.HeaderMap, endStream bool) {
	r.upstreamRespHeaders = headers
	r.downStream.onUpstreamHeaders(headers, endStream)
}

func (r *upstreamRequest) OnReceiveData(context context.Context, data types.IoBuffer, endStream bool) {
	r.downStream.downstreamRespDataBuf = data.Clone()
	data.Drain(data.Len())

	workerPool.Offer(&receiveDataEvent{
		streamEvent: streamEvent{
			direction: Upstream,
			streamID:  r.downStream.streamID,
			stream:    r.downStream,
		},
		data:      r.downStream.downstreamRespDataBuf,
		endStream: endStream,
	})
}

func (r *upstreamRequest) ReceiveData(data types.IoBuffer, endStream bool) {
	if !r.setupRetry {
		r.downStream.onUpstreamData(data, endStream)
	}
}

func (r *upstreamRequest) OnReceiveTrailers(context context.Context, trailers types.HeaderMap) {
	workerPool.Offer(&receiveTrailerEvent{
		streamEvent: streamEvent{
			direction: Upstream,
			streamID:  r.downStream.streamID,
			stream:    r.downStream,
		},
		trailers: trailers,
	})
}

func (r *upstreamRequest) ReceiveTrailers(trailers types.HeaderMap) {
	if !r.setupRetry {
		r.downStream.onUpstreamTrailers(trailers)
	}
}

func (r *upstreamRequest) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
	r.OnResetStream(types.StreamLocalReset)
}

// ~~~ send request wrapper
func (r *upstreamRequest) appendHeaders(headers types.HeaderMap, endStream bool) {
	log.StartLogger.Tracef("upstream request encode headers")
	r.sendComplete = endStream

	log.StartLogger.Tracef("upstream request before conn pool new stream")
	r.connPool.NewStream(r.downStream.context, r.downStream.streamID, r, r)
}

func (r *upstreamRequest) convertHeader(headers types.HeaderMap) types.HeaderMap {
	dp := types.Protocol(r.proxy.config.DownstreamProtocol)
	up := types.Protocol(r.proxy.config.UpstreamProtocol)

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
	log.DefaultLogger.Debugf("upstream request encode data")
	r.sendComplete = endStream
	r.dataSent = true
	r.requestSender.AppendData(r.downStream.context, r.convertData(data), endStream)
}

func (r *upstreamRequest) convertData(data types.IoBuffer) types.IoBuffer {
	dp := types.Protocol(r.proxy.config.DownstreamProtocol)
	up := types.Protocol(r.proxy.config.UpstreamProtocol)

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
	log.DefaultLogger.Debugf("upstream request encode trailers")
	r.sendComplete = true
	r.trailerSent = true
	r.requestSender.AppendTrailers(r.downStream.context, trailers)
}

func (r *upstreamRequest) convertTrailer(trailers types.HeaderMap) types.HeaderMap {
	dp := types.Protocol(r.proxy.config.DownstreamProtocol)
	up := types.Protocol(r.proxy.config.UpstreamProtocol)

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
func (r *upstreamRequest) OnFailure(streamID string, reason types.PoolFailureReason, host types.Host) {
	var resetReason types.StreamResetReason

	switch reason {
	case types.Overflow:
		resetReason = types.StreamOverflow
	case types.ConnectionFailure:
		resetReason = types.StreamConnectionFailed
	}

	r.ResetStream(resetReason)
}

func (r *upstreamRequest) OnReady(streamID string, sender types.StreamSender, host types.Host) {
	r.requestSender = sender
	r.requestSender.GetStream().AddEventListener(r)

	endStream := r.sendComplete && !r.dataSent && !r.trailerSent
	r.requestSender.AppendHeaders(r.downStream.context, r.convertHeader(r.downStream.downstreamReqHeaders), endStream)

	r.downStream.requestInfo.OnUpstreamHostSelected(host)
	r.downStream.requestInfo.SetUpstreamLocalAddress(host.Address())

	// todo: check if we get a reset on send headers
}
