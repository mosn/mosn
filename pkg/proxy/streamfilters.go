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
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// run stream append filters
func (s *downStream) runAppendFilters(p types.Phase, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) bool {
	for ; s.senderFiltersIndex < len(s.senderFilters); s.senderFiltersIndex++ {
		f := s.senderFilters[s.senderFiltersIndex]

		status := f.filter.Append(s.context, headers, data, trailers)
		if status == types.StreamFilterStop {
			return true
		}
	}
	s.senderFiltersIndex = 0
	return false
}

// run stream receive filters
func (s *downStream) runReceiveFilters(p types.Phase, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) bool {
	for ; s.receiverFiltersIndex < len(s.receiverFilters); s.receiverFiltersIndex++ {
		f := s.receiverFilters[s.receiverFiltersIndex]
		if f.p != p {
			continue
		}

		status := f.filter.OnReceive(s.context, headers, data, trailers)
		if status == types.StreamFilterStop {
			return true
		}

		if status == types.StreamFilterReMatchRoute {
			s.receiverFiltersIndex++
			s.receiverFiltersAgain = true
			return false
		}
	}

	s.receiverFiltersIndex = 0
	return false
}

type activeStreamFilter struct {
	activeStream     *downStream
}

func (f *activeStreamFilter) Connection() types.Connection {
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
	filter types.StreamReceiverFilter
}

func newActiveStreamReceiverFilter(activeStream *downStream,
	filter types.StreamReceiverFilter, p types.Phase) *activeStreamReceiverFilter {
	if p != types.DownFilter && p != types.DownFilterAfterRoute {
		p = types.DownFilterAfterRoute
	}
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
	f.activeStream.doAppendHeaders(headers, endStream)
}

func (f *activeStreamReceiverFilter) AppendData(buf types.IoBuffer, endStream bool) {
	f.activeStream.downstreamRespDataBuf = buf
	f.activeStream.doAppendData(buf, endStream)
}

func (f *activeStreamReceiverFilter) AppendTrailers(trailers types.HeaderMap) {
	f.activeStream.downstreamRespTrailers = trailers
	f.activeStream.doAppendTrailers(trailers)
}

func (f *activeStreamReceiverFilter) SendHijackReply(code int, headers types.HeaderMap) {
	f.activeStream.sendHijackReply(code, headers)
}

// types.StreamSenderFilterHandler
type activeStreamSenderFilter struct {
	activeStreamFilter

	filter types.StreamSenderFilter
}


func newActiveStreamSenderFilter(activeStream *downStream,
	filter types.StreamSenderFilter) *activeStreamSenderFilter {
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
		f.activeStream.downstreamReqDataBuf.Count(1)
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
