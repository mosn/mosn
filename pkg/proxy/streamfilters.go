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
	"sync/atomic"

	"mosn.io/api"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

// proxy-specified implementation of interface StreamFilterChain.
type streamFilterChain struct {
	downStream *downStream

	*streamfilter.DefaultStreamFilterChainImpl
}

func (sfc *streamFilterChain) init(s *downStream) {
	sfc.downStream = s
	sfc.DefaultStreamFilterChainImpl = streamfilter.GetDefaultStreamFilterChain()
}

func (sfc *streamFilterChain) AddStreamSenderFilter(filter api.StreamSenderFilter, phase api.SenderFilterPhase) {
	handler := newStreamSenderFilterHandler(sfc.downStream)
	filter.SetSenderFilterHandler(handler)
	sfc.DefaultStreamFilterChainImpl.AddStreamSenderFilter(filter, phase)
}

func (sfc *streamFilterChain) AddStreamReceiverFilter(filter api.StreamReceiverFilter, phase api.ReceiverFilterPhase) {
	handler := newStreamReceiverFilterHandler(sfc.downStream)
	filter.SetReceiveFilterHandler(handler)
	sfc.DefaultStreamFilterChainImpl.AddStreamReceiverFilter(filter, phase)
}

func (sfc *streamFilterChain) AddStreamAccessLog(accessLog api.AccessLog) {
	if sfc.downStream.proxy != nil {
		sfc.DefaultStreamFilterChainImpl.AddStreamAccessLog(accessLog)
	}
}

// the stream filter chain are not allowed to be used anymore after calling this func
func (sfc *streamFilterChain) destroy() {
	// filter destroy
	sfc.DefaultStreamFilterChainImpl.OnDestroy()

	// reset fields
	streamfilter.PutStreamFilterChain(sfc.DefaultStreamFilterChainImpl)
	sfc.downStream = nil
	sfc.DefaultStreamFilterChainImpl = nil
}

// implement api.StreamFilterHandler.
type streamFilterHandlerBase struct {
	activeStream *downStream
}

func (f *streamFilterHandlerBase) Connection() api.Connection {
	return f.activeStream.proxy.readCallbacks.Connection()
}

func (f *streamFilterHandlerBase) Route() types.Route {
	return f.activeStream.route
}

func (f *streamFilterHandlerBase) RequestInfo() types.RequestInfo {
	return f.activeStream.requestInfo
}

// implement api.StreamReceiverFilterHandler.
type streamReceiverFilterHandler struct {
	streamFilterHandlerBase

	id uint32
}

func newStreamReceiverFilterHandler(activeStream *downStream) *streamReceiverFilterHandler {
	f := &streamReceiverFilterHandler{
		streamFilterHandlerBase: streamFilterHandlerBase{
			activeStream: activeStream,
		},
		id: activeStream.ID,
	}

	return f
}

func (f *streamReceiverFilterHandler) AppendHeaders(headers types.HeaderMap, endStream bool) {
	f.activeStream.downstreamRespHeaders = headers
	f.activeStream.appendHeaders(endStream)
}

func (f *streamReceiverFilterHandler) AppendData(buf types.IoBuffer, endStream bool) {
	f.activeStream.downstreamRespDataBuf = buf
	f.activeStream.appendData(endStream)
}

func (f *streamReceiverFilterHandler) AppendTrailers(trailers types.HeaderMap) {
	f.activeStream.downstreamRespTrailers = trailers
	f.activeStream.appendTrailers()
}

func (f *streamReceiverFilterHandler) SendHijackReply(code int, headers types.HeaderMap) {
	f.activeStream.sendHijackReply(code, headers)
}

func (f *streamReceiverFilterHandler) SendHijackReplyWithBody(code int, headers types.HeaderMap, body string) {
	f.activeStream.sendHijackReplyWithBody(code, headers, body)
}

