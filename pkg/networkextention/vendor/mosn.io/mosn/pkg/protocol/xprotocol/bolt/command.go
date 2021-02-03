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

package bolt

import (
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

// RequestHeader is the header part of bolt v1 request
type RequestHeader struct {
	Protocol   byte // meta fields
	CmdType    byte
	CmdCode    uint16
	Version    byte
	RequestId  uint32
	Codec      byte
	Timeout    int32
	ClassLen   uint16
	HeaderLen  uint16
	ContentLen uint32

	Class string // payload fields
	xprotocol.Header
}

// ~ HeaderMap
func (h *RequestHeader) Clone() types.HeaderMap {
	clone := &RequestHeader{}
	*clone = *h

	// deep copy
	clone.Header = *h.Header.Clone()

	return clone
}

// Request is the cmd struct of bolt v1 request
type Request struct {
	RequestHeader

	rawData    []byte // raw data
	rawMeta    []byte // sub slice of raw data, start from protocol code, ends to content length
	rawClass   []byte // sub slice of raw data, class bytes
	rawHeader  []byte // sub slice of raw data, header bytes
	rawContent []byte // sub slice of raw data, content bytes

	Data    types.IoBuffer // wrapper of raw data
	Content types.IoBuffer // wrapper of raw content

	ContentChanged bool // indicate that content changed
}

// ~ XFrame
func (r *Request) GetRequestId() uint64 {
	return uint64(r.RequestHeader.RequestId)
}

func (r *Request) SetRequestId(id uint64) {
	r.RequestHeader.RequestId = uint32(id)
}

func (r *Request) IsHeartbeatFrame() bool {
	return r.RequestHeader.CmdCode == CmdCodeHeartbeat
}

func (r *Request) GetStreamType() xprotocol.StreamType {
	switch r.RequestHeader.CmdType {
	case CmdTypeRequest:
		return xprotocol.Request
	case CmdTypeRequestOneway:
		return xprotocol.RequestOneWay
	default:
		return xprotocol.Request
	}
}

func (r *Request) GetHeader() types.HeaderMap {
	return r
}

func (r *Request) GetData() types.IoBuffer {
	return r.Content
}

func (r *Request) SetData(data types.IoBuffer) {
	// judge if the address unchanged, assume that proxy logic will not operate the original Content buffer.
	if r.Content != data {
		r.ContentChanged = true
		r.Content = data
	}
}

// RequestHeader is the header part of bolt v1 response
type ResponseHeader struct {
	Protocol       byte // meta fields
	CmdType        byte
	CmdCode        uint16
	Version        byte
	RequestId      uint32
	Codec          byte
	ResponseStatus uint16
	ClassLen       uint16
	HeaderLen      uint16
	ContentLen     uint32

	Class string // payload fields
	xprotocol.Header
}

// ~ HeaderMap
func (h *ResponseHeader) Clone() types.HeaderMap {
	clone := &ResponseHeader{}
	*clone = *h

	// deep copy
	clone.Header = *h.Header.Clone()

	return clone
}

// Response is the cmd struct of bolt v1 response
type Response struct {
	ResponseHeader

	rawData    []byte // raw data
	rawMeta    []byte // sub slice of raw data, start from protocol code, ends to content length
	rawClass   []byte // sub slice of raw data, class bytes
	rawHeader  []byte // sub slice of raw data, header bytes
	rawContent []byte // sub slice of raw data, content bytes

	Data    types.IoBuffer // wrapper of raw data
	Content types.IoBuffer // wrapper of raw content

	ContentChanged bool // indicate that content changed
}

// ~ XRespFrame
func (r *Response) GetRequestId() uint64 {
	return uint64(r.ResponseHeader.RequestId)
}

func (r *Response) SetRequestId(id uint64) {
	r.ResponseHeader.RequestId = uint32(id)
}

func (r *Response) IsHeartbeatFrame() bool {
	return r.ResponseHeader.CmdCode == CmdCodeHeartbeat
}

func (r *Response) GetStreamType() xprotocol.StreamType {
	return xprotocol.Response
}

func (r *Response) GetHeader() types.HeaderMap {
	return r
}

func (r *Response) GetData() types.IoBuffer {
	return r.Content
}

func (r *Response) SetData(data types.IoBuffer) {
	// judge if the address unchanged, assume that proxy logic will not operate the original Content buffer.
	if r.Content != data {
		r.ContentChanged = true
		r.Content = data
	}
}

func (r *Response) GetStatusCode() uint32 {
	return uint32(r.ResponseStatus)
}
