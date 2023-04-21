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

	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/header"
)

func decodeRequest(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	bytesLen := data.Len()
	bytes := data.Bytes()

	// 1. least bytes to decode header is RequestHeaderLen
	if bytesLen < RequestHeaderLen {
		return
	}

	// 2. least bytes to decode whole frame
	payloadLen := binary.BigEndian.Uint32(bytes[7:])
	frameLen := RequestHeaderLen + int(payloadLen)
	if bytesLen < frameLen {
		return
	}
	data.Drain(frameLen)

	// 3. decode header
	request := &Request{
		Type:         bytes[1],
		RequestId:    binary.BigEndian.Uint32(bytes[RequestIdIndex:]),
		PayloadLen:   payloadLen,
		CommonHeader: header.CommonHeader{},
	}

	//4. copy data for io multiplexing
	request.Payload = make([]byte, int(payloadLen))
	copy(request.Payload, bytes[RequestHeaderLen:])

	return request, err
}

func decodeResponse(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	bytesLen := data.Len()
	bytes := data.Bytes()

	// 1. least bytes to decode header is ResponseHeaderLen
	if bytesLen < ResponseHeaderLen {
		return
	}

	// 2. least bytes to decode whole frame
	payloadLen := binary.BigEndian.Uint32(bytes[9:])
	frameLen := ResponseHeaderLen + int(payloadLen)
	if bytesLen < frameLen {
		return
	}
	data.Drain(frameLen)

	// 3. decode header
	response := &Response{
		Request: Request{
			Type:         bytes[1],
			RequestId:    binary.BigEndian.Uint32(bytes[RequestIdIndex:]),
			PayloadLen:   payloadLen,
			CommonHeader: header.CommonHeader{},
		},
		Status: binary.BigEndian.Uint16(bytes[7:]),
	}

	//4. copy data for io multiplexing
	response.Payload = make([]byte, int(payloadLen))
	copy(response.Payload, bytes[ResponseHeaderLen:])

	return response, nil
}
