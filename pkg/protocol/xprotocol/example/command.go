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

package example

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/header"
)

type Request struct {
	Type       byte
	RequestId  uint32
	PayloadLen uint32
	Payload    []byte
	Content    types.IoBuffer
	header.CommonHeader
}

func (r *Request) IsHeartbeatFrame() bool {
	return r.Type == TypeHeartbeat
}

func (r *Request) IsGoAwayFrame() bool {
	return r.Type == TypeGoAway
}

func (r *Request) GetTimeout() int32 {
	return -1
}

func (r *Request) GetHeader() api.HeaderMap {
	return r
}

func (r *Request) GetData() api.IoBuffer {
	return r.Content
}

func (r *Request) SetData(data api.IoBuffer) {
	if r.Content != data {
		r.Content = data
	}
}

func (r *Request) GetStreamType() api.StreamType {
	return api.Request
}

func (r *Request) GetRequestId() uint64 {
	return uint64(r.RequestId)
}

func (r *Request) SetRequestId(id uint64) {
	r.RequestId = uint32(id)
}

type Response struct {
	Request
	Status uint16
}

var _ api.XRespFrame = &Response{}

func (r *Response) GetStatusCode() uint32 {
	return uint32(r.Status)
}

func (r *Response) GetStreamType() api.StreamType {
	return api.Response
}

func (r *Response) GetRequestId() uint64 {
	return uint64(r.RequestId)
}

func (r *Response) SetRequestId(id uint64) {
	r.RequestId = uint32(id)
}

func (r *Response) GetHeader() api.HeaderMap {
	return r
}

type MessageCommand struct {
	Request

	// TODO: pb deserialize target, extract from Payload
	Message interface{}
}

type MessageAckCommand struct {
	Response

	// TODO: pb deserialize target, extract from Payload
	Message interface{}
}
