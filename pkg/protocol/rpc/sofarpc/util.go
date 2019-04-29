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

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
)

// NewResponse build sofa response msg according to given protocol code and respStatus
func NewResponse(protocolCode byte, respStatus int16) SofaRpcCmd {
	if builder, ok := responseFactory[protocolCode]; ok {
		return builder.BuildResponse(respStatus)
	}
	return nil
}

// NewHeartbeat
// New Heartbeat for given protocol, requestID should be specified by caller's own logic
func NewHeartbeat(protocolCode byte) SofaRpcCmd {
	if builder, ok := heartbeatFactory[protocolCode]; ok {
		return builder.Trigger()
	}
	return nil
}

// NewHeartbeatAck
// New Heartbeat ack for given protocol, requestID should be specified by caller's own logic
func NewHeartbeatAck(protocolCode byte) SofaRpcCmd {
	if builder, ok := heartbeatFactory[protocolCode]; ok {
		return builder.Reply()
	}
	return nil
}

func DeserializeBoltRequest(ctx context.Context, request *BoltRequest) {
	//get instance
	serializeIns := serialize.Instance

	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	request.RequestHeader = protocolCtx.GetReqHeaders()

	//request.RequestHeader = make(map[string]string, 8)

	logger := log.Proxy
	debugEnabled := logger.GetLogLevel() >= log.DEBUG

	//deserialize header
	serializeIns.DeserializeMap(request.HeaderMap, request.RequestHeader)
	if debugEnabled {
		logger.Debugf(ctx, "[protocol][sofarpc] deserialize bolt request, header: %v", request.RequestHeader)
	}

	//deserialize class name
	request.RequestClass = string(request.ClassName)
	if debugEnabled {
		logger.Debugf(ctx, "[protocol][sofarpc] deserialize bolt request, className: %s", request.RequestClass)
	}
}

func DeserializeBoltResponse(ctx context.Context, response *BoltResponse) {
	//get instance
	serializeIns := serialize.Instance

	//logger
	logger := log.Proxy
	debugEnabled := logger.GetLogLevel() >= log.DEBUG

	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	response.ResponseHeader = protocolCtx.GetRspHeaders()

	//response.ResponseHeader = make(map[string]string, 8)

	//deserialize header
	serializeIns.DeserializeMap(response.HeaderMap, response.ResponseHeader)
	if debugEnabled {
		logger.Debugf(ctx, "[protocol][sofarpc] deserialize bolt response, header: %+v", response.ResponseHeader)
	}

	//deserialize class name
	response.ResponseClass = string(response.ClassName)
	if debugEnabled {
		logger.Debugf(ctx, "[protocol][sofarpc] deserialize bolt response, className: %s", response.ResponseClass)
	}
}
