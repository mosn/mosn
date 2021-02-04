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

package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"sync/atomic"

	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

const (
	MessageUnknown       byte = 0
	MessageTypeCall           = 1
	MessageTypeReply          = 2
	MessageTypeException      = 3
	MessageTypeOneway         = 4
)

type Message struct {
	Version string
	Type    byte
	NameLen int32
	Name    string
	SeqId   int32
}

type thriftProtocol struct{}

func (proto *thriftProtocol) PoolMode() api.PoolMode {
	return api.PingPong
}

func (proto *thriftProtocol) EnableWorkerPool() bool {
	return false
}

func (proto *thriftProtocol) GenerateRequestID(u *uint64) uint64 {
	return atomic.AddUint64(u, 1)
}

// types.Protocol
func (proto *thriftProtocol) Name() api.ProtocolName {
	return ProtocolName
}

func (proto *thriftProtocol) Encode(ctx context.Context, model interface{}) (buffer.IoBuffer, error) {
	switch frame := model.(type) {
	case *Request:
		return encodeRequest(ctx, frame)
	case *Response:
		return encodeResponse(ctx, frame)
	default:
		return nil, api.ErrUnknownType
	}
}

func messageType(b byte) byte {
	switch b {
	case 1:
		return MessageTypeCall
	case 2:
		return MessageTypeReply
	case 3:
		return MessageTypeException
	case 4:
		return MessageTypeOneway
	default:
		return MessageUnknown
	}
}

func (proto *thriftProtocol) decodeMessage(ctx context.Context, data buffer.IoBuffer) (Message, error) {
	bytesLen := data.Len()
	bytes := data.Bytes()

	// 1. least bytes to decode header is RequestHeaderLen(22)
	if bytesLen < MessageMinLen {
		// todo
		panic("implement me")
	}

	// 2. least bytes to decode whole frame
	messageType := messageType(bytes[3])
	nameLen := binary.BigEndian.Uint32(bytes[4:8])
	name := string(bytes[8 : 8+nameLen])
	seqId := binary.BigEndian.Uint32(bytes[8+nameLen : 8+nameLen+4])

	return Message{
		Version: "", // todo
		Type:    messageType,
		NameLen: int32(nameLen),
		Name:    name,
		SeqId:   int32(seqId),
	}, nil
}

func (proto *thriftProtocol) Decode(ctx context.Context, data buffer.IoBuffer) (interface{}, error) {
	if data.Len() >= MessageMinLen {
		message, err := proto.decodeMessage(ctx, data)
		if err != nil {
			return nil, err
		}

		cmdType := data.Bytes()[0]

		switch message.Type {
		case MessageTypeCall:
			return decodeRequest(ctx, data, false, message)
		case MessageTypeOneway:
			return decodeRequest(ctx, data, true, message)
		case MessageTypeReply:
			return decodeResponse(ctx, data, message)
		case MessageTypeException:
			// todo
		default:
			// unknown cmd type
			return nil, fmt.Errorf("Decode Error, type = %s, value = %d", UnKnownCmdType, cmdType)
		}
	}

	return nil, nil
}

// Heartbeater
func (proto *thriftProtocol) Trigger(requestId uint64) api.XFrame {
	request := &Request{
		RequestHeader: RequestHeader{
			Protocol: ProtocolCode,
			CmdType:  CmdTypeRequest,
			CmdCode:  CmdCodeHeartbeat,
			//Version:   1,
			RequestId: uint32(requestId),
			//Codec:     Hessian2Serialize,
			Timeout: -1,
		},
	}

	request.Data = buffer.GetIoBuffer(len(pingCmd))

	//5. copy data for io multiplexing
	request.Data.Write(pingCmd)
	request.rawData = request.Data.Bytes()
	request.rawContent = request.rawData[:]
	request.Content = buffer.NewIoBufferBytes(request.rawContent)

	return request
}

func (proto *thriftProtocol) Reply(request api.XFrame) api.XRespFrame {
	response := &Response{
		ResponseHeader: ResponseHeader{
			Protocol: ProtocolCode,
			CmdType:  CmdTypeResponse,
			CmdCode:  CmdCodeHeartbeat,
			//Version:        ProtocolVersion,
			RequestId: uint32(request.GetRequestId()),
			//Codec:          Hessian2Serialize,
			ResponseStatus: ResponseStatusSuccess,
		},
	}

	response.Data = buffer.GetIoBuffer(len(pongCmd))
	//5. copy data for io multiplexing
	response.Data.Write(pongCmd)
	response.rawData = response.Data.Bytes()
	response.rawContent = response.rawData[:]
	response.Content = buffer.NewIoBufferBytes(response.rawContent)

	return response
}

// Hijacker
func (proto *thriftProtocol) Hijack(request api.XFrame, statusCode uint32) api.XRespFrame {
	return &Response{
		ResponseHeader: ResponseHeader{
			Protocol: ProtocolCode,
			CmdType:  CmdTypeResponse,
			CmdCode:  CmdCodeResponse,
			//Version:        ProtocolVersion,
			RequestId:      0,                 // this would be overwrite by stream layer
			Codec:          Hessian2Serialize, //todo: read default codec from config
			ResponseStatus: uint16(statusCode),
		},
	}
}

const (
	CodecExceptionCode    = 0
	UnknownCode           = 2
	DeserialExceptionCode = 3
	SuccessCode           = 200
	PermissionDeniedCode  = 403
	RouterUnavailableCode = 404
	InternalErrorCode     = 500
	NoHealthUpstreamCode  = 502
	UpstreamOverFlowCode  = 503
	TimeoutExceptionCode  = 504
	LimitExceededCode     = 509
)

func (proto *thriftProtocol) Mapping(httpStatusCode uint32) uint32 {
	switch httpStatusCode {
	case http.StatusOK:
		return uint32(ResponseStatusSuccess)
	case RouterUnavailableCode:
		return uint32(ResponseStatusNoProcessor)
	case NoHealthUpstreamCode:
		return uint32(ResponseStatusConnectionClosed)
	case UpstreamOverFlowCode:
		return uint32(ResponseStatusServerThreadpoolBusy)
	case CodecExceptionCode:
		//Decode or Encode Error
		return uint32(ResponseStatusCodecException)
	case DeserialExceptionCode:
		//Hessian Exception
		return uint32(ResponseStatusServerDeserialException)
	case TimeoutExceptionCode:
		//Response Timeout
		return uint32(ResponseStatusTimeout)
	default:
		return uint32(ResponseStatusUnknown)
	}
}
