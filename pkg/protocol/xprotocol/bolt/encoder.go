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

package bolt

import (
	"context"
	"encoding/binary"

	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func encodeRequest(ctx context.Context, request *Request) (types.IoBuffer, error) {
	// 1. fast-path, use existed raw data
	if request.rawData != nil {
		// 1.1 replace requestId
		binary.BigEndian.PutUint32(request.rawMeta[RequestIdIndex:], request.RequestId)

		// 1.2 check if header/content changed
		if !request.Header.Changed && !request.ContentChanged {
			// hack: increase the buffer count to avoid premature recycle
			request.Data.Count(1)
			return request.Data, nil
		}
	}

	// 2. slow-path, construct buffer from scratch

	// 2.1 calculate frame length
	if request.Class != "" {
		request.ClassLen = uint16(len(request.Class))
	}
	if len(request.Header.Kvs) != 0 {
		request.HeaderLen = uint16(xprotocol.GetHeaderEncodeLength(&request.Header))
	}
	if request.Content != nil {
		request.ContentLen = uint32(request.Content.Len())
	}
	frameLen := RequestHeaderLen + int(request.ClassLen) + int(request.HeaderLen) + int(request.ContentLen)

	// 2.2 alloc encode buffer, this buffer will be recycled after connection.Write
	buf := buffer.GetIoBuffer(frameLen)

	// 2.3 encode: meta, class, header, content
	// 2.3.1 meta
	buf.WriteByte(request.Protocol)
	buf.WriteByte(request.CmdType)
	buf.WriteUint16(request.CmdCode)
	buf.WriteByte(request.Version)
	buf.WriteUint32(request.RequestId)
	buf.WriteByte(request.Codec)
	buf.WriteUint32(uint32(request.Timeout))
	buf.WriteUint16(request.ClassLen)
	buf.WriteUint16(request.HeaderLen)
	buf.WriteUint32(request.ContentLen)
	// 2.3.2 class
	if request.ClassLen > 0 {
		buf.WriteString(request.Class)
	}
	// 2.3.3 header
	if request.HeaderLen > 0 {
		xprotocol.EncodeHeader(buf, &request.Header)
	}
	// 2.3.4 content
	if request.ContentLen > 0 {
		// use request.Content.WriteTo might have error under retry scene
		buf.Write(request.Content.Bytes())
	}

	return buf, nil
}

func encodeResponse(ctx context.Context, response *Response) (types.IoBuffer, error) {
	// 1. fast-path, use existed raw data
	if response.rawData != nil {
		// 1. replace requestId
		binary.BigEndian.PutUint32(response.rawMeta[RequestIdIndex:], uint32(response.RequestId))

		// 2. check header change
		if !response.Header.Changed && !response.ContentChanged {
			// hack: increase the buffer count to avoid premature recycle
			response.Data.Count(1)
			return response.Data, nil
		}
	}

	// 2. slow-path, construct buffer from scratch

	// 2.1 calculate frame length
	if response.Class != "" {
		response.ClassLen = uint16(len(response.Class))
	}
	if len(response.Header.Kvs) != 0 {
		response.HeaderLen = uint16(xprotocol.GetHeaderEncodeLength(&response.Header))
	}
	if response.Content != nil {
		response.ContentLen = uint32(response.Content.Len())
	}
	frameLen := ResponseHeaderLen + int(response.ClassLen) + int(response.HeaderLen) + int(response.ContentLen)

	// 2.2 alloc encode buffer, this buffer will be recycled after connection.Write
	buf := buffer.GetIoBuffer(frameLen)

	// 2.3 encode: meta, class, header, content
	// 2.3.1 meta
	buf.WriteByte(response.Protocol)
	buf.WriteByte(response.CmdType)
	buf.WriteUint16(response.CmdCode)
	buf.WriteByte(response.Version)
	buf.WriteUint32(response.RequestId)
	buf.WriteByte(response.Codec)
	buf.WriteUint16(response.ResponseStatus)
	buf.WriteUint16(response.ClassLen)
	buf.WriteUint16(response.HeaderLen)
	buf.WriteUint32(response.ContentLen)
	// 2.3.2 class
	if response.ClassLen > 0 {
		buf.WriteString(response.Class)
	}
	// 2.3.3 header
	if response.HeaderLen > 0 {
		xprotocol.EncodeHeader(buf, &response.Header)
	}
	// 2.3.4 content
	if response.ContentLen > 0 {
		// use request.Content.WriteTo might have error under retry scene
		buf.Write(response.Content.Bytes())
	}

	return buf, nil
}
