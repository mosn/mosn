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
	"mosn.io/mosn/pkg/streamfilter"
	"sync/atomic"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

type streamFilterManager struct {
	downStream                *downStream
	receiverFiltersAgainPhase types.Phase

	streamfilter.DefaultStreamFilterChainImpl
}

func (manager *streamFilterManager) AddStreamSenderFilter(filter api.StreamSenderFilter, phase api.SenderFilterPhase) {
	sf := newActiveStreamSenderFilter(manager.downStream, filter)
	manager.DefaultStreamFilterChainImpl.AddStreamSenderFilter(sf, phase)
}

func (manager *streamFilterManager) AddStreamReceiverFilter(filter api.StreamReceiverFilter, phase api.ReceiverFilterPhase) {
	sf := newActiveStreamReceiverFilter(manager.downStream, filter)
	manager.DefaultStreamFilterChainImpl.AddStreamReceiverFilter(sf, phase)
}

func (manager *streamFilterManager) AddStreamAccessLog(accessLog api.AccessLog) {
	if manager.downStream.proxy != nil {
		manager.DefaultStreamFilterChainImpl.AddStreamAccessLog(accessLog)
	}
}

func (manager *streamFilterManager) RunReceiverFilter(ctx context.Context, phase api.ReceiverFilterPhase,
	headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
	statusHandler streamfilter.StreamFilterStatusHandler) api.StreamFilterStatus {

	return manager.DefaultStreamFilterChainImpl.RunReceiverFilter(ctx, phase, headers, data, trailers,
		func(status api.StreamFilterStatus) {
			switch status {
			case api.StreamFiltertermination:
				// no reuse buffer
				atomic.StoreUint32(&manager.downStream.reuseBuffer, 0)
				manager.downStream.cleanStream()
			case api.StreamFilterReMatchRoute:
				// Retry only at the AfterRoute phase
				if phase == api.AfterRoute {
					// FiltersIndex is not increased until no retry is required
					manager.receiverFiltersAgainPhase = types.MatchRoute
				}
			case api.StreamFilterReChooseHost:
				// Retry only at the AfterChooseHost phase
				if phase == api.AfterChooseHost {
					// FiltersIndex is not increased until no retry is required
					manager.receiverFiltersAgainPhase = types.ChooseHost
				}
			}
		})
}

func (manager *streamFilterManager) RunSenderFilter(ctx context.Context, phase api.SenderFilterPhase,
	headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
	statusHandler streamfilter.StreamFilterStatusHandler) api.StreamFilterStatus {

	return manager.DefaultStreamFilterChainImpl.RunSenderFilter(ctx, phase, headers, data, trailers,
		func(status api.StreamFilterStatus) {
			if status == api.StreamFiltertermination {
				// no reuse buffer
				atomic.StoreUint32(&manager.downStream.reuseBuffer, 0)
				manager.downStream.cleanStream()
			}
		})
}

type activeStreamFilter struct {
	activeStream *downStream
}

func (f *activeStreamFilter) Connection() api.Connection {
	return f.activeStream.proxy.readCallbacks.Connection()
}

func (f *activeStreamFilter) Route() types.Route {
	return f.activeStream.route
}

func (f *activeStreamFilter) RequestInfo() types.RequestInfo {
	return f.activeStream.requestInfo
}

// types.StreamReceiverFilter
// types.StreamReceiverFilterHandler
type activeStreamReceiverFilter struct {
	activeStreamFilter
	api.StreamReceiverFilter
	id uint32
}

func newActiveStreamReceiverFilter(activeStream *downStream,
	filter api.StreamReceiverFilter) *activeStreamReceiverFilter {
	f := &activeStreamReceiverFilter{
		activeStreamFilter: activeStreamFilter{
			activeStream: activeStream,
		},
		StreamReceiverFilter: filter,
		id:                   activeStream.ID,
	}
	filter.SetReceiveFilterHandler(f)

	return f
}

func (f *activeStreamReceiverFilter) AppendHeaders(headers types.HeaderMap, endStream bool) {
	f.activeStream.downstreamRespHeaders = headers
	f.activeStream.noConvert = true
	f.activeStream.appendHeaders(endStream)
}

