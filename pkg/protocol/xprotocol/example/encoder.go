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
	"encoding/binary"

	mbuffer "mosn.io/mosn/pkg/buffer"
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
	buf := *mbuffer.GetBytesByContext(ctx, frameLen)

	// 2.3 encode: meta, payload
	buf[0] = Magic
	buf[1] = request.Type
	buf[2] = DirRequest
	binary.BigEndian.PutUint32(buf[3:], request.RequestId)
	binary.BigEndian.PutUint32(buf[7:], request.PayloadLen)

	if request.PayloadLen > 0 {
		copy(buf[RequestHeaderLen:], request.Payload)
	}

	return buffer.NewIoBufferBytes(buf), nil
}

func encodeResponse(ctx context.Context, response *Response) (types.IoBuffer, error) {
	// 1. TODO: fast-path, use existed raw data

	// 2.1 calculate frame length
	if response.Payload != nil {
		response.PayloadLen = uint32(len(response.Payload))
	}
	frameLen := ResponseHeaderLen + int(response.PayloadLen)

	// 2.2 alloc encode buffer
	buf := *mbuffer.GetBytesByContext(ctx, frameLen)

	// 2.3 encode: meta, payload
	buf[0] = Magic
	buf[1] = response.Type
	buf[2] = DirResponse
	binary.BigEndian.PutUint32(buf[3:], response.RequestId)
	binary.BigEndian.PutUint16(buf[7:], response.Status)
	binary.BigEndian.PutUint32(buf[9:], response.PayloadLen)

	if response.PayloadLen > 0 {
		copy(buf[ResponseHeaderLen:], response.Payload)
	}

	return buffer.NewIoBufferBytes(buf), nil
}
