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

package boltv2

import (
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/types"
)

type RequestHeader struct {
	bolt.RequestHeader
	Version1   byte //00
	SwitchCode byte
}

// Request is the cmd struct of bolt v2 request
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
	return r.RequestHeader.CmdCode == bolt.CmdCodeHeartbeat
}

func (r *Request) GetStreamType() xprotocol.StreamType {
	switch r.RequestHeader.CmdType {
	case bolt.CmdTypeRequest:
		return xprotocol.Request
	case bolt.CmdTypeRequestOneway:
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

type ResponseHeader struct {
	bolt.ResponseHeader
	Version1   byte //00
	SwitchCode byte
}

// Response is the cmd struct of bolt v2 response
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
	return r.ResponseHeader.CmdCode == bolt.CmdCodeHeartbeat
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