func (f *activeStreamReceiverFilter) AppendData(buf types.IoBuffer, endStream bool) {
	f.activeStream.downstreamRespDataBuf = buf
	f.activeStream.noConvert = true
	f.activeStream.appendData(endStream)
}

func (f *activeStreamReceiverFilter) AppendTrailers(trailers types.HeaderMap) {
	f.activeStream.downstreamRespTrailers = trailers
	f.activeStream.noConvert = true
	f.activeStream.appendTrailers()
}

func (f *activeStreamReceiverFilter) SendHijackReply(code int, headers types.HeaderMap) {
	f.activeStream.sendHijackReply(code, headers)
}

func (f *activeStreamReceiverFilter) SendHijackReplyWithBody(code int, headers types.HeaderMap, body string) {
	f.activeStream.sendHijackReplyWithBody(code, headers, body)
}

func (f *activeStreamReceiverFilter) SendDirectResponse(headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {
	atomic.StoreUint32(&f.activeStream.reuseBuffer, 0)
	f.activeStream.noConvert = true
	f.activeStream.downstreamRespHeaders = headers
	f.activeStream.downstreamRespDataBuf = buf
	f.activeStream.downstreamRespTrailers = trailers
	f.activeStream.directResponse = true
}

func (f *activeStreamReceiverFilter) TerminateStream(code int) bool {
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
	s.sendHijackReply(code, f.activeStream.downstreamReqHeaders)
	s.sendNotify() // wake up proxy workflow
	return true
}

func (f *activeStreamReceiverFilter) SetConvert(on bool) {
	f.activeStream.noConvert = !on
}

// GetFilterCurrentPhase get current phase for filter
func (f *activeStreamReceiverFilter) GetFilterCurrentPhase() api.ReceiverFilterPhase {
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
func (f *activeStreamReceiverFilter) GetRequestHeaders() types.HeaderMap {
	return f.activeStream.downstreamReqHeaders
}
func (f *activeStreamReceiverFilter) SetRequestHeaders(headers types.HeaderMap) {
	f.activeStream.downstreamReqHeaders = headers
}
func (f *activeStreamReceiverFilter) GetRequestData() types.IoBuffer {
	return f.activeStream.downstreamReqDataBuf
}

func (f *activeStreamReceiverFilter) SetRequestData(data types.IoBuffer) {
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

func (f *activeStreamReceiverFilter) GetRequestTrailers() types.HeaderMap {
	return f.activeStream.downstreamReqTrailers
}

func (f *activeStreamReceiverFilter) SetRequestTrailers(trailers types.HeaderMap) {
	f.activeStream.downstreamReqTrailers = trailers
}

// types.StreamSenderFilterHandler
type activeStreamSenderFilter struct {
	activeStreamFilter

	api.StreamSenderFilter
}

func newActiveStreamSenderFilter(activeStream *downStream,
	filter api.StreamSenderFilter) *activeStreamSenderFilter {
	f := &activeStreamSenderFilter{
		activeStreamFilter: activeStreamFilter{
			activeStream: activeStream,
		},
		StreamSenderFilter: filter,
	}

	filter.SetSenderFilterHandler(f)

	return f
}

func (f *activeStreamSenderFilter) GetResponseHeaders() types.HeaderMap {
	return f.activeStream.downstreamRespHeaders
}

func (f *activeStreamSenderFilter) SetResponseHeaders(headers types.HeaderMap) {
	f.activeStream.downstreamRespHeaders = headers
}

func (f *activeStreamSenderFilter) GetResponseData() types.IoBuffer {
	return f.activeStream.downstreamRespDataBuf
}

func (f *activeStreamSenderFilter) SetResponseData(data types.IoBuffer) {
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

func (f *activeStreamSenderFilter) GetResponseTrailers() types.HeaderMap {
	return f.activeStream.downstreamRespTrailers
}

func (f *activeStreamSenderFilter) SetResponseTrailers(trailers types.HeaderMap) {
	f.activeStream.downstreamRespTrailers = trailers
}
