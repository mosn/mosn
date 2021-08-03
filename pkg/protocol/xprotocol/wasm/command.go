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
	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

type RpcHeader struct {
	Flag         byte   // rpc request or response flag
	Id           uint64 // request or response id
	RpcId        uint64 // request or response id (replaced by stream)
	IsReplacedId bool   // check if the id has been replaced
	HeaderLen    uint32 // header length

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
	ctx            *Context       // current context
}

func (r *Request) GetTimeout() int32 {
	return int32(r.Timeout)
}

func (r *Request) GetRequestId() uint64 {
	return uint64(r.Id)
}

func (r *Request) GetId() uint64 {
	return r.Id
}

func (r *Request) SetRequestId(id uint64) {
	// save replace stream id
	r.RpcId = r.Id
	// new request id
	r.Id = id
	r.IsReplacedId = true
}

func (r *Request) IsHeartbeatFrame() bool {
	return r.Flag&HeartBeatFlag != 0
}

func (r *Request) GetStreamType() api.StreamType {
	reqType := r.Flag >> 6 // high 2 bit
	switch reqType {
	case RequestType:
		return api.Request
	case RequestOneWayType:
		return api.RequestOneWay
	default:
		return api.Request
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

func (r *Request) clean() {
	ctx := r.ctx
	if ctx != nil {
		ctx.instance.Lock(ctx.abi)
		ctx.abi.SetABIImports(ctx)
		ctx.exports.ProxyOnDelete(ctx.contextId)
		ctx.instance.Unlock()
		r.ctx = nil
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
	ctx            *Context       // current context
}

func (r *Response) GetRequestId() uint64 {
	return uint64(r.Id)
}

func (r *Response) SetRequestId(id uint64) {
	// save replace stream id
	r.RpcId = r.Id
	r.Id = id
	r.IsReplacedId = true
}

func (r *Response) IsHeartbeatFrame() bool {
	return r.Flag&HeartBeatFlag != 0
}

func (r *Response) GetStreamType() api.StreamType {
	return api.Response
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

func (r *Response) GetId() uint64 {
	return r.Id
}

func (r *Response) GetTimeout() int32 {
	return 0
}

func (r *Response) clean() {
	ctx := r.ctx
	if ctx != nil {
		ctx.instance.Lock(ctx.abi)
		ctx.abi.SetABIImports(ctx)
		ctx.exports.ProxyOnDelete(ctx.contextId)
		ctx.instance.Unlock()
		r.ctx = nil
	}
}
