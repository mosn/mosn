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

package wasm

import (
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

type RpcHeader struct {
	Flag       byte   // rpc request or response flag
	Id         uint32 // request or response id
	ReplacedId uint32 // request or response id (replaced by stream)
	HeaderLen  uint32 // header length
	PayloadLen uint32 // payload length

	xprotocol.Header
}

type RequestHeader struct {
	RpcHeader        // common rpc header
	Timeout   uint32 // request timeout
}

func (h *RequestHeader) Clone() types.HeaderMap {
	clone := &RequestHeader{}
	*clone = *h

	// deep copy
	clone.Header = *h.Header.Clone()

	return clone
}

type Request struct {
	RequestHeader
	data           []byte         // raw request bytes
	Data           types.IoBuffer // wrapper of data
	payload        []byte         // sub slice of data, rpc payload bytes
	PayLoad        types.IoBuffer // wrapper of payload
	PayloadChanged bool           // check if payload if modified
}

// ~ XFrame
func (r *Request) GetRequestId() uint64 {
	return uint64(r.Id)
}

func (r *Request) GetId() uint32 {
	return r.Id
}

func (r *Request) SetRequestId(id uint64) {
	// save replace stream id
	r.ReplacedId = r.Id
	// new request id
	r.Id = uint32(id)
}

func (r *Request) IsHeartbeatFrame() bool {
	return r.Flag&HeartBeatFlag != 0
}

func (r *Request) GetStreamType() xprotocol.StreamType {
	reqType := r.Flag >> 6 // high 2 bit
	switch reqType {
	case RequestType:
		return xprotocol.Request
	case RequestOneWayType:
		return xprotocol.RequestOneWay
	default:
		return xprotocol.Request
	}
}

func (r *Request) GetHeader() types.HeaderMap {
	return r
}

func (r *Request) GetData() types.IoBuffer {
	return r.PayLoad
}

func (r *Request) SetData(data types.IoBuffer) {
	if r.PayLoad != data {
		r.PayLoad = data
		r.PayloadChanged = true
	}
}

type ResponseHeader struct {
	RpcHeader        // common rpc header
	Status    uint32 // response status
}

func (h *ResponseHeader) Clone() types.HeaderMap {
	clone := &ResponseHeader{}
	*clone = *h

	// deep copy
	clone.Header = *h.Header.Clone()

	return clone
}

type Response struct {
	ResponseHeader
	data           []byte         // raw response bytes
	Data           types.IoBuffer // wrapper of data
	payload        []byte         // sub slice of data, rpc payload bytes
	PayLoad        types.IoBuffer // wrapper of payload
	PayloadChanged bool           // check if payload if modified
}

func (r *Response) GetRequestId() uint64 {
	return uint64(r.Id)
}

func (r *Response) SetRequestId(id uint64) {
	// save replace stream id
	r.ReplacedId = r.Id
	r.Id = uint32(id)
}

func (r *Response) IsHeartbeatFrame() bool {
	return r.Flag&HeartBeatFlag != 0
}

func (r *Response) GetStreamType() xprotocol.StreamType {
	return xprotocol.Response
}

func (r *Response) GetHeader() types.HeaderMap {
	return r
}

func (r *Response) GetData() types.IoBuffer {
	return r.PayLoad
}

func (r *Response) SetData(data types.IoBuffer) {
	if r.PayLoad != data {
		r.PayLoad = data
		r.PayloadChanged = true
	}
}

func (r *Response) GetStatusCode() uint32 {
	return r.Status
}

func (r *Response) getId() uint32 {
	return uint32(r.Id)
}
