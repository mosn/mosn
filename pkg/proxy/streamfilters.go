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

func (s *downStream) addEncodedData(filter *activeStreamSenderFilter, data types.IoBuffer, streaming bool) {
	if s.filterStage == 0 || s.filterStage&EncodeHeaders > 0 ||
		s.filterStage&EncodeData > 0 {

		filter.handleBufferData(data)
	} else if s.filterStage&EncodeTrailers > 0 {
		s.runAppendDataFilters(filter, data, false)
	}
}

func (s *downStream) addDecodedData(filter *activeStreamReceiverFilter, data types.IoBuffer, streaming bool) {
	if s.filterStage == 0 || s.filterStage&DecodeHeaders > 0 ||
		s.filterStage&DecodeData > 0 {

		filter.handleBufferData(data)
	} else if s.filterStage&EncodeTrailers > 0 {
		s.runReceiveDataFilters(filter, data, false)
	}
}

func (s *downStream) runAppendHeaderFilters(filter *activeStreamSenderFilter, headers types.HeaderMap, endStream bool) bool {
	var index int
	var f *activeStreamSenderFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.senderFilters); index++ {
		f = s.senderFilters[index]

		s.filterStage |= EncodeHeaders
		status := f.filter.AppendHeaders(headers, endStream)
		s.filterStage &= ^EncodeHeaders

		if status == types.StreamHeadersFilterStop {
			f.stopped = true

			return true
		}

		f.headersContinued = true

		return false
	}

	return false
}

func (s *downStream) runAppendDataFilters(filter *activeStreamSenderFilter, data types.IoBuffer, endStream bool) bool {
	var index int
	var f *activeStreamSenderFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.senderFilters); index++ {
		f = s.senderFilters[index]

		s.filterStage |= EncodeData
		status := f.filter.AppendData(data, endStream)
		s.filterStage &= ^EncodeData

		if status == types.StreamDataFilterContinue {
			if f.stopped {
				f.handleBufferData(data)
				f.doContinue()

				return true
			}
		} else {
			f.stopped = true

			switch status {
			case types.StreamDataFilterStopAndBuffer:
				f.handleBufferData(data)
			case types.StreamDataFilterStop:
				f.stoppedNoBuf = true
				// make sure no data banked up
				data.Reset()
			}

			return true
		}
	}

	return false
}

func (s *downStream) runAppendTrailersFilters(filter *activeStreamSenderFilter, trailers types.HeaderMap) bool {
	var index int
	var f *activeStreamSenderFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.senderFilters); index++ {
		f = s.senderFilters[index]

		s.filterStage |= EncodeTrailers
		status := f.filter.AppendTrailers(trailers)
		s.filterStage &= ^EncodeTrailers

		if status == types.StreamTrailersFilterContinue {
			if f.stopped {
				f.doContinue()

				return true
			}
		} else {
			return true
		}
	}

	return false
}

func (s *downStream) runReceiveHeadersFilters(filter *activeStreamReceiverFilter, headers types.HeaderMap, endStream bool) bool {
	var index int
	var f *activeStreamReceiverFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.receiverFilters); index++ {
		f = s.receiverFilters[index]

		s.filterStage |= DecodeHeaders
		status := f.filter.OnDecodeHeaders(headers, endStream)
		s.filterStage &= ^DecodeHeaders

		if status == types.StreamHeadersFilterStop {
			f.stopped = true

			return true
		}

		f.headersContinued = true

		return false
	}

	return false
}

func (s *downStream) runReceiveDataFilters(filter *activeStreamReceiverFilter, data types.IoBuffer, endStream bool) bool {
	if s.upstreamProcessDone {
		return false
	}

	var index int
	var f *activeStreamReceiverFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.receiverFilters); index++ {
		f = s.receiverFilters[index]

		s.filterStage |= DecodeData
		status := f.filter.OnDecodeData(data, endStream)
		s.filterStage &= ^DecodeData

		if status == types.StreamDataFilterContinue {
			if f.stopped {
				f.handleBufferData(data)
				f.doContinue()

				return false
			}
		} else {
			f.stopped = true

			switch status {
			case types.StreamDataFilterStopAndBuffer:
				f.handleBufferData(data)
			case types.StreamDataFilterStop:
				f.stoppedNoBuf = true
				// make sure no data banked up
				data.Reset()
			}

			return true
		}
	}

	return false
}

