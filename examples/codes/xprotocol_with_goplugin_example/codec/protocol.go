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
	"errors"
	"fmt"
	"sync/atomic"

	"mosn.io/api"
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

type Proto struct{}

func (proto *Proto) Name() api.ProtocolName {
	return ProtocolName
}

//判断是request还是responce对象，然后添加协议部分，比如magic dir type payloadlen ，然后返回二进制流
func (proto *Proto) Encode(ctx context.Context, model interface{}) (api.IoBuffer, error) {
	switch frame := model.(type) {
	case *Request:
		return encodeRequest(ctx, frame)
	case *Response:
		return encodeResponse(ctx, frame)
	default:
		fmt.Println(ctx, "[protocol][mychain] encode with unknown command : %+v", model)
		return nil, errors.New("unknown command type")
	}
}

//读取二进制流，然后判断判断是否符合协议，再根据是request 还是responce 读取二进制流信息 返回封装好的request responce对象
func (proto *Proto) Decode(ctx context.Context, data api.IoBuffer) (interface{}, error) {
	if data.Len() >= MinimalDecodeLen {
		magic := data.Bytes()[MagicIdx]
		dir := data.Bytes()[DirIndex]

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

// Heartbeater
func (proto *Proto) Trigger(context context.Context, requestId uint64) api.XFrame {
	// not supported for poc demo
	return nil
}

func (proto *Proto) Reply(context context.Context, request api.XFrame) api.XRespFrame {
	// not supported for poc demo
	return nil
}

// Hijacker
func (proto *Proto) Hijack(context context.Context, request api.XFrame, statusCode uint32) api.XRespFrame {
	// not supported for poc demo
	return nil
}

func (proto *Proto) Mapping(httpStatusCode uint32) uint32 {
	// not supported for poc demo
	return 0
}

// PoolMode returns whether pingpong or multiplex
func (proto *Proto) PoolMode() api.PoolMode {
	return api.Multiplex
}

func (proto *Proto) EnableWorkerPool() bool {
	return true
}

func (proto *Proto) GenerateRequestID(streamID *uint64) uint64 {
	return atomic.AddUint64(streamID, 1)
}
