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

package dubbo

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync/atomic"

	hessian "github.com/apache/dubbo-go-hessian2"
	"mosn.io/api"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

/**
* Dubbo protocol
* Request & Response: (byte)
* 0           1           2           3           4           5           6           7           8
* +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
* |magic high | magic low |  flag     | status    |               id                              |
* +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
* |      id                                       |               data length                     |
* +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
* |                               payload                                                         |
* +-----------------------------------------------------------------------------------------------+
* magic: 0xdabb
*
* flag: (bit offset)
* 0           1           2           3           4           5           6           7           8
* +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
* |              serialization id                             |  event    | two way   |   req/rsp |
* +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
* event: 1 mean ping
* two way: 1 mean req & rsp pair
* req/rsp: 1 mean req
 */
func init() {
	xprotocol.RegisterProtocol(ProtocolName, &dubboProtocol{})
}

var MagicTag = []byte{0xda, 0xbb}

type dubboProtocol struct{}

func (proto *dubboProtocol) Name() types.ProtocolName {
	return ProtocolName
}

func (proto *dubboProtocol) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	if frame, ok := model.(*Frame); ok {
		if frame.Direction == EventRequest {
			return encodeRequest(ctx, frame)
		} else if frame.Direction == EventResponse {
			return encodeResponse(ctx, frame)
		}
	}
	log.Proxy.Errorf(ctx, "[protocol][dubbo] encode with unknown command : %+v", model)
	return nil, api.ErrUnknownType
}

func (proto *dubboProtocol) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	if data.Len() >= HeaderLen {
		// check frame size
		payLoadLen := binary.BigEndian.Uint32(data.Bytes()[DataLenIdx:(DataLenIdx + DataLenSize)])
		if data.Len() >= (HeaderLen + int(payLoadLen)) {
			frame, err := decodeFrame(ctx, data)
			if err != nil {
				// unknown cmd type
				return nil, fmt.Errorf("[protocol][dubbo] Decode Error, type = %s , err = %v", UnKnownCmdType, err)
			}
			return frame, err
		}
	}
	return nil, nil
}

// heartbeater
func (proto *dubboProtocol) Trigger(ctx context.Context, requestId uint64) api.XFrame {
	// not support
	return nil
}

func (proto *dubboProtocol) Reply(ctx context.Context, request api.XFrame) api.XRespFrame {
	// TODO make readable
	return &Frame{
		Header: Header{
			Magic:   MagicTag,
			Flag:    0x22,
			Status:  0x14,
			Id:      request.GetRequestId(),
			DataLen: 0x02,
		},
		payload: []byte{0x4e, 0x4e},
	}
}

// https://dubbo.apache.org/zh-cn/blog/dubbo-protocol.html
// hijacker
func (proto *dubboProtocol) Hijack(ctx context.Context, request api.XFrame, statusCode uint32) api.XRespFrame {
	dubboStatus, ok := dubboMosnStatusMap[int(statusCode)]
	if !ok {
		dubboStatus = dubboStatusInfo{
			Status: hessian.Response_SERVICE_ERROR,
			Msg:    fmt.Sprintf("%d status not define", statusCode),
		}
	}

	encoder := hessian.NewEncoder()
	encoder.Encode(dubboStatus.Msg)
	payload := encoder.Buffer()
	return &Frame{
		Header: Header{
			Magic:   MagicTag,
			Flag:    0x02,
			Status:  dubboStatus.Status,
			Id:      request.GetRequestId(),
			DataLen: uint32(len(payload)),
		},
		payload: payload,
	}
}

func (proto *dubboProtocol) Mapping(httpStatusCode uint32) uint32 {
	return httpStatusCode
}

// PoolMode returns whether pingpong or multiplex
func (proto *dubboProtocol) PoolMode() api.PoolMode {
	return api.Multiplex
}

func (proto *dubboProtocol) EnableWorkerPool() bool {
	return true
}

func (proto *dubboProtocol) GenerateRequestID(streamID *uint64) uint64 {
	return atomic.AddUint64(streamID, 1)
}
