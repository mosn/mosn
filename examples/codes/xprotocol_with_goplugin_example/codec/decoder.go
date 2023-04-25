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
	"encoding/binary"
	"errors"
	"fmt"

	"mosn.io/api"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/header"
)

// 根据传入的io流信息（已经符合协议） 封装好request对象并且返回
func decodeRequest(ctx context.Context, data api.IoBuffer) (cmd interface{}, err error) {
	fmt.Printf("[in decodeRequest]\n")

	bytesLen := data.Len()
	bytes := data.Bytes()

	// 1. least bytes to decode header is RequestHeaderLen
	if bytesLen < RequestHeaderLen {
		return nil, errors.New("error request")
	}

	// 2. least bytes to decode whole frame
	payloadLen := binary.BigEndian.Uint32(bytes[RequestPayloadIndex:RequestHeaderLen])
	frameLen := RequestHeaderLen + int(payloadLen)
	if bytesLen < frameLen {
		return nil, errors.New("error request")
	}
	data.Drain(frameLen)

	// 3. decode header
	request := &Request{
		Type:         bytes[TypeIndex],
		RequestId:    binary.BigEndian.Uint32(bytes[RequestIdIndex:RequestPayloadIndex]),
		PayloadLen:   payloadLen,
		CommonHeader: header.CommonHeader{},
	}

	//4. copy data for io multiplexing
	request.Payload = buffer.NewIoBufferBytes(bytes[RequestHeaderLen:])

	fmt.Printf("[out decodeRequest] payload: %s\n", request.Payload)

	return request, nil
}

// 根据传入的io流信息（已经符合协议） 封装好response对象并且返回
func decodeResponse(ctx context.Context, data api.IoBuffer) (cmd interface{}, err error) {
	fmt.Printf("[in decodeResponse]\n")

	bytesLen := data.Len()
	bytes := data.Bytes()

	// 1. least bytes to decode header is ResponseHeaderLen
	if bytesLen < ResponseHeaderLen {
		return nil, errors.New("error response")
	}

	// 2. least bytes to decode whole frame
	payloadLen := binary.BigEndian.Uint32(bytes[ResponsePayloadIndex:ResponseHeaderLen])
	frameLen := ResponseHeaderLen + int(payloadLen)
	if bytesLen < frameLen {
		return nil, errors.New("error response")
	}
	data.Drain(frameLen)

	// 3. decode header
	response := &Response{
		Request: Request{
			Type:         bytes[TypeIndex],
			RequestId:    binary.BigEndian.Uint32(bytes[RequestIdIndex:RequestPayloadIndex]),
			PayloadLen:   payloadLen,
			CommonHeader: header.CommonHeader{},
		},
		Status: ResponseStatusSuccess,
	}

	//4. copy data for io multiplexing
	response.Payload = buffer.NewIoBufferBytes(bytes[ResponseHeaderLen:])

	fmt.Printf("[out decodeResponse] payload: %s\n", response.Payload)

	return response, nil
}
