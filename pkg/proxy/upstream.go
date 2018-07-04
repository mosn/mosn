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

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// types.StreamEventListener
// types.StreamReceiver
// types.PoolEventListener
type upstreamRequest struct {
	proxy         *proxy
	element       *list.Element
	activeStream  *downStream
	host          types.Host
	requestSender types.StreamSender
	connPool      types.ConnectionPool

	// ~~~ upstream response buf
	upstreamRespHeaders map[string]string

	//~~~ state
	appendComplete  bool
	dataAppended    bool
	trailerAppended bool
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
	r.requestSender = nil

	// todo: check if we get a reset on encode request headers. e.g. send failed
	r.activeStream.onUpstreamReset(UpstreamReset, reason)
}

// types.StreamReceiver
// Method to decode upstream's response message
func (r *upstreamRequest) OnReceiveHeaders(headers map[string]string, endStream bool) {
	r.upstreamRespHeaders = headers
	r.activeStream.onUpstreamHeaders(headers, endStream)
}

func (r *upstreamRequest) OnReceiveData(data types.IoBuffer, endStream bool) {
	r.activeStream.onUpstreamData(data, endStream)
}

func (r *upstreamRequest) OnReceiveTrailers(trailers map[string]string) {
	r.activeStream.onUpstreamTrailers(trailers)
}

func (r *upstreamRequest) OnDecodeError(err error, headers map[string]string) {
}

// ~~~ send request wrapper

func (r *upstreamRequest) appendHeaders(headers map[string]string, endStream bool) {
	log.StartLogger.Tracef("upstream request encode headers")
	r.appendComplete = endStream
	streamID := ""

	if streamid, ok := headers[types.HeaderStreamID]; ok {
		streamID = streamid
	}

	log.StartLogger.Tracef("upstream request before conn pool new stream")
	r.connPool.NewStream(r.proxy.context, streamID, r, r)
}

func (r *upstreamRequest) appendData(data types.IoBuffer, endStream bool) {
	log.DefaultLogger.Debugf("upstream request encode data")
	r.appendComplete = endStream
	r.dataAppended = true
	r.requestSender.AppendData(data, endStream)
}

func (r *upstreamRequest) appendTrailers(trailers map[string]string) {
	log.DefaultLogger.Debugf("upstream request encode trailers")
	r.appendComplete = true
	r.trailerAppended = true
	r.requestSender.AppendTrailers(trailers)
}

// types.PoolEventListener
func (r *upstreamRequest) OnFailure(streamId string, reason types.PoolFailureReason, host types.Host) {
	var resetReason types.StreamResetReason

	switch reason {
	case types.Overflow:
		resetReason = types.StreamOverflow
	case types.ConnectionFailure:
		resetReason = types.StreamConnectionFailed
	}

	r.OnResetStream(resetReason)
}

func (r *upstreamRequest) OnReady(streamId string, sender types.StreamSender, host types.Host) {
	r.requestSender = sender
	r.requestSender.GetStream().AddEventListener(r)

	endStream := r.appendComplete && !r.dataAppended && !r.trailerAppended
	r.requestSender.AppendHeaders(r.activeStream.downstreamReqHeaders, endStream)

	r.activeStream.requestInfo.OnUpstreamHostSelected(host)
	r.activeStream.requestInfo.SetUpstreamLocalAddress(host.Address())

	// todo: check if we get a reset on send headers
}
