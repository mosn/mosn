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
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func (s *activeStream) addEncodedData(filter *activeStreamEncoderFilter, data types.IoBuffer, streaming bool) {
	if s.filterStage == 0 || s.filterStage&EncodeHeaders > 0 ||
		s.filterStage&EncodeData > 0 {
		s.encoderFiltersStreaming = streaming

		filter.handleBufferData(data)
	} else if s.filterStage&EncodeTrailers > 0 {
		s.encodeDataFilters(filter, data, false)
	}
}

func (s *activeStream) addDecodedData(filter *activeStreamDecoderFilter, data types.IoBuffer, streaming bool) {
	if s.filterStage == 0 || s.filterStage&DecodeHeaders > 0 ||
		s.filterStage&DecodeData > 0 {
		s.decoderFiltersStreaming = streaming

		filter.handleBufferData(data)
	} else if s.filterStage&EncodeTrailers > 0 {
		s.decodeDataFilters(filter, data, false)
	}
}

func (s *activeStream) encodeHeaderFilters(filter *activeStreamEncoderFilter, headers interface{}, endStream bool) bool {
	var index int
	var f *activeStreamEncoderFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.encoderFilters); index++ {
		f = s.encoderFilters[index]

		s.filterStage |= EncodeHeaders
		status := f.filter.EncodeHeaders(headers, endStream)
		s.filterStage &= ^EncodeHeaders

		if status == types.FilterHeadersStatusStopIteration {
			f.stopped = true

			return true
		} else {
			f.headersContinued = true

			return false
		}
	}

	return false
}

func (s *activeStream) encodeDataFilters(filter *activeStreamEncoderFilter, data types.IoBuffer, endStream bool) bool {
	var index int
	var f *activeStreamEncoderFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.encoderFilters); index++ {
		f = s.encoderFilters[index]

		s.filterStage |= EncodeData
		status := f.filter.EncodeData(data, endStream)
		s.filterStage &= ^EncodeData

		if status == types.FilterDataStatusContinue {
			if f.stopped {
				f.handleBufferData(data)
				f.doContinue()

				return true
			}
		} else {
			f.stopped = true

			switch status {
			case types.FilterDataStatusStopIterationAndBuffer,
				types.FilterDataStatusStopIterationAndWatermark:
				s.encoderFiltersStreaming = status == types.FilterDataStatusStopIterationAndWatermark
				f.handleBufferData(data)
			case types.FilterDataStatusStopIterationNoBuffer:
				f.stoppedNoBuf = true
				// make sure no data banked up
				data.Reset()
			}

			return true
		}
	}

	return false
}

func (s *activeStream) encodeTrailersFilters(filter *activeStreamEncoderFilter, trailers map[string]string) bool {
	var index int
	var f *activeStreamEncoderFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.encoderFilters); index++ {
		f = s.encoderFilters[index]

		s.filterStage |= EncodeTrailers
		status := f.filter.EncodeTrailers(trailers)
		s.filterStage &= ^EncodeTrailers

		if status == types.FilterTrailersStatusContinue {
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

func (s *activeStream) decodeHeaderFilters(filter *activeStreamDecoderFilter, headers map[string]string, endStream bool) bool {
	var index int
	var f *activeStreamDecoderFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.decoderFilters); index++ {
		f = s.decoderFilters[index]

		s.filterStage |= DecodeHeaders
		status := f.filter.DecodeHeaders(headers, endStream)
		s.filterStage &= ^DecodeHeaders

		if status == types.FilterHeadersStatusStopIteration {
			f.stopped = true

			return true
		} else {
			f.headersContinued = true

			return false
		}
	}

	return false
}

func (s *activeStream) decodeDataFilters(filter *activeStreamDecoderFilter, data types.IoBuffer, endStream bool) bool {
	if s.localProcessDone {
		return false
	}

	var index int
	var f *activeStreamDecoderFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.decoderFilters); index++ {
		f = s.decoderFilters[index]

		s.filterStage |= DecodeData
		status := f.filter.DecodeData(data, endStream)
		s.filterStage &= ^DecodeData

		if status == types.FilterDataStatusContinue {
			if f.stopped {
				f.handleBufferData(data)
				f.doContinue()

				return false
			}
		} else {
			f.stopped = true

			switch status {
			case types.FilterDataStatusStopIterationAndBuffer,
				types.FilterDataStatusStopIterationAndWatermark:
				s.decoderFiltersStreaming = status == types.FilterDataStatusStopIterationAndWatermark
				f.handleBufferData(data)
			case types.FilterDataStatusStopIterationNoBuffer:
				f.stoppedNoBuf = true
				// make sure no data banked up
				data.Reset()
			}

			return true
		}
	}

	return false
}

