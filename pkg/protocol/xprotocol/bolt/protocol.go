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
	"fmt"
	"net/http"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

/**
 * Request command protocol for v1
 * 0     1     2           4           6           8          10           12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |proto| type| cmdcode   |ver2 |   requestID           |codec|        timeout        |  classLen |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * |headerLen  | contentLen            |                             ... ...                       |
 * +-----------+-----------+-----------+                                                                                               +
 * |               className + header  + content  bytes                                            |
 * +                                                                                               +
 * |                               ... ...                                                         |
 * +-----------------------------------------------------------------------------------------------+
 *
 * proto: code for protocol
 * type: request/response/request oneway
 * cmdcode: code for remoting command
 * ver2:version for remoting command
 * requestID: id of request
 * codec: code for codec
 * headerLen: length of header
 * contentLen: length of content
 *
 * Response command protocol for v1
 * 0     1     2     3     4           6           8          10           12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |proto| type| cmdcode   |ver2 |   requestID           |codec|respstatus |  classLen |headerLen  |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * | contentLen            |                  ... ...                                              |
 * +-----------------------+                                                                       +
 * |                         className + header  + content  bytes                                  |
 * +                                                                                               +
 * |                               ... ...                                                         |
 * +-----------------------------------------------------------------------------------------------+
 * respstatus: response status
 */

func init() {
	xprotocol.RegisterProtocol(ProtocolName, &boltProtocol{})
}

type boltProtocol struct{}

// types.Protocol
func (proto *boltProtocol) Name() types.ProtocolName {
	return ProtocolName
}

func (proto *boltProtocol) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	switch frame := model.(type) {
	case *Request:
		return encodeRequest(ctx, frame)
	case *Response:
		return encodeResponse(ctx, frame)
	default:
		log.Proxy.Errorf(ctx, "[protocol][bolt] encode with unknown command : %+v", model)
		return nil, xprotocol.ErrUnknownType
	}
}

func (proto *boltProtocol) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	if data.Len() >= LessLen {
		cmdType := data.Bytes()[1]

		switch cmdType {
		case CmdTypeRequest:
			return decodeRequest(ctx, data, false)
		case CmdTypeRequestOneway:
			return decodeRequest(ctx, data, true)
		case CmdTypeResponse:
			return decodeResponse(ctx, data)
		default:
			// unknown cmd type
			return nil, fmt.Errorf("Decode Error, type = %s, value = %d", UnKnownCmdType, cmdType)
		}
	}

	return nil, nil
}

// Heartbeater
func (proto *boltProtocol) Trigger(requestId uint64) xprotocol.XFrame {
	return &Request{
		RequestHeader: RequestHeader{
			Protocol:  ProtocolCode,
			CmdType:   CmdTypeRequest,
			CmdCode:   CmdCodeHeartbeat,
			Version:   1,
			RequestId: uint32(requestId),
			Codec:     Hessian2Serialize,
			Timeout:   -1,
		},
	}
}

func (proto *boltProtocol) Reply(request xprotocol.XFrame) xprotocol.XRespFrame {
	return &Response{
		ResponseHeader: ResponseHeader{
			Protocol:       ProtocolCode,
			CmdType:        CmdTypeResponse,
			CmdCode:        CmdCodeHeartbeat,
			Version:        ProtocolVersion,
			RequestId:      uint32(request.GetRequestId()),
			Codec:          Hessian2Serialize,
			ResponseStatus: ResponseStatusSuccess,
		},
	}
}

// Hijacker
func (proto *boltProtocol) Hijack(request xprotocol.XFrame, statusCode uint32) xprotocol.XRespFrame {
	return &Response{
		ResponseHeader: ResponseHeader{
			Protocol:       ProtocolCode,
			CmdType:        CmdTypeResponse,
			CmdCode:        CmdCodeRpcResponse,
			Version:        ProtocolVersion,
			RequestId:      0,                 // this would be overwrite by stream layer
			Codec:          Hessian2Serialize, //todo: read default codec from config
			ResponseStatus: uint16(statusCode),
		},
	}
}

func (proto *boltProtocol) Mapping(httpStatusCode uint32) uint32 {
	switch httpStatusCode {
	case http.StatusOK:
		return uint32(ResponseStatusSuccess)
	case types.RouterUnavailableCode:
		return uint32(ResponseStatusNoProcessor)
	case types.NoHealthUpstreamCode:
		return uint32(ResponseStatusConnectionClosed)
	case types.UpstreamOverFlowCode:
		return uint32(ResponseStatusServerThreadpoolBusy)
	case types.CodecExceptionCode:
		//Decode or Encode Error
		return uint32(ResponseStatusCodecException)
	case types.DeserialExceptionCode:
		//Hessian Exception
		return uint32(ResponseStatusServerDeserialException)
	case types.TimeoutExceptionCode:
		//Response Timeout
		return uint32(ResponseStatusTimeout)
	default:
		return uint32(ResponseStatusUnknown)
	}
}
