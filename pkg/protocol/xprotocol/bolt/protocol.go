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
	"sync/atomic"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol/internal/registry"
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

type boltProtocol struct{}

// api.Protocol
func (proto boltProtocol) Name() api.ProtocolName {
	return ProtocolName
}

func (proto boltProtocol) Encode(ctx context.Context, model interface{}) (api.IoBuffer, error) {
	switch frame := model.(type) {
	case *Request:
		return encodeRequest(ctx, frame)
	case *Response:
		return encodeResponse(ctx, frame)
	default:
		// log.Proxy.Errorf(ctx, "[protocol][bolt] encode with unknown command : %+v", model)
		// return nil, api.ErrUnknownType
		// FIXME: makes sofarpc protocol common
		// bolt and boltv2 can be handled success on a same connection
		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(ctx, "[protocol][bolt] bolt maybe receive boltv2 encode")
		}
		// TODO: use internal package to avoid cycle import temporary
		// makes bolt and boltv2 reuse in same package
		engine := registry.GetXProtocolCodec("boltv2").NewXProtocol(ctx)
		return engine.Encode(ctx, model)
	}
}

func (proto boltProtocol) Decode(ctx context.Context, data api.IoBuffer) (interface{}, error) {
	if data.Len() > 0 {
		code := data.Bytes()[0]
		if code == 0x02 { // protocol boltv2
			// TODO: use internal package to avoid cycle import temporary
			// makes bolt and boltv2 reuse in same package
			engine := registry.GetXProtocolCodec("boltv2").NewXProtocol(ctx)
			return engine.Decode(ctx, data)
		}
	}
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

// GoAwayer
func (proto boltProtocol) GoAway(ctx context.Context) api.XFrame {
	// GoAway is disabled
	config := parseConfig(ctx)
	if !config.EnableBoltGoAway {
		return nil
	}

	return &Request{
		RequestHeader: RequestHeader{
			Protocol:  ProtocolCode,
			CmdType:   CmdTypeRequest,
			CmdCode:   CmdCodeGoAway,
			Version:   ProtocolVersion,
			RequestId: 0, // this would be overwrite by stream layer
			Codec:     Hessian2Serialize,
			Timeout:   0,
		},
	}
}

// Heartbeater
func (proto boltProtocol) Trigger(ctx context.Context, requestId uint64) api.XFrame {
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

func (proto boltProtocol) Reply(ctx context.Context, request api.XFrame) api.XRespFrame {
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
func (proto boltProtocol) Hijack(ctx context.Context, request api.XFrame, statusCode uint32) api.XRespFrame {
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

func (proto boltProtocol) Mapping(httpStatusCode uint32) uint32 {
	switch httpStatusCode {
	case http.StatusOK:
		return uint32(ResponseStatusSuccess)
	case api.RouterUnavailableCode:
		return uint32(ResponseStatusNoProcessor)
	case api.NoHealthUpstreamCode:
		return uint32(ResponseStatusNoProcessor)
	case api.UpstreamOverFlowCode:
		return uint32(ResponseStatusServerThreadpoolBusy)
	case api.CodecExceptionCode:
		//Decode or Encode Error
		return uint32(ResponseStatusCodecException)
	case api.DeserialExceptionCode:
		//Hessian Exception
		return uint32(ResponseStatusServerDeserialException)
	case api.TimeoutExceptionCode:
		//Response Timeout
		return uint32(ResponseStatusTimeout)
	case api.PermissionDeniedCode:
		//Response Permission Denied
		// bolt protocol do not have a permission deny code, use server exception
		return uint32(ResponseStatusServerException)
	default:
		return uint32(ResponseStatusUnknown)
	}
}

// PoolMode returns whether pingpong or multiplex
func (proto boltProtocol) PoolMode() api.PoolMode {
	return api.Multiplex
}

func (proto boltProtocol) EnableWorkerPool() bool {
	return true
}

func (proto boltProtocol) GenerateRequestID(streamID *uint64) uint64 {
	return atomic.AddUint64(streamID, 1)
}