func (f *streamReceiverFilterHandler) SendDirectResponse(headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {
	atomic.StoreUint32(&f.activeStream.reuseBuffer, 0)
	f.activeStream.downstreamRespHeaders = headers
	f.activeStream.downstreamRespDataBuf = buf
	f.activeStream.downstreamRespTrailers = trailers
	f.activeStream.directResponse = true
}

func (f *streamReceiverFilterHandler) TerminateStream(code int) bool {
	s := f.activeStream
	atomic.StoreUint32(&s.reuseBuffer, 0)

	if s.downstreamRespHeaders != nil {
		return false
	}
	if atomic.LoadUint32(&s.downstreamCleaned) == 1 {
		return false
	}
	if f.id != s.ID {
		return false
	}
	if !atomic.CompareAndSwapUint32(&s.upstreamResponseReceived, 0, 1) {
		return false
	}
	// stop timeout timer
	if s.responseTimer != nil {
		s.responseTimer.Stop()
	}
	if s.perRetryTimer != nil {
		s.perRetryTimer.Stop()
	}
	// send hijacks response, request finished
	s.requestInfo.SetResponseFlag(api.DownStreamTerminate)
	s.sendHijackReply(code, f.activeStream.downstreamReqHeaders)
	s.sendNotify() // wake up proxy workflow
	return true
}

// DEPRECATED: remove me
func (f *streamReceiverFilterHandler) SetConvert(on bool) {
}

// GetFilterCurrentPhase get current phase for filter
func (f *streamReceiverFilterHandler) GetFilterCurrentPhase() api.ReceiverFilterPhase {
	// default AfterRoute
	p := api.AfterRoute

	switch f.activeStream.phase {
	case types.DownFilter:
		p = api.BeforeRoute
	case types.DownFilterAfterRoute:
		p = api.AfterRoute
	case types.DownFilterAfterChooseHost:
		p = api.AfterChooseHost
	}

	return p
}

// TODO: remove all of the following when proxy changed to single request @lieyuan
func (f *streamReceiverFilterHandler) GetRequestHeaders() types.HeaderMap {
	return f.activeStream.downstreamReqHeaders
}
func (f *streamReceiverFilterHandler) SetRequestHeaders(headers types.HeaderMap) {
	f.activeStream.downstreamReqHeaders = headers
}
func (f *streamReceiverFilterHandler) GetRequestData() types.IoBuffer {
	return f.activeStream.downstreamReqDataBuf
}

func (f *streamReceiverFilterHandler) SetRequestData(data types.IoBuffer) {
	// data is the original data. do nothing
	if f.activeStream.downstreamReqDataBuf == data {
		return
	}
	if f.activeStream.downstreamReqDataBuf == nil {
		f.activeStream.downstreamReqDataBuf = buffer.NewIoBuffer(0)
	}
	f.activeStream.downstreamReqDataBuf.Reset()
	f.activeStream.downstreamReqDataBuf.ReadFrom(data)
}

func (f *streamReceiverFilterHandler) GetRequestTrailers() types.HeaderMap {
	return f.activeStream.downstreamReqTrailers
}

func (f *streamReceiverFilterHandler) SetRequestTrailers(trailers types.HeaderMap) {
	f.activeStream.downstreamReqTrailers = trailers
}

// implement api.StreamSenderFilterHandler.
type streamSenderFilterHandler struct {
	streamFilterHandlerBase
}

func newStreamSenderFilterHandler(activeStream *downStream) *streamSenderFilterHandler {
	f := &streamSenderFilterHandler{
		streamFilterHandlerBase: streamFilterHandlerBase{
			activeStream: activeStream,
		},
	}

	return f
}

func (f *streamSenderFilterHandler) GetResponseHeaders() types.HeaderMap {
	return f.activeStream.downstreamRespHeaders
}

func (f *streamSenderFilterHandler) SetResponseHeaders(headers types.HeaderMap) {
	f.activeStream.downstreamRespHeaders = headers
}

func (f *streamSenderFilterHandler) GetResponseData() types.IoBuffer {
	return f.activeStream.downstreamRespDataBuf
}

func (f *streamSenderFilterHandler) SetResponseData(data types.IoBuffer) {
	// data is the original data. do nothing
	if f.activeStream.downstreamRespDataBuf == data {
		return
	}
	if f.activeStream.downstreamRespDataBuf == nil {
		f.activeStream.downstreamRespDataBuf = buffer.NewIoBuffer(0)
	}
	f.activeStream.downstreamRespDataBuf.Reset()
	f.activeStream.downstreamRespDataBuf.ReadFrom(data)
}

func (f *streamSenderFilterHandler) GetResponseTrailers() types.HeaderMap {
	return f.activeStream.downstreamRespTrailers
}

func (f *streamSenderFilterHandler) SetResponseTrailers(trailers types.HeaderMap) {
	f.activeStream.downstreamRespTrailers = trailers
}
