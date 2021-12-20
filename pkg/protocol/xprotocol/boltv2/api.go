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
	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
)

// NewRpcRequest is a utility function which build rpc Request object of bolt protocol.
func NewRpcRequest(requestId uint32, headers api.HeaderMap, data api.IoBuffer) *Request {
	request := &Request{
		RequestHeader: RequestHeader{
			Version1: ProtocolVersion1,
			RequestHeader: bolt.RequestHeader{
				Protocol:  ProtocolCode,
				CmdType:   bolt.CmdTypeRequest,
				CmdCode:   bolt.CmdCodeRpcRequest,
				Version:   ProtocolVersion,
				RequestId: requestId,
				Codec:     bolt.Hessian2Serialize,
				Timeout:   -1,
			},
		},
	}

	// set headers
	if headers != nil {
		headers.Range(func(key, value string) bool {
			request.Set(key, value)
			return true
		})
	}

	// set content
	if data != nil {
		request.Content = data
	}
	return request
}

// NewRpcResponse is a utility function which build rpc Response object of bolt protocol.
func NewRpcResponse(requestId uint32, statusCode uint16, headers api.HeaderMap, data api.IoBuffer) *Response {
	response := &Response{
		ResponseHeader: ResponseHeader{
			Version1: ProtocolVersion1,
			ResponseHeader: bolt.ResponseHeader{
				Protocol:       ProtocolCode,
				CmdType:        bolt.CmdTypeResponse,
				CmdCode:        bolt.CmdCodeRpcResponse,
				Version:        ProtocolVersion,
				RequestId:      requestId,
				Codec:          bolt.Hessian2Serialize,
				ResponseStatus: statusCode,
			},
		},
	}

	// set headers
	if headers != nil {
		headers.Range(func(key, value string) bool {
			response.Set(key, value)
			return true
		})
	}

	// set content
	if data != nil {
		response.Content = data
	}
	return response
}
