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
	"context"
	"encoding/binary"

	mbuffer "mosn.io/mosn/pkg/buffer"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func encodeRequest(ctx context.Context, request *Request) (types.IoBuffer, error) {
	// 1. fast-path, use existed raw data
	if request.rawData != nil {
		// 1. replace requestId
		binary.BigEndian.PutUint32(request.rawMeta[RequestIdIndex:], request.RequestId)

		// 2. TODO: header mutate

		return request.Data, nil
	}

	// 2. slow-path, construct buffer from scratch

	// 2.1 calculate frame length
	if request.Class != "" {
		request.ClassLen = uint16(len(request.Class))
	}
	if len(request.Header.Kvs) != 0 {
		request.HeaderLen = uint16(getHeaderEncodeLength(&request.Header))
	}
	if request.Content != nil {
		request.ContentLen = uint32(request.Content.Len())
	}
	frameLen := RequestHeaderLen + int(request.ClassLen) + int(request.HeaderLen) + int(request.ContentLen)

	// 2.2 alloc encode buffer
	buf := *mbuffer.GetBytesByContext(ctx, frameLen)

	// 2.3 encode: meta, class, header, content
	buf[0] = request.Protocol
	buf[1] = request.CmdType
	binary.BigEndian.PutUint16(buf[2:], request.CmdCode)
	buf[4] = request.Version
	binary.BigEndian.PutUint32(buf[5:], request.RequestId)
	buf[9] = request.Codec
	binary.BigEndian.PutUint32(buf[10:], uint32(request.Timeout))
	binary.BigEndian.PutUint16(buf[14:], request.ClassLen)
	binary.BigEndian.PutUint16(buf[16:], request.HeaderLen)
	binary.BigEndian.PutUint32(buf[18:], request.ContentLen)

	headerIndex := RequestHeaderLen + int(request.ClassLen)
	contentIndex := headerIndex + int(request.HeaderLen)

	if request.ClassLen > 0 {
		copy(buf[RequestHeaderLen:], request.Class)
	}

	if request.HeaderLen > 0 {
		encodeHeader(buf[headerIndex:], &request.Header)
	}

	if request.ContentLen > 0 {
		copy(buf[contentIndex:], request.Content.Bytes())
	}

	return buffer.NewIoBufferBytes(buf), nil
}

func encodeResponse(ctx context.Context, response *Response) (types.IoBuffer, error) {
	// 1. fast-path, use existed raw data
	if response.rawData != nil {
		// 1. replace requestId
		binary.BigEndian.PutUint32(response.rawMeta[RequestIdIndex:], uint32(response.RequestId))

		// 2. TODO: header mutate

		return response.Data, nil
	}

	// 2. slow-path, construct buffer from scratch

	// 2.1 calculate frame length
	if response.Class != "" {
		response.ClassLen = uint16(len(response.Class))
	}
	if len(response.Header.Kvs) != 0 {
		response.HeaderLen = uint16(getHeaderEncodeLength(&response.Header))
	}
	if response.Content != nil {
		response.ContentLen = uint32(response.Content.Len())
	}
	frameLen := ResponseHeaderLen + int(response.ClassLen) + int(response.HeaderLen) + int(response.ContentLen)

	// 2.2 alloc encode buffer
	buf := *mbuffer.GetBytesByContext(ctx, frameLen)

	// 2.3 encode: meta, class, header, content
	buf[0] = response.Protocol
	buf[1] = response.CmdType
	binary.BigEndian.PutUint16(buf[2:], response.CmdCode)
	buf[4] = response.Version
	binary.BigEndian.PutUint32(buf[5:], response.RequestId)
	buf[9] = response.Codec
	binary.BigEndian.PutUint16(buf[10:], uint16(response.ResponseStatus))
	binary.BigEndian.PutUint16(buf[12:], response.ClassLen)
	binary.BigEndian.PutUint16(buf[14:], response.HeaderLen)
	binary.BigEndian.PutUint32(buf[16:], response.ContentLen)

	headerIndex := ResponseHeaderLen + int(response.ClassLen)
	contentIndex := headerIndex + int(response.HeaderLen)

	if response.ClassLen > 0 {
		copy(buf[ResponseHeaderLen:], response.Class)
	}

	if response.HeaderLen > 0 {
		encodeHeader(buf[headerIndex:], &response.Header)
	}

	if response.ContentLen > 0 {
		copy(buf[contentIndex:], response.Content.Bytes())
	}

	return buffer.NewIoBufferBytes(buf), nil
}

func getHeaderEncodeLength(h *xprotocol.Header) (size int) {
	for i, n := 0, len(h.Kvs); i < n; i++ {
		size += 8 + len(h.Kvs[i].Key) + len(h.Kvs[i].Value)
	}
	return
}

func encodeHeader(buf []byte, h *xprotocol.Header) {
	index := 0

	for _, kv := range h.Kvs {
		index = encodeStr(buf, index, kv.Key)
		index = encodeStr(buf, index, kv.Value)
	}
}

func encodeStr(buf []byte, index int, str []byte) (newIndex int) {
	length := len(str)

	// 1. encode str length
	binary.BigEndian.PutUint32(buf[index:], uint32(length))

	// 2. encode str value
	copy(buf[index+4:], str)

	return index + 4 + length
}
