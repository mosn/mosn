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
	"errors"
	"fmt"
	"sync/atomic"

	"mosn.io/api"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

/**
 * Request command
 * 0     1     2           4           6           8          10           12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |magic| type| dir |      requestId        |     payloadLength     |     payload bytes ...       |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 *
 * Response command
 * 0     1     2     3     4           6           8          10           12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |magic| type| dir |      requestId        |   status  |      payloadLength    | payload bytes ..|
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 */

type proto struct{}

// types.Protocol
func (proto proto) Name() types.ProtocolName {
	return ProtocolName
}

func (proto proto) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	switch frame := model.(type) {
	case *Request:
		return encodeRequest(ctx, frame)
	case *Response:
		return encodeResponse(ctx, frame)
	default:
		log.Proxy.Errorf(ctx, "[protocol][mychain] encode with unknown command : %+v", model)
		return nil, errors.New("unknown command type")
	}
}

func (proto proto) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	if data.Len() >= MinimalDecodeLen {
		magic := data.Bytes()[0]
		dir := data.Bytes()[2]

		// 1. magic assert
		if magic != Magic {
			return nil, fmt.Errorf("[protocol][mychain] decode failed, magic error = %d", magic)
		}

		// 2. decode
		switch dir {
		case DirRequest:
			return decodeRequest(ctx, data)
		case DirResponse:
			return decodeResponse(ctx, data)
		default:
			// unknown cmd type
			return nil, fmt.Errorf("[protocol][mychain] decode failed, direction error = %d", dir)
		}
	}

	return nil, nil
}

func NewCodec() api.Protocol {
	return &proto{}
}

// TODOs

// Heartbeater
func (proto proto) Trigger(ctx context.Context, requestId uint64) api.XFrame {
	// not supported for poc demo
	return nil
}

func (proto proto) Reply(ctx context.Context, request api.XFrame) api.XRespFrame {
	// not supported for poc demo
	return nil
}

func (proto proto) GoAway(ctx context.Context) api.XFrame {
	return &Request{
		Type: TypeGoAway,
	}
}

// Hijacker
func (proto proto) Hijack(ctx context.Context, request api.XFrame, statusCode uint32) api.XRespFrame {
	// not supported for poc demo
	return nil
}

func (proto proto) Mapping(httpStatusCode uint32) uint32 {
	// not supported for poc demo
	return 0
}

// PoolMode returns whether pingpong or multiplex
func (proto proto) PoolMode() api.PoolMode {
	return api.Multiplex
}

func (proto proto) EnableWorkerPool() bool {
	return true
}

func (proto proto) GenerateRequestID(streamID *uint64) uint64 {
	return atomic.AddUint64(streamID, 1)
}