func (s *activeStream) decodeTrailersFilters(filter *activeStreamDecoderFilter, trailers map[string]string) bool {
	if s.localProcessDone {
		return false
	}

	var index int
	var f *activeStreamDecoderFilter

	if filter != nil {
		index = filter.index + 1
	}

	for ; index < len(s.decoderFilters); index++ {
		f = s.decoderFilters[index]

		s.filterStage |= DecodeTrailers
		status := f.filter.DecodeTrailers(trailers)
		s.filterStage &= ^DecodeTrailers

		if status == types.FilterTrailersStatusContinue {
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

type FilterStage int

const (
	DecodeHeaders = iota
	DecodeData
	DecodeTrailers
	EncodeHeaders
	EncodeData
	EncodeTrailers
)

// types.StreamFilterCallbacks
type activeStreamFilter struct {
	index int

	activeStream     *activeStream
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

func (f *activeStreamFilter) StreamId() string {
	return f.activeStream.streamId
}

func (f *activeStreamFilter) RequestInfo() types.RequestInfo {
	return f.activeStream.requestInfo
}

// types.StreamDecoderFilterCallbacks
type activeStreamDecoderFilter struct {
	activeStreamFilter

	filter types.StreamDecoderFilter
}

func newActiveStreamDecoderFilter(idx int, activeStream *activeStream,
	filter types.StreamDecoderFilter) *activeStreamDecoderFilter {
	f := &activeStreamDecoderFilter{
		activeStreamFilter: activeStreamFilter{
			index:        idx,
			activeStream: activeStream,
		},
		filter: filter,
	}
	filter.SetDecoderFilterCallbacks(f)

	return f
}

func (f *activeStreamDecoderFilter) ContinueDecoding() {
	f.doContinue()
}

func (f *activeStreamDecoderFilter) doContinue() {
	if f.activeStream.localProcessDone {
		return
	}

	f.stopped = false
	hasBuffedData := f.activeStream.downstreamReqDataBuf != nil
	hasTrailer := f.activeStream.downstreamReqTrailers != nil

	if !f.headersContinued {
		f.headersContinued = true

		endStream := f.activeStream.downstreamRecvDone && !hasBuffedData && !hasTrailer
		f.activeStream.doDecodeHeaders(f, f.activeStream.downstreamReqHeaders, endStream)
	}

	if hasBuffedData || f.stoppedNoBuf {
		if f.stoppedNoBuf || f.activeStream.downstreamReqDataBuf == nil {
			f.activeStream.downstreamReqDataBuf = buffer.NewIoBuffer(0)
		}

		endStream := f.activeStream.downstreamRecvDone && !hasTrailer
		f.activeStream.doDecodeData(f, f.activeStream.downstreamReqDataBuf, endStream)
	}

	if hasTrailer {
		f.activeStream.doDecodeTrailers(f, f.activeStream.downstreamReqTrailers)
	}
}

func (f *activeStreamDecoderFilter) handleBufferData(buf types.IoBuffer) {
	if f.activeStream.downstreamReqDataBuf != buf {
		if f.activeStream.downstreamReqDataBuf == nil {
			f.activeStream.downstreamReqDataBuf = buffer.NewIoBuffer(buf.Len())
		}

		f.activeStream.downstreamReqDataBuf.ReadFrom(buf)
	}
}

func (f *activeStreamDecoderFilter) DecodingBuffer() types.IoBuffer {
	return f.activeStream.downstreamReqDataBuf
}

func (f *activeStreamDecoderFilter) AddDecodedData(buf types.IoBuffer, streamingFilter bool) {
	f.activeStream.addDecodedData(f, buf, streamingFilter)
}

func (f *activeStreamDecoderFilter) EncodeHeaders(headers interface{}, endStream bool) {
	f.activeStream.downstreamRespHeaders = headers
	f.activeStream.doEncodeHeaders(nil, headers, endStream)
}

func (f *activeStreamDecoderFilter) EncodeData(buf types.IoBuffer, endStream bool) {
	f.activeStream.doEncodeData(nil, buf, endStream)
}

func (f *activeStreamDecoderFilter) EncodeTrailers(trailers map[string]string) {
	f.activeStream.downstreamRespTrailers = trailers
	f.activeStream.doEncodeTrailers(nil, trailers)
}

func (f *activeStreamDecoderFilter) OnDecoderFilterAboveWriteBufferHighWatermark() {
	f.activeStream.responseEncoder.GetStream().ReadDisable(true)
}

func (f *activeStreamDecoderFilter) OnDecoderFilterBelowWriteBufferLowWatermark() {
	f.activeStream.responseEncoder.GetStream().ReadDisable(false)
}

func (f *activeStreamDecoderFilter) AddDownstreamWatermarkCallbacks(cb types.DownstreamWatermarkEventListener) {
	f.activeStream.watermarkCallbacks = cb
}

func (f *activeStreamDecoderFilter) RemoveDownstreamWatermarkCallbacks(cb types.DownstreamWatermarkEventListener) {
	f.activeStream.watermarkCallbacks = nil
}

func (f *activeStreamDecoderFilter) SetDecoderBufferLimit(limit uint32) {
	f.activeStream.setBufferLimit(limit)
}

func (f *activeStreamDecoderFilter) DecoderBufferLimit() uint32 {
	return f.activeStream.bufferLimit
}

// types.StreamEncoderFilterCallbacks
type activeStreamEncoderFilter struct {
	activeStreamFilter

	filter types.StreamEncoderFilter
}

func newActiveStreamEncoderFilter(idx int, activeStream *activeStream,
	filter types.StreamEncoderFilter) *activeStreamEncoderFilter {
	f := &activeStreamEncoderFilter{
		activeStreamFilter: activeStreamFilter{
			index:        idx,
			activeStream: activeStream,
		},
		filter: filter,
	}

	filter.SetEncoderFilterCallbacks(f)

	return f
}

func (f *activeStreamEncoderFilter) ContinueEncoding() {
	f.doContinue()
}

func (f *activeStreamEncoderFilter) doContinue() {
	f.stopped = false
	hasBuffedData := f.activeStream.downstreamRespDataBuf != nil
	hasTrailer := f.activeStream.downstreamRespTrailers == nil

	if !f.headersContinued {
		f.headersContinued = true
		endStream := f.activeStream.localProcessDone && !hasBuffedData && !hasTrailer
		f.activeStream.doEncodeHeaders(f, f.activeStream.downstreamRespHeaders, endStream)
	}

	if hasBuffedData || f.stoppedNoBuf {
		if f.stoppedNoBuf || f.activeStream.downstreamRespDataBuf == nil {
			f.activeStream.downstreamRespDataBuf = buffer.NewIoBuffer(0)
		}

		endStream := f.activeStream.downstreamRecvDone && !hasTrailer
		f.activeStream.doEncodeData(f, f.activeStream.downstreamRespDataBuf, endStream)
	}

	if hasTrailer {
		f.activeStream.doEncodeTrailers(f, f.activeStream.downstreamRespTrailers)
	}
}

func (f *activeStreamEncoderFilter) handleBufferData(buf types.IoBuffer) {
	if f.activeStream.downstreamRespDataBuf != buf {
		if f.activeStream.downstreamRespDataBuf == nil {
			f.activeStream.downstreamRespDataBuf = buffer.NewIoBuffer(buf.Len())
		}

		f.activeStream.downstreamRespDataBuf.ReadFrom(buf)
	}
}

func (f *activeStreamEncoderFilter) EncodingBuffer() types.IoBuffer {
	return f.activeStream.downstreamRespDataBuf
}

func (f *activeStreamEncoderFilter) AddEncodedData(buf types.IoBuffer, streamingFilter bool) {
	f.activeStream.addEncodedData(f, buf, streamingFilter)
}

func (f *activeStreamEncoderFilter) OnEncoderFilterAboveWriteBufferHighWatermark() {
	f.activeStream.callHighWatermarkCallbacks()
}

func (f *activeStreamEncoderFilter) OnEncoderFilterBelowWriteBufferLowWatermark() {
	f.activeStream.callLowWatermarkCallbacks()
}

func (f *activeStreamEncoderFilter) SetEncoderBufferLimit(limit uint32) {
	f.activeStream.setBufferLimit(limit)
}

func (f *activeStreamEncoderFilter) EncoderBufferLimit() uint32 {
	return f.activeStream.bufferLimit
}
