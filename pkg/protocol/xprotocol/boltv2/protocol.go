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
	"fmt"
	"net/http"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/types"
)

/**
 * Request command protocol for v2
 * 0     1     2           4           6           8          10     11     12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+------+-----+-----+-----+-----+
 * |proto| ver1|type | cmdcode   |ver2 |   requestID           |codec|switch|   timeout             |
 * +-----------+-----------+-----------+-----------+-----------+------------+-----------+-----------+
 * |classLen   |headerLen  |contentLen             |           ...                                  |
 * +-----------+-----------+-----------+-----------+                                                +
 * |               className + header  + content  bytes                                             |
 * +                                                                                                +
 * |                               ... ...                                  | CRC32(optional)       |
 * +------------------------------------------------------------------------------------------------+
 *
 * proto: code for protocol
 * ver1: version for protocol
 * type: request/response/request oneway
 * cmdcode: code for remoting command
 * ver2:version for remoting command
 * requestID: id of request
 * codec: code for codec
 * switch: function switch for protocol
 * headerLen: length of header
 * contentLen: length of content
 * CRC32: CRC32 of the frame(Exists when ver1 > 1)
 *
 * Response command protocol for v2
 * 0     1     2     3     4           6           8          10     11    12          14          16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+------+-----+-----+-----+-----+
 * |proto| ver1| type| cmdcode   |ver2 |   requestID           |codec|switch|respstatus |  classLen |
 * +-----------+-----------+-----------+-----------+-----------+------------+-----------+-----------+
 * |headerLen  | contentLen            |                      ...                                   |
 * +-----------------------------------+                                                            +
 * |               className + header  + content  bytes                                             |
 * +                                                                                                +
 * |                               ... ...                                  | CRC32(optional)       |
 * +------------------------------------------------------------------------------------------------+
 * respstatus: response status
 */

func init() {
	xprotocol.RegisterProtocol(ProtocolName, &boltv2Protocol{})
}

type boltv2Protocol struct{}

func (proto *boltv2Protocol) Name() types.ProtocolName {
	return ProtocolName
}

func (proto *boltv2Protocol) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
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

func (proto *boltv2Protocol) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	if data.Len() >= LessLen {
		cmdType := data.Bytes()[1]

		switch cmdType {
		case bolt.CmdTypeRequest:
			return decodeRequest(ctx, data, false)
		case bolt.CmdTypeRequestOneway:
			return decodeRequest(ctx, data, true)
		case bolt.CmdTypeResponse:
			return decodeResponse(ctx, data)
		default:
			// unknown cmd type
			return nil, fmt.Errorf("Decode Error, type = %s, value = %d", bolt.UnKnownCmdType, cmdType)
		}
	}

	return nil, nil
}

// heartbeater
func (proto *boltv2Protocol) Trigger(requestId uint64) xprotocol.XFrame {
	return &Request{
		RequestHeader: RequestHeader{
			RequestHeader: bolt.RequestHeader{
				Protocol:  ProtocolCode,
				CmdType:   bolt.CmdTypeRequest,
				CmdCode:   bolt.CmdCodeHeartbeat,
				Version:   ProtocolVersion,
				RequestId: uint32(requestId),
				Codec:     bolt.Hessian2Serialize,
				Timeout:   -1,
			},
		},
	}
}

func (proto *boltv2Protocol) Reply(request xprotocol.XFrame) xprotocol.XRespFrame {
	return &Response{
		ResponseHeader: ResponseHeader{
			ResponseHeader: bolt.ResponseHeader{
				Protocol:       ProtocolCode,
				CmdType:        bolt.CmdTypeResponse,
				CmdCode:        bolt.CmdCodeHeartbeat,
				Version:        ProtocolVersion,
				RequestId:      uint32(request.GetRequestId()),
				Codec:          bolt.Hessian2Serialize,
				ResponseStatus: bolt.ResponseStatusSuccess,
			},
		},
	}
}

// hijacker
func (proto *boltv2Protocol) Hijack(request xprotocol.XFrame, statusCode uint32) xprotocol.XRespFrame {
	return &Response{
		ResponseHeader: ResponseHeader{
			ResponseHeader: bolt.ResponseHeader{
				Protocol:       ProtocolCode,
				CmdType:        bolt.CmdTypeResponse,
				CmdCode:        bolt.CmdCodeRpcResponse,
				Version:        ProtocolVersion,
				RequestId:      0,                      // this would be overwrite by stream layer
				Codec:          bolt.Hessian2Serialize, //todo: read default codec from config
				ResponseStatus: uint16(statusCode),
			},
		},
	}
}

func (proto *boltv2Protocol) Mapping(httpStatusCode uint32) uint32 {
	switch httpStatusCode {
	case http.StatusOK:
		return uint32(bolt.ResponseStatusSuccess)
	case types.RouterUnavailableCode:
		return uint32(bolt.ResponseStatusNoProcessor)
	case types.NoHealthUpstreamCode:
		return uint32(bolt.ResponseStatusConnectionClosed)
	case types.UpstreamOverFlowCode:
		return uint32(bolt.ResponseStatusServerThreadpoolBusy)
	case types.CodecExceptionCode:
		//Decode or Encode Error
		return uint32(bolt.ResponseStatusCodecException)
	case types.DeserialExceptionCode:
		//Hessian Exception
		return uint32(bolt.ResponseStatusServerDeserialException)
	case types.TimeoutExceptionCode:
		//Response Timeout
		return uint32(bolt.ResponseStatusTimeout)
	default:
		return uint32(bolt.ResponseStatusUnknown)
	}
}
