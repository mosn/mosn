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
	"context"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

const ShadowHeaderKey = "x-mosn-shadow"

// shadow upstream request will send request to shadow upstream
type shadowUpstreamRequest struct {
	proxy         *proxy
	downStream    *shadowDownstream
	host          types.Host
	requestSender types.StreamSender
	connPool      types.ConnectionPool

	sendComplete bool
	dataSent     bool
	trailerSent  bool
}

func (r *shadowUpstreamRequest) resetStream() {
	r.downStream.logger.Tracef("shadow upstream reset stream id = %s", r.downStream.streamID)
	if r.requestSender != nil {
		r.requestSender.GetStream().RemoveEventListener(r)
		r.requestSender.GetStream().ResetStream(types.StreamLocalReset)
	}
}

func (r *shadowUpstreamRequest) OnResetStream(reason types.StreamResetReason) {
	r.requestSender = nil
	r.downStream.onUpstreamReset(UpstreamReset, reason)
}

func (r *shadowUpstreamRequest) response() {
	r.downStream.upstreamProcessDone = true
	r.downStream.endStream()

}

func (r *shadowUpstreamRequest) OnReceiveHeaders(context context.Context, headers types.HeaderMap, endStream bool) {
	r.downStream.logger.Tracef("shadow upstream receive headers, stream id = %s , endstream=%v", r.downStream.streamID, endStream)
	if endStream {
		r.response()
	}
}

func (r *shadowUpstreamRequest) OnReceiveData(context context.Context, data types.IoBuffer, endStream bool) {
	r.downStream.logger.Tracef("shadow upstream receive data, stream id = %s , endstream=%v", r.downStream.streamID, endStream)
	if endStream {
		r.response()
	}
}

func (r *shadowUpstreamRequest) OnReceiveTrailers(context context.Context, trailers types.HeaderMap) {
	r.downStream.logger.Tracef("shadow upstream receive trailers,  stream id = %s ", r.downStream.streamID)
	r.response()
}

func (r *shadowUpstreamRequest) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
	r.OnResetStream(types.StreamLocalReset)
}

func (r *shadowUpstreamRequest) appendHeaders(headers types.HeaderMap, endStream bool) {
	r.sendComplete = endStream
	r.connPool.NewStream(r.downStream.context, r.downStream.streamID, r, r)
}

func (r *shadowUpstreamRequest) appendData(data types.IoBuffer, endStream bool) {
	// if upstream is unreachable, the appendHeader->OnFailure, requestSender is nil
	// ignore the data
	if r.requestSender != nil {
		r.sendComplete = endStream
		r.dataSent = true
		r.requestSender.AppendData(r.downStream.context, r.convertData(data), endStream)
	}
}

func (r *shadowUpstreamRequest) appendTrailers(trailers types.HeaderMap) {
	// if upstream is unreachable, the appendHeader->OnFailure, requestSender is nil
	// ignore the data
	if r.requestSender != nil {
		r.sendComplete = true
		r.trailerSent = true
		r.requestSender.AppendTrailers(r.downStream.context, r.convertTrailer(trailers))
	}
}

func (r *shadowUpstreamRequest) OnFailure(streamID string, reason types.PoolFailureReason, host types.Host) {
	var resetReason types.StreamResetReason

	switch reason {
	case types.Overflow:
		resetReason = types.StreamOverflow
	case types.ConnectionFailure:
		resetReason = types.StreamConnectionFailed
	}

	r.OnResetStream(resetReason)
}

func (r *shadowUpstreamRequest) OnReady(streamID string, sender types.StreamSender, host types.Host) {
	r.requestSender = sender
	r.requestSender.GetStream().AddEventListener(r)

	endStream := r.sendComplete && !r.dataSent && !r.trailerSent

	// headers must deep copy.  becasue the AppendHeaders maybe modify the original headers
	// and we may add a shadow key for shadow copy
	headers := r.downStream.downstreamReqHeaders.Clone()
	// TODO: Add ShadowHeaderKey into headers. value is ?
	r.requestSender.AppendHeaders(r.downStream.context, r.convertHeader(headers), endStream)

	// todo: check if we get a reset on send headers
}

func (r *shadowUpstreamRequest) convertHeader(headers types.HeaderMap) types.HeaderMap {
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

func (r *shadowUpstreamRequest) convertData(data types.IoBuffer) types.IoBuffer {
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

func (r *shadowUpstreamRequest) convertTrailer(trailers types.HeaderMap) types.HeaderMap {
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
