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
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

// run stream append filters
func (s *downStream) runAppendFilters(p types.Phase, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	for ; s.senderFiltersIndex < len(s.senderFilters); s.senderFiltersIndex++ {
		f := s.senderFilters[s.senderFiltersIndex]

		status := f.filter.Append(s.context, headers, data, trailers)
		switch status {
		case api.StreamFilterStop:
			return
		case api.StreamFiltertermination:
			s.cleanStream()
			return
		default:
		}
	}
	s.senderFiltersIndex = 0
	return
}

// run stream receive filters
func (s *downStream) runReceiveFilters(p types.Phase, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	for ; s.receiverFiltersIndex < len(s.receiverFilters); s.receiverFiltersIndex++ {
		f := s.receiverFilters[s.receiverFiltersIndex]
		if f.p != p {
			continue
		}

		s.context = mosnctx.WithValue(s.context, types.ContextKeyStreamFilterPhase, p)

		status := f.filter.OnReceive(s.context, headers, data, trailers)
		switch status {
		case api.StreamFilterStop:
			return
		case api.StreamFiltertermination:
			s.cleanStream()
			return
		case api.StreamFilterReMatchRoute:
			// Retry only at the DownFilterAfterRoute phase
			if p == types.DownFilterAfterRoute {
				// FiltersIndex is not increased until no retry is required
				s.receiverFiltersAgainPhase = types.MatchRoute
			} else {
				s.receiverFiltersIndex++
			}
			return
		case api.StreamFilterReChooseHost:
			// Retry only at the DownFilterAfterChooseHost phase
			if p == types.DownFilterAfterChooseHost {
				// FiltersIndex is not increased until no retry is required
				s.receiverFiltersAgainPhase = types.ChooseHost
			} else {
				s.receiverFiltersIndex++
			}
			return
		}

	}

	s.receiverFiltersIndex = 0
	return
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
	p types.Phase
	activeStreamFilter
	filter api.StreamReceiverFilter
}

func newActiveStreamReceiverFilter(activeStream *downStream,
	filter api.StreamReceiverFilter, p types.Phase) *activeStreamReceiverFilter {
	f := &activeStreamReceiverFilter{
		activeStreamFilter: activeStreamFilter{
			activeStream: activeStream,
		},
		filter: filter,
		p:      p,
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

func (f *activeStreamReceiverFilter) SendDirectResponse(headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {
	atomic.StoreUint32(&f.activeStream.reuseBuffer, 0)
	f.activeStream.noConvert = true
	f.activeStream.downstreamRespHeaders = headers
	f.activeStream.downstreamRespDataBuf = buf
	f.activeStream.downstreamRespTrailers = trailers
	f.activeStream.directResponse = true
}

func (f *activeStreamReceiverFilter) SetConvert(on bool) {
	f.activeStream.noConvert = !on
}

// GetFilterCurrentPhase get current phase for filter
func (f *activeStreamReceiverFilter) GetFilterCurrentPhase() api.FilterPhase {
	// default AfterRoute
	p := api.AfterRoute

	switch f.p {
	case types.DownFilter:
		p = api.BeforeRoute
	case types.DownFilterAfterRoute:
		p = api.AfterRoute
	case types.DownFilterAfterChooseHost:
		p = api.AfterChooseHost
	}

	return p
}

// types.StreamSenderFilterHandler
type activeStreamSenderFilter struct {
	activeStreamFilter

	filter api.StreamSenderFilter
}

func newActiveStreamSenderFilter(activeStream *downStream,
	filter api.StreamSenderFilter) *activeStreamSenderFilter {
	f := &activeStreamSenderFilter{
		activeStreamFilter: activeStreamFilter{
			activeStream: activeStream,
		},
		filter: filter,
	}

	filter.SetSenderFilterHandler(f)

	return f
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
