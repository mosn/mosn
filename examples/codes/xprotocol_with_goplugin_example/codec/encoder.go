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

package codec

import (
	"context"
	"fmt"

	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

// 传入request对象，根据他的payload内容，获取PayloadLen并且为拼接符合协议的io流返回
func encodeRequest(ctx context.Context, request *Request) (api.IoBuffer, error) {
	fmt.Printf("[in encodeRequest] request: %+v\n", request.Payload)

	// 1. TODO: fast-path, use existed raw data

	// 2.1 calculate frame length
	if request.Payload != nil {
		request.PayloadLen = uint32(len(request.Payload.Bytes()))
	}
	frameLen := RequestHeaderLen + int(request.PayloadLen)

	// 2.2 alloc encode buffer
	buf := buffer.GetIoBuffer(frameLen)

	// 2.3 encode: meta, payload
	buf.WriteByte(Magic)
	buf.WriteByte(request.Type)
	buf.WriteByte(DirRequest)
	buf.WriteUint32(request.RequestId)
	buf.WriteUint32(request.PayloadLen)

	if request.PayloadLen > 0 {
		buf.Write(request.Payload.Bytes())
	}

	fmt.Printf("[out encodeRequest]\n")

	return buf, nil
}

// response，根据他的payload内容，获取PayloadLen并且为拼接符合协议的io流返回
func encodeResponse(ctx context.Context, response *Response) (api.IoBuffer, error) {
	// 1. TODO: fast-path, use existed raw data
	fmt.Printf("[in encodeResponse] response: %+v\n", response.Payload)

	// 2.1 calculate frame length
	if response.Payload != nil {
		response.PayloadLen = uint32(len(response.Payload.Bytes()))
	}
	frameLen := ResponseHeaderLen + int(response.PayloadLen)

	// 2.2 alloc encode buffer
	buf := buffer.GetIoBuffer(frameLen)

	// 2.3 encode: meta, payload
	buf.WriteByte(Magic)
	buf.WriteByte(response.Type)
	buf.WriteByte(DirResponse)
	buf.WriteUint32(response.RequestId)
	buf.WriteUint16(response.Status)
	buf.WriteUint32(response.PayloadLen)

	if response.PayloadLen > 0 {
		buf.Write(response.Payload.Bytes())
	}

	fmt.Printf("[out encodeResponse]\n")

	return buf, nil
}