func (s *downStream) runReceiveTrailersFilters(filter *activeStreamReceiverFilter, trailers types.HeaderMap) bool {
	if s.upstreamProcessDone {
		return false
	}

	var index int
	var f *activeStreamReceiverFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.receiverFilters); index++ {
		f = s.receiverFilters[index]

		s.filterStage |= DecodeTrailers
		status := f.filter.OnDecodeTrailers(trailers)
		s.filterStage &= ^DecodeTrailers

		if status == types.StreamTrailersFilterContinue {
			if f.stopped {
				f.doContinue()

				return false
			}
		} else {
			return true
		}
	}

	return false
}

// FilterStage is the type of the filter stage
type FilterStage int

// Const of all stages
const (
	DecodeHeaders = iota
	DecodeData
	DecodeTrailers
	EncodeHeaders
	EncodeData
	EncodeTrailers
)

type activeStreamFilter struct {
	index int

	activeStream     *downStream
	stopped          bool
	stoppedNoBuf     bool
	headersContinued bool
}

func (f *activeStreamFilter) Connection() types.Connection {
	return f.activeStream.proxy.readCallbacks.Connection()
}

func (f *activeStreamFilter) ResetStream() {
	f.activeStream.resetStream()
}

func (f *activeStreamFilter) Route() types.Route {
	return f.activeStream.route
}

func (f *activeStreamFilter) StreamID() string {
	return f.activeStream.streamID
}

func (f *activeStreamFilter) RequestInfo() types.RequestInfo {
	return f.activeStream.requestInfo
}

// types.StreamReceiverFilterCallbacks
type activeStreamReceiverFilter struct {
	activeStreamFilter

	filter types.StreamReceiverFilter
}

func newActiveStreamReceiverFilter(idx int, activeStream *downStream,
	filter types.StreamReceiverFilter) *activeStreamReceiverFilter {
	f := &activeStreamReceiverFilter{
		activeStreamFilter: activeStreamFilter{
			index:        idx,
			activeStream: activeStream,
		},
		filter: filter,
	}
	filter.SetDecoderFilterCallbacks(f)

	return f
}

func (f *activeStreamReceiverFilter) ContinueDecoding() {
	f.doContinue()
}

func (f *activeStreamReceiverFilter) doContinue() {
	if f.activeStream.upstreamProcessDone {
		return
	}

	f.stopped = false
	hasBuffedData := f.activeStream.downstreamReqDataBuf != nil
	hasTrailer := f.activeStream.downstreamReqTrailers != nil

	if !f.headersContinued {
		f.headersContinued = true

		endStream := f.activeStream.downstreamRecvDone && !hasBuffedData && !hasTrailer
		f.activeStream.doReceiveHeaders(f, f.activeStream.downstreamReqHeaders, endStream)
	}

	if hasBuffedData || f.stoppedNoBuf {
		if f.stoppedNoBuf || f.activeStream.downstreamReqDataBuf == nil {
			f.activeStream.downstreamReqDataBuf = buffer.NewIoBuffer(0)
		}

		endStream := f.activeStream.downstreamRecvDone && !hasTrailer
		f.activeStream.doReceiveData(f, f.activeStream.downstreamReqDataBuf, endStream)
	}

	if hasTrailer {
		f.activeStream.doReceiveTrailers(f, f.activeStream.downstreamReqTrailers)
	}
}

func (f *activeStreamReceiverFilter) handleBufferData(buf types.IoBuffer) {
	if f.activeStream.downstreamReqDataBuf != buf {
		if f.activeStream.downstreamReqDataBuf == nil {
			f.activeStream.downstreamReqDataBuf = buffer.NewIoBuffer(buf.Len())
		}

		f.activeStream.downstreamReqDataBuf.ReadFrom(buf)
	}
}

