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

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// types.StreamEventListener
// types.StreamReceiver
// types.PoolEventListener
type shadowUpstreamRequest struct {
	host               types.Host
	requestSender      types.StreamSender
	connPool           types.ConnectionPool
	downStream         *downStream
	context            context.Context
	logger             log.Logger
	downstreamID       string
	DownstreamProtocol string
	UpstreamProtocol   string
	sendComplete       bool
	dataSent           bool
	trailerSent        bool
}

// on upstream per req timeout
func (r *shadowUpstreamRequest) resetStream() {
	// only reset a alive request sender stream
	r.ResetStream(types.StreamConnectionFailed)
}

// types.StreamEventListener
// Called by stream layer normally
func (r *shadowUpstreamRequest) OnResetStream(reason types.StreamResetReason) {
	r.ResetStream(reason)
}

// types.StreamReceiver
// Method to decode upstream's response message
func (r *shadowUpstreamRequest) OnReceiveHeaders(context context.Context, headers types.HeaderMap, endStream bool) {
	r.logger.Debugf("shadowUpstreamRequest get response, reset now")
	r.ResetStream(types.StreamLocalReset)
}

func (r *shadowUpstreamRequest) OnReceiveData(context context.Context, data types.IoBuffer, endStream bool) {
}

func (r *shadowUpstreamRequest) OnReceiveTrailers(context context.Context, trailers types.HeaderMap) {
}

func (r *shadowUpstreamRequest) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
	r.logger.Debugf("shadowUpstreamRequest get response,but decode error, reset now")
	r.ResetStream(types.StreamLocalReset)
}

// ~~~ send request wrapper
func (r *shadowUpstreamRequest) appendHeaders(headers types.HeaderMap, endStream bool) {
	log.StartLogger.Tracef("upstream request encode headers")
	r.sendComplete = endStream

	log.StartLogger.Tracef("upstream request before conn pool new stream")
	r.connPool.NewStream(r.context, r.downstreamID, r, r)
}


func (r *shadowUpstreamRequest) appendData(data types.IoBuffer, endStream bool) {
	log.DefaultLogger.Debugf("upstream request encode data")
	r.sendComplete = endStream
	r.dataSent = true
	r.requestSender.AppendData(r.context, data, endStream)
}

func (r *shadowUpstreamRequest) appendTrailers(trailers types.HeaderMap) {
	log.DefaultLogger.Debugf("upstream request encode trailers")
	r.sendComplete = true
	r.trailerSent = true
	r.requestSender.AppendTrailers(r.context, trailers)
}

// types.PoolEventListener
func (r *shadowUpstreamRequest) OnFailure(streamID string, reason types.PoolFailureReason, host types.Host) {
	var resetReason types.StreamResetReason

	switch reason {
	case types.Overflow:
		resetReason = types.StreamOverflow
	case types.ConnectionFailure:
		resetReason = types.StreamConnectionFailed
	}

	r.ResetStream(resetReason)
}

func (r *shadowUpstreamRequest) OnReady(streamID string, sender types.StreamSender, host types.Host) {
	r.requestSender = sender
	r.requestSender.GetStream().AddEventListener(r)

	endStream := r.sendComplete && !r.dataSent && !r.trailerSent
	
	// we use converted header
	r.requestSender.AppendHeaders(r.context, r.downStream.downstreamReqHeaders, endStream)

	r.downStream.requestInfo.OnUpstreamHostSelected(host)
	r.downStream.requestInfo.SetUpstreamLocalAddress(host.Address())

	// todo: check if we get a reset on send headers
}

// release downstream, drop shadow traffic, when:
// 1. request timeout
// 2. request failue
// 3 requeset success, got response header
func (r *shadowUpstreamRequest) ResetStream(reason types.StreamResetReason) {
	r.logger.Debugf("shadowUpstreamRequest resetStream:", reason)
	if r.requestSender != nil {
		r.requestSender.GetStream().RemoveEventListener(r)
		r.requestSender.GetStream().ResetStream(types.StreamLocalReset)
	}
	
	r.requestSender = nil
	r.downStream.cleanUp()
	r.downStream = nil
}