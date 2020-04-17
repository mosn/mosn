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
	"context"

	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func encodeRequest(ctx context.Context, request *Request) (types.IoBuffer, error) {
	// 1. TODO: fast-path, use existed raw data

	// 2.1 calculate frame length
	if request.Payload != nil {
		request.PayloadLen = uint32(len(request.Payload))
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
		buf.Write(request.Payload)
	}

	return buf, nil
}

func encodeResponse(ctx context.Context, response *Response) (types.IoBuffer, error) {
	// 1. TODO: fast-path, use existed raw data

	// 2.1 calculate frame length
	if response.Payload != nil {
		response.PayloadLen = uint32(len(response.Payload))
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
		buf.Write(response.Payload)
	}

	return buf, nil
}
