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
	"fmt"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/models"
	"github.com/alipay/sofa-mosn/pkg/trace"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/types"

	mosnctx "github.com/alipay/sofa-mosn/pkg/context"
)

var (
	BoltCodec = &boltCodec{}
)

func init() {
	sofarpc.RegisterProtocol(sofarpc.PROTOCOL_CODE_V1, BoltCodec, BoltCodec, &BoltV1SpanBuilder{})
	sofarpc.RegisterResponseBuilder(sofarpc.PROTOCOL_CODE_V1, BoltCodec)
	sofarpc.RegisterHeartbeatBuilder(sofarpc.PROTOCOL_CODE_V1, BoltCodec)
}

// ~~ types.Encoder
// ~~ types.Decoder
type boltCodec struct{}

func (c *boltCodec) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	switch cmd := model.(type) {
	case *sofarpc.BoltRequest:
		return encodeRequest(ctx, cmd)
	case *sofarpc.BoltResponse:
		return encodeResponse(ctx, cmd)
	default:
		log.Proxy.Errorf(ctx, "[protocol][sofarpc] boltv1 encode with unknown command : %+v", model)
		return nil, rpc.ErrUnknownType
	}
}

func encodeRequest(ctx context.Context, cmd *sofarpc.BoltRequest) (types.IoBuffer, error) {
	// serialize classname and header
	if cmd.RequestClass != "" {
		cmd.ClassName = serialize.UnsafeStrToByte(cmd.RequestClass)
		cmd.ClassLen = int16(len(cmd.ClassName))
	}

	headerLen := int(cmd.HeaderLen)
	if headerLen == 0 && cmd.RequestHeader != nil {
		headerLen = 256
	}

	var b [4]byte

	size := sofarpc.REQUEST_HEADER_LEN_V1 + int(cmd.ClassLen) + headerLen
	//buf := sofarpc.GetBuffer(context, size)

	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	buf := protocolCtx.GetReqHeader(size)
	//buf := buffer.NewIoBuffer(size)

	b[0] = cmd.Protocol
	buf.Write(b[0:1])
	b[0] = cmd.CmdType
	buf.Write(b[0:1])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.CmdCode))
	buf.Write(b[0:2])

	b[0] = cmd.Version
	buf.Write(b[0:1])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ReqID))
	buf.Write(b[0:4])

	b[0] = cmd.Codec
	buf.Write(b[0:1])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.Timeout))
	buf.Write(b[0:4])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.ClassLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.HeaderLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ContentLen))
	buf.Write(b[0:4])

	if cmd.ClassLen > 0 {
		buf.Write(cmd.ClassName)
	}

	if cmd.RequestHeader != nil {
		l := buf.Len()
		serialize.Instance.SerializeMap(cmd.RequestHeader, buf)
		headerLen = buf.Len() - l

		// reset HeaderLen
		headerData := buf.Bytes()[sofarpc.RequestHeaderLenIndex:]
		binary.BigEndian.PutUint16(headerData, uint16(headerLen))
	} else {
		buf.Write(cmd.HeaderMap)
	}

	return buf, nil
}

func encodeResponse(ctx context.Context, cmd *sofarpc.BoltResponse) (types.IoBuffer, error) {
	// serialize classname and header
	if cmd.ResponseClass != "" {
		cmd.ClassName = serialize.UnsafeStrToByte(cmd.ResponseClass)
		cmd.ClassLen = int16(len(cmd.ClassName))
	}

	headerLen := int(cmd.HeaderLen)
	if headerLen == 0 && cmd.ResponseHeader != nil {
		headerLen = 256
	}

	var b [4]byte
	// todo: reuse bytes @boqin
	size := sofarpc.RESPONSE_HEADER_LEN_V1 + int(cmd.ClassLen) + headerLen
	//buf := sofarpc.GetBuffer(context, size)
	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	buf := protocolCtx.GetRspHeader(size)
	//buf := buffer.GetIoBuffer(size)

	b[0] = cmd.Protocol
	buf.Write(b[0:1])
	b[0] = cmd.CmdType
	buf.Write(b[0:1])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.CmdCode))
	buf.Write(b[0:2])

	b[0] = cmd.Version
	buf.Write(b[0:1])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ReqID))
	buf.Write(b[0:4])

	b[0] = cmd.Codec
	buf.Write(b[0:1])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.ResponseStatus))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.ClassLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.HeaderLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ContentLen))
	buf.Write(b[0:4])

	if cmd.ClassLen > 0 {
		buf.Write(cmd.ClassName)
	}

	if cmd.ResponseHeader != nil {
		l := buf.Len()
		serialize.Instance.SerializeMap(cmd.ResponseHeader, buf)
		headerLen = buf.Len() - l

		// reset HeaderLen
		headerData := buf.Bytes()[sofarpc.ResponseHeaderLenIndex:]
		binary.BigEndian.PutUint16(headerData, uint16(headerLen))
	} else {
		buf.Write(cmd.HeaderMap)
	}

	return buf, nil
}