func (f *activeStreamReceiverFilter) DecodingBuffer() types.IoBuffer {
	return f.activeStream.downstreamReqDataBuf
}

func (f *activeStreamReceiverFilter) AddDecodedData(buf types.IoBuffer, streamingFilter bool) {
	f.activeStream.addDecodedData(f, buf, streamingFilter)
}

func (f *activeStreamReceiverFilter) AppendHeaders(headers types.HeaderMap, endStream bool) {
	f.activeStream.downstreamRespHeaders = headers
	f.activeStream.doAppendHeaders(nil, headers, endStream)
}

func (f *activeStreamReceiverFilter) AppendData(buf types.IoBuffer, endStream bool) {
	f.activeStream.doAppendData(nil, buf, endStream)
}

func (f *activeStreamReceiverFilter) AppendTrailers(trailers types.HeaderMap) {
	f.activeStream.downstreamRespTrailers = trailers
	f.activeStream.doAppendTrailers(nil, trailers)
}

func (f *activeStreamReceiverFilter) SetDecoderBufferLimit(limit uint32) {
	f.activeStream.setBufferLimit(limit)
}

func (f *activeStreamReceiverFilter) DecoderBufferLimit() uint32 {
	return f.activeStream.bufferLimit
}

// types.StreamSenderFilterCallbacks
type activeStreamSenderFilter struct {
	activeStreamFilter

	filter types.StreamSenderFilter
}

func newActiveStreamSenderFilter(idx int, activeStream *downStream,
	filter types.StreamSenderFilter) *activeStreamSenderFilter {
	f := &activeStreamSenderFilter{
		activeStreamFilter: activeStreamFilter{
			index:        idx,
			activeStream: activeStream,
		},
		filter: filter,
	}

	filter.SetEncoderFilterCallbacks(f)

	return f
}

func (f *activeStreamSenderFilter) ContinueEncoding() {
	f.doContinue()
}

func (f *activeStreamSenderFilter) doContinue() {
	f.stopped = false
	hasBuffedData := f.activeStream.downstreamRespDataBuf != nil
	hasTrailer := f.activeStream.downstreamRespTrailers == nil

	if !f.headersContinued {
		f.headersContinued = true
		endStream := f.activeStream.upstreamProcessDone && !hasBuffedData && !hasTrailer
		f.activeStream.doAppendHeaders(f, f.activeStream.downstreamRespHeaders, endStream)
	}

	if hasBuffedData || f.stoppedNoBuf {
		if f.stoppedNoBuf || f.activeStream.downstreamRespDataBuf == nil {
			f.activeStream.downstreamRespDataBuf = buffer.NewIoBuffer(0)
		}

		endStream := f.activeStream.downstreamRecvDone && !hasTrailer
		f.activeStream.doAppendData(f, f.activeStream.downstreamRespDataBuf, endStream)
	}

	if hasTrailer {
		f.activeStream.doAppendTrailers(f, f.activeStream.downstreamRespTrailers)
	}
}

func (f *activeStreamSenderFilter) handleBufferData(buf types.IoBuffer) {
	if f.activeStream.downstreamRespDataBuf != buf {
		if f.activeStream.downstreamRespDataBuf == nil {
			f.activeStream.downstreamRespDataBuf = buffer.NewIoBuffer(buf.Len())
		}

		f.activeStream.downstreamRespDataBuf.ReadFrom(buf)
	}
}

func (f *activeStreamSenderFilter) EncodingBuffer() types.IoBuffer {
	return f.activeStream.downstreamRespDataBuf
}

func (f *activeStreamSenderFilter) AddEncodedData(buf types.IoBuffer, streamingFilter bool) {
	f.activeStream.addEncodedData(f, buf, streamingFilter)
}

func (f *activeStreamSenderFilter) SetEncoderBufferLimit(limit uint32) {
	f.activeStream.setBufferLimit(limit)
}

func (f *activeStreamSenderFilter) EncoderBufferLimit() uint32 {
	return f.activeStream.bufferLimit
}
