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

package stream

import (
	"context"
	"net/http"

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

type ActiveStream struct {
	// current receive phase
	currentReceivePhase api.ReceiverFilterPhase

	code int
	ctx  context.Context

	cid uint64
	sid uint64

	// request
	reqHeaders  *Headers
	reqTrailers *Headers
	reqDataBuf  types.IoBuffer

	// response
	respHeaders  *Headers
	respTrailers *Headers
	respDataBuf  types.IoBuffer

	directResponse bool

	api.StreamReceiverFilterHandler
	api.StreamSenderFilterHandler

	fm *streamfilter.DefaultStreamFilterChainImpl
}

// CreateActiveStream create ActiveStream.
func CreateActiveStream(ctx context.Context) ActiveStream {
	return ActiveStream{
		ctx:            ctx,
		directResponse: false,
		reqHeaders:     &Headers{},
		reqTrailers:    &Headers{},
		respHeaders:    &Headers{},
		respTrailers:   &Headers{},
	}
}

// InitResuestStream init request stream.
func (s *ActiveStream) InitResuestStream(headers *Headers, data types.IoBuffer, trailers *Headers) {
	s.reqHeaders = headers
	s.reqDataBuf = data
	s.reqTrailers = trailers
}

// InitResponseStream init response stream.
func (s *ActiveStream) InitResponseStream(headers *Headers, data types.IoBuffer, trailers *Headers) {
	s.respHeaders = headers
	s.respDataBuf = data
	s.respTrailers = trailers
}

// SetCurrentReveivePhase set current reveive phase.
func (s *ActiveStream) SetCurrentReveivePhase(phase api.ReceiverFilterPhase) {
	s.currentReceivePhase = phase
}

// IsDirectResponse check whether it is direct response.
func (s *ActiveStream) IsDirectResponse() bool {
	return s.directResponse
}

// GetResponseCode get response status code.
func (s *ActiveStream) GetResponseCode() int {
	return s.code
}

// SetConnectionID is set connection id.
func (s *ActiveStream) SetConnectionID(cid uint64) {
	s.cid = cid
}

// SetStreamID is set stream id.
func (s *ActiveStream) SetStreamID(sid uint64) {
	s.sid = sid
}

// TODO: implement api.StreamFilterHandler after add context and stream id for stream
// implement api.StreamFilterHandler

// fakeConnection is used to implement Connection interface.
type fakeConnection struct {
	api.Connection // TODO implement more api
	id             uint64
}

func (c *fakeConnection) ID() uint64 {
	return c.id
}

func (f *ActiveStream) Connection() api.Connection {
	return &fakeConnection{id: f.cid}
}

func (f *ActiveStream) Route() types.Route {
	var route types.Route
	panic("not implemented")
	return route
}

func (f *ActiveStream) RequestInfo() types.RequestInfo {
	var requestInfo types.RequestInfo
	panic("not implemented")
	return requestInfo
}

func (s *ActiveStream) GetRequestUpdatedHeaders() types.HeaderMap {
	return protocol.CommonHeader(s.reqHeaders.Update)
}

func (s *ActiveStream) GetResponseUpdatedHeaders() types.HeaderMap {
	return protocol.CommonHeader(s.respHeaders.Update)
}

// implement api.StreamReceiverFilterHandler
func (s *ActiveStream) GetRequestHeaders() types.HeaderMap {
	return s.reqHeaders
}

func (s *ActiveStream) GetRequestData() types.IoBuffer {
	return s.reqDataBuf
}

func (s *ActiveStream) GetRequestTrailers() types.HeaderMap {
	return s.reqTrailers
}

// receiver filter handler maybe delete AppendXXX API
func (s *ActiveStream) AppendHeaders(headers types.HeaderMap, endStream bool) {
	panic("not implemented")
}

func (s *ActiveStream) AppendData(buf types.IoBuffer, endStream bool) {
	panic("not implemented")
}

func (s *ActiveStream) AppendTrailers(trailers types.HeaderMap) {
	panic("not implemented")
}

func (s *ActiveStream) SendHijackReply(code int, headers types.HeaderMap) {
	s.directResponse = true
	s.code = code
	h := make(map[string]string)
	if headers != nil {
		headers.Range(func(k, v string) bool {
			h[k] = v
			return true
		})
	}
	s.respHeaders = &Headers{
		CommonHeader: protocol.CommonHeader(h),
	}
}

func (s *ActiveStream) SendHijackReplyWithBody(code int, headers types.HeaderMap, body string) {
	s.directResponse = true
	s.code = code
	h := make(map[string]string)
	if headers != nil {
		headers.Range(func(k, v string) bool {
			h[k] = v
			return true
		})
	}
	s.respHeaders = &Headers{
		CommonHeader: protocol.CommonHeader(h),
	}
	s.respDataBuf = buffer.NewIoBufferString(body)
}

func (s *ActiveStream) SendDirectResponse(headers types.HeaderMap, buf buffer.IoBuffer, trailers types.HeaderMap) {
	s.directResponse = true
	// TODO status suppport auto mapping
	s.code = http.StatusOK
	h := make(map[string]string)
	if headers != nil {
		headers.Range(func(k, v string) bool {
			h[k] = v
			return true
		})
	}
	s.respHeaders = &Headers{
		CommonHeader: protocol.CommonHeader(h),
	}

	s.respDataBuf = buf
	t := make(map[string]string)
	if trailers != nil {
		trailers.Range(func(k, v string) bool {
			t[k] = v
			return true
		})
	}

	s.respTrailers = &Headers{
		CommonHeader: protocol.CommonHeader(t),
	}
}

func (s *ActiveStream) TerminateStream(code int) bool {
	// check if it has been responded.
	if s.GetResponseHeaders() != nil {
		return false
	}

	headers := s.GetRequestHeaders()
	if headers == nil {
		return false
	}

	s.SendHijackReply(code, headers)
	return true
}

func (s *ActiveStream) SetConvert(on bool) {
	panic("not implemented")
}

func (s *ActiveStream) GetFilterCurrentPhase() api.ReceiverFilterPhase {
	return s.currentReceivePhase
}

// implement api.StreamSenderFilterHandler
func (s *ActiveStream) GetResponseHeaders() types.HeaderMap {
	return s.respHeaders
}

func (s *ActiveStream) GetResponseData() types.IoBuffer {
	return s.respDataBuf
}

func (s *ActiveStream) GetResponseTrailers() types.HeaderMap {
	return s.respTrailers
}
