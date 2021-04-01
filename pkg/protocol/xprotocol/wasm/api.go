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

func NewWasmRequestWithId(id uint64, headers *xprotocol.Header, payload types.IoBuffer) *Request {
	request := &Request{
		RequestHeader: RequestHeader{
			RpcHeader: RpcHeader{
				Flag: RpcRequestFlag,
				Id:   id,
			},
			Timeout: 3 * 1000, // 3 seconds
		},
	}

	if headers != nil {
		headers.Range(func(key, value string) bool {
			request.Set(key, value)
			return true
		})
		// request is initialized. The default header has not changed.
		request.Changed = false
	}

	if payload != nil {
		request.payload = payload.Bytes()
		request.PayLoad = payload
	}
	return request
}

func NewWasmResponseWithId(requestId uint64, headers *xprotocol.Header, payload types.IoBuffer) *Response {
	response := &Response{
		ResponseHeader: ResponseHeader{
			RpcHeader: RpcHeader{
				Flag: RpcResponseFlag,
				Id:   requestId,
			},
		},
	}

	if headers != nil {
		headers.Range(func(key, value string) bool {
			response.Set(key, value)
			return true
		})
		// request is initialized. The default header has not changed.
		response.Changed = false
	}

	if payload != nil {
		response.payload = payload.Bytes()
		response.PayLoad = payload
	}
	return response
}
