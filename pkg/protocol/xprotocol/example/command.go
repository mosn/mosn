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
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

type Request struct {
	Type       byte
	RequestId  uint32
	PayloadLen uint32
	Payload    []byte
	Content    types.IoBuffer
}

func (r *Request) GetStreamType() xprotocol.StreamType {
	return xprotocol.Request
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

func (r *Response) GetStreamType() xprotocol.StreamType {
	return xprotocol.Response
}

func (r *Response) GetRequestId() uint64 {
	return uint64(r.RequestId)
}

func (r *Response) SetRequestId(id uint64) {
	r.RequestId = uint32(id)
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
