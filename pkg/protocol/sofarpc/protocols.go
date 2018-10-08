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

package sofarpc

import (
	"context"
	"errors"
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/models"
	"github.com/alipay/sofa-mosn/pkg/trace"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

//All of the protocolMaps

var defaultProtocols = &protocols{
	protocolMaps: make(map[byte]Protocol),
}

type protocols struct {
	protocolMaps map[byte]Protocol
}

func DefaultProtocols() types.Protocols {
	return defaultProtocols
}

func NewProtocols(protocolMaps map[byte]Protocol) types.Protocols {
	return &protocols{
		protocolMaps: protocolMaps,
	}
}

//PROTOCOL LEVEL's Unified AppendHeaders for BOLTV1、BOLTV2、TR
func (p *protocols) EncodeHeaders(context context.Context, headers interface{}) (types.IoBuffer, error) {
	var protocolCode byte

	switch headers.(type) {
	case ProtoBasicCmd:
		protocolCode = headers.(ProtoBasicCmd).GetProtocol()
	case map[string]string:
		headersMap := headers.(map[string]string)

		if proto, exist := headersMap[SofaPropertyHeader(HeaderProtocolCode)]; exist {
			protoValue := ConvertPropertyValueUint8(proto)
			protocolCode = protoValue
		} else {
			errMsg := NoProCodeInHeader
			log.ByContext(context).Errorf(errMsg)
			err := errors.New(errMsg)
			return nil, err
		}
	default:
		errMsg := InvalidHeaderType
		log.ByContext(context).Errorf(errMsg+" headers = %+v", headers)
		err := errors.New(errMsg)
		return nil, err
	}

	if proto, exists := p.protocolMaps[protocolCode]; exists {
		//Return encoded data in map[string]string to stream layer
		return proto.GetEncoder().EncodeHeaders(context, headers)
	}

	errMsg := types.UnSupportedProCode
	err := errors.New(errMsg)
	log.ByContext(context).Errorf(errMsg+"protocolCode = %s", protocolCode)

	return nil, err
}

func (p *protocols) EncodeData(context context.Context, data types.IoBuffer) types.IoBuffer {
	return data
}

func (p *protocols) EncodeTrailers(context context.Context, trailers map[string]string) types.IoBuffer {
	return nil
}

func (p *protocols) Decode(context context.Context, data types.IoBuffer, filter types.DecodeFilter) {
	// at least 1 byte for protocol code recognize
	for data.Len() > 1 {
		logger := log.ByContext(context)
		protocolCode := data.Bytes()[0]
		maybeProtocolVersion := data.Bytes()[1]
		logger.Debugf("Decoderprotocol code = %x, maybeProtocolVersion = %x", protocolCode, maybeProtocolVersion)

		if proto, exists := p.protocolMaps[protocolCode]; exists {
			if cmd, error := proto.GetDecoder().Decode(context, data); cmd != nil && error == nil {
				if err := proto.GetCommandHandler().HandleCommand(context, cmd, filter); err != nil {
					filter.OnDecodeError(err, nil)
					break
				}
			} else if error != nil {
				// request type error, the second byte in protocol
				filter.OnDecodeError(error, nil)
				break
			} else {
				break
			}
		} else {
			errMsg := types.UnSupportedProCode
			logger.Errorf(errMsg+"protocolCode = %s", protocolCode)
			filter.OnDecodeError(errors.New(errMsg), nil)
			break
		}
	}
}

func (p *protocols) BuildSpan(context context.Context) types.Span {
	sofabuffers := SofaProtocolBuffersByContent(context)
	request := &sofabuffers.BoltReq

	if request.CmdCode == CMD_CODE_HEARTBEAT {
		return nil
	}

	span := trace.Tracer().Start(time.Now())

	traceId := request.RequestHeader[models.TRACER_ID_KEY]
	if traceId == "" {
		traceId = trace.IdGen().GenerateTraceId()
	}
	span.SetTag(trace.TRACE_ID, traceId)
	lType := context.Value(types.ContextKeyListenerType)

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

func (p *protocols) RegisterProtocol(protocolCode byte, protocol Protocol) {
	if _, exists := p.protocolMaps[protocolCode]; exists {
		log.DefaultLogger.Warnf("protocol already Exist:", protocolCode)
	} else {
		p.protocolMaps[protocolCode] = protocol
	}
}

func (p *protocols) UnRegisterProtocol(protocolCode byte) {
	if _, exists := p.protocolMaps[protocolCode]; exists {
		delete(p.protocolMaps, protocolCode)
		log.StartLogger.Debugf("unregister protocol:%x", protocolCode)
	}
}

func RegisterProtocol(protocolCode byte, protocol Protocol) {
	defaultProtocols.RegisterProtocol(protocolCode, protocol)
}

func UnRegisterProtocol(protocolCode byte) {
	defaultProtocols.UnRegisterProtocol(protocolCode)
}