func (c *boltCodec) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	readableBytes := data.Len()
	read := 0
	var cmd interface{}

	if readableBytes >= sofarpc.LESS_LEN_V1 {
		bytesData := data.Bytes()
		cmdType := bytesData[1]

		//1. request
		if cmdType == sofarpc.REQUEST || cmdType == sofarpc.REQUEST_ONEWAY {
			if readableBytes >= sofarpc.REQUEST_HEADER_LEN_V1 {

				cmdCode := binary.BigEndian.Uint16(bytesData[2:4])
				ver2 := bytesData[4]
				requestID := binary.BigEndian.Uint32(bytesData[5:9])
				codec := bytesData[9]
				timeout := int32(binary.BigEndian.Uint32(bytesData[10:14]))
				classLen := binary.BigEndian.Uint16(bytesData[14:16])
				headerLen := binary.BigEndian.Uint16(bytesData[16:18])
				contentLen := binary.BigEndian.Uint32(bytesData[18:22])

				read = sofarpc.REQUEST_HEADER_LEN_V1
				var class, header, content []byte

				if readableBytes >= read+int(classLen)+int(headerLen)+int(contentLen) {
					if classLen > 0 {
						class = bytesData[read : read+int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytesData[read : read+int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytesData[read : read+int(contentLen)]
						read += int(contentLen)
					}

					data.Drain(read)

				} else { // not enough data
					log.Proxy.Debugf(ctx, "[protocol][sofarpc] boltv1 decode request, no enough data for fully decode")
					return cmd, nil
				}

				buffers := sofarpc.SofaProtocolBuffersByContext(ctx)
				request := &buffers.BoltReq
				//request := &sofarpc.BoltRequest{}
				request.Protocol = sofarpc.PROTOCOL_CODE_V1
				request.CmdType = cmdType
				request.CmdCode = int16(cmdCode)
				request.Version = ver2
				request.ReqID = requestID
				request.Codec = codec
				request.Timeout = int(timeout)
				request.ClassLen = int16(classLen)
				request.HeaderLen = int16(headerLen)
				request.ContentLen = int(contentLen)
				request.ClassName = class
				request.HeaderMap = header
				// avoid valid IoBuffer with empty buffer
				if content != nil {
					request.Content = buffer.NewIoBufferBytes(content)
				}
				sofarpc.DeserializeBoltRequest(ctx, request)

				cmd = request
			}
		} else if cmdType == sofarpc.RESPONSE {
			//2. response
			if readableBytes >= sofarpc.RESPONSE_HEADER_LEN_V1 {

				cmdCode := binary.BigEndian.Uint16(bytesData[2:4])
				ver2 := bytesData[4]
				requestID := binary.BigEndian.Uint32(bytesData[5:9])
				codec := bytesData[9]
				status := binary.BigEndian.Uint16(bytesData[10:12])
				classLen := binary.BigEndian.Uint16(bytesData[12:14])
				headerLen := binary.BigEndian.Uint16(bytesData[14:16])
				contentLen := binary.BigEndian.Uint32(bytesData[16:20])

				read = sofarpc.RESPONSE_HEADER_LEN_V1
				var class, header, content []byte

				if readableBytes >= read+int(classLen)+int(headerLen)+int(contentLen) {
					if classLen > 0 {
						class = bytesData[read : read+int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytesData[read : read+int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytesData[read : read+int(contentLen)]
						read += int(contentLen)
					}

					data.Drain(read)
				} else {
					// not enough data
					log.Proxy.Debugf(ctx, "[protocol][sofarpc] boltv1 decode response, no enough data for fully decode")
					return cmd, nil
				}

				buffers := sofarpc.SofaProtocolBuffersByContext(ctx)
				response := &buffers.BoltRsp
				//response := &sofarpc.BoltResponse{}
				response.Protocol = sofarpc.PROTOCOL_CODE_V1
				response.CmdType = cmdType
				response.CmdCode = int16(cmdCode)
				response.Version = ver2
				response.ReqID = requestID
				response.Codec = codec
				response.ResponseStatus = int16(status)
				response.ClassLen = int16(classLen)
				response.HeaderLen = int16(headerLen)
				response.ContentLen = int(contentLen)
				response.ClassName = class
				response.HeaderMap = header
				response.Content = buffer.NewIoBufferBytes(content)

				response.ResponseTimeMillis = time.Now().UnixNano() / int64(time.Millisecond)
				sofarpc.DeserializeBoltResponse(ctx, response)

				cmd = response
			}
		} else {
			// 3. unknown type error
			return nil, fmt.Errorf("Decode Error, type = %s, value = %d", sofarpc.UnKnownCmdType, cmdType)
		}
	}

	return cmd, nil
}

// ~ HeartbeatBuilder
func (c *boltCodec) Trigger() sofarpc.SofaRpcCmd {
	return &sofarpc.BoltRequest{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.HEARTBEAT,
		Version:  1,
		ReqID:    0,                          // this would be overwrite by stream layer
		Codec:    sofarpc.HESSIAN2_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}
}

func (c *boltCodec) Reply() sofarpc.SofaRpcCmd {
	return &sofarpc.BoltResponse{
		Protocol:       sofarpc.PROTOCOL_CODE_V1,
		CmdType:        sofarpc.RESPONSE,
		CmdCode:        sofarpc.HEARTBEAT,
		Version:        1,
		ReqID:          0,                          // this would be overwrite by stream layer
		Codec:          sofarpc.HESSIAN2_SERIALIZE, //todo: read default codec from config
		ResponseStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
	}
}

// ~ ResponseBuilder
func (c *boltCodec) BuildResponse(respStatus int16) sofarpc.SofaRpcCmd {
	return &sofarpc.BoltResponse{
		Protocol:       sofarpc.PROTOCOL_CODE_V1,
		CmdType:        sofarpc.RESPONSE,
		CmdCode:        sofarpc.RPC_RESPONSE,
		Version:        1,
		ReqID:          0,                          // this would be overwrite by stream layer
		Codec:          sofarpc.HESSIAN2_SERIALIZE, //todo: read default codec from config
		ResponseStatus: respStatus,
	}
}

type BoltV1SpanBuilder struct {
}

func (sb *BoltV1SpanBuilder) BuildSpan(args ...interface{}) types.Span {
	if len(args) == 0 {
		return nil
	}

	ctx, ok := args[0].(context.Context)
	if !ok {
		log.Proxy.Errorf(ctx, "[protocol][sofarpc] boltv1 span build failed, first arg unexpected:%+v", args[0])
		return nil
	}

	request, ok := args[1].(*sofarpc.BoltRequest)
	if !ok {
		log.Proxy.Errorf(ctx, "[protocol][sofarpc] boltv1 span build failed, second arg unexpected:%+v", args[0])
		return nil
	}

	if request.CmdCode == sofarpc.HEARTBEAT {
		return nil
	}

	span := trace.Tracer().Start(time.Now())

	traceId := request.RequestHeader[models.TRACER_ID_KEY]
	if traceId == "" {
		// TODO: set generated traceId into header?
		traceId = trace.IdGen().GenerateTraceId()
	}

	span.SetTag(trace.TRACE_ID, traceId)
	lType := mosnctx.Get(ctx, types.ContextKeyListenerType)
	if lType == nil {
		return span
	}

	spanId := request.RequestHeader[models.RPC_ID_KEY]
	if spanId == "" {
		spanId = "0" // Generate a new span id
	} else {
		if lType == v2.INGRESS {
			trace.AddSpanIdGenerator(trace.NewSpanIdGenerator(traceId, spanId))
		} else if lType == v2.EGRESS {
			span.SetTag(trace.PARENT_SPAN_ID, spanId)
			spanKey := &trace.SpanKey{TraceId: traceId, SpanId: spanId}
			if spanIdGenerator := trace.GetSpanIdGenerator(spanKey); spanIdGenerator != nil {
				spanId = spanIdGenerator.GenerateNextChildIndex()
			}
		}
	}
	span.SetTag(trace.SPAN_ID, spanId)

	if lType == v2.EGRESS {
		span.SetTag(trace.APP_NAME, request.RequestHeader[models.APP_NAME])
	}
	span.SetTag(trace.SPAN_TYPE, string(lType.(v2.ListenerType)))
	span.SetTag(trace.METHOD_NAME, request.RequestHeader[models.TARGET_METHOD])
	span.SetTag(trace.PROTOCOL, "bolt")
	span.SetTag(trace.SERVICE_NAME, request.RequestHeader[models.SERVICE_KEY])
	span.SetTag(trace.BAGGAGE_DATA, request.RequestHeader[models.SOFA_TRACE_BAGGAGE_DATA])
	return span
}
