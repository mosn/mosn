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

package conv

import (
	"context"
	"errors"
	"reflect"
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
)

var (
	// PropertyHeaders map the cmdkey and its data type
	PropertyHeadersV2 = make(map[string]reflect.Kind, 14)
	boltv2            = new(boltv2conv)
)

func init() {
	PropertyHeadersV2[sofarpc.HeaderProtocolCode] = reflect.Uint8
	PropertyHeadersV2[sofarpc.HeaderCmdType] = reflect.Uint8
	PropertyHeadersV2[sofarpc.HeaderCmdCode] = reflect.Int16
	PropertyHeadersV2[sofarpc.HeaderVersion] = reflect.Uint8
	PropertyHeadersV2[sofarpc.HeaderReqID] = reflect.Uint32
	PropertyHeadersV2[sofarpc.HeaderCodec] = reflect.Uint8
	PropertyHeadersV2[sofarpc.HeaderClassLen] = reflect.Int16
	PropertyHeadersV2[sofarpc.HeaderHeaderLen] = reflect.Int16
	PropertyHeadersV2[sofarpc.HeaderContentLen] = reflect.Int
	PropertyHeadersV2[sofarpc.HeaderTimeout] = reflect.Int
	PropertyHeadersV2[sofarpc.HeaderRespStatus] = reflect.Int16
	PropertyHeadersV2[sofarpc.HeaderRespTimeMills] = reflect.Int64
	PropertyHeadersV2[sofarpc.HeaderVersion1] = reflect.Uint8
	PropertyHeadersV2[sofarpc.HeaderSwitchCode] = reflect.Uint8

	sofarpc.RegisterConv(sofarpc.PROTOCOL_CODE_V2, boltv2)
}

type boltv2conv struct{}

func (b *boltv2conv) MapToCmd(ctx context.Context, headers map[string]string) (sofarpc.SofaRpcCmd, error) {
	if len(headers) < 10 {
		return nil, errors.New("headers count not enough")
	}

	value := sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderProtocolCode)
	protocolCode := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderVersion1)
	ver1 := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderCmdType)
	cmdType := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderCmdCode)
	cmdCode := sofarpc.ConvertPropertyValueInt16(value)
	value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderVersion)
	version := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderReqID)
	requestID := sofarpc.ConvertPropertyValueUint32(value)
	value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderCodec)
	codec := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderSwitchCode)
	switchcode := sofarpc.ConvertPropertyValueUint8(value)
	//value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderClassLen)
	//classLength := sofarpc.ConvertPropertyValueInt16(value)
	//value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderHeaderLen)
	//headerLength := sofarpc.ConvertPropertyValueInt16(value)
	value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderContentLen)
	contentLength := sofarpc.ConvertPropertyValueInt(value)

	//class
	className := sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderClassName)

	//RPC Request
	if cmdType == sofarpc.REQUEST || cmdType == sofarpc.REQUEST_ONEWAY {
		value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderTimeout)
		timeout := sofarpc.ConvertPropertyValueInt(value)

		//TODO: reuse cmd
		//sofabuffers := SofaProtocolBuffersByContext(ctx)
		//request := &sofabuffers.BoltEncodeReq
		request := &sofarpc.BoltRequestV2{}
		request.Protocol = protocolCode
		request.Version1 = ver1
		request.CmdType = cmdType
		request.CmdCode = cmdCode
		request.Version = version
		request.ReqID = requestID
		request.Codec = codec
		request.SwitchCode = switchcode

		request.Timeout = timeout

		//request.ClassLen = classLength
		//request.HeaderLen = headerLength
		request.ContentLen = contentLength
		request.RequestClass = className
		request.RequestHeader = headers
		return request, nil
	} else if cmdType == sofarpc.RESPONSE {
		value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderRespStatus)
		responseStatus := sofarpc.ConvertPropertyValueInt16(value)
		value = sofarpc.GetPropertyValue(PropertyHeaders, headers, sofarpc.HeaderRespTimeMills)
		responseTime := sofarpc.ConvertPropertyValueInt64(value)

		//sofabuffers := SofaProtocolBuffersByContext(ctx)
		//response := &sofabuffers.BoltEncodeRsp

		response := &sofarpc.BoltResponseV2{}
		response.Protocol = protocolCode
		response.Version1 = ver1
		response.CmdType = cmdType
		response.CmdCode = cmdCode
		response.Version = version
		response.ReqID = requestID
		response.Codec = codec
		response.SwitchCode = switchcode

		response.ResponseStatus = responseStatus
		//response.ClassLen = classLength
		//response.HeaderLen = headerLength
		response.ContentLen = contentLength
		response.ResponseClass = className
		response.ResponseHeader = headers
		response.ResponseTimeMillis = responseTime
		return response, nil
	}

	return nil, rpc.ErrUnknownType
}

func (b *boltv2conv) MapToFields(ctx context.Context, cmd sofarpc.SofaRpcCmd) (map[string]string, error) {
	switch c := cmd.(type) {
	case *sofarpc.BoltRequestV2:
		return mapReqToFieldsV2(ctx, c)
	case *sofarpc.BoltResponseV2:
		return mapRespToFieldsV2(ctx, c)
	}

	return nil, rpc.ErrUnknownType
}

func mapReqToFieldsV2(ctx context.Context, req *sofarpc.BoltRequestV2) (map[string]string, error) {
	// TODO: map reuse
	//protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	//headers := make(map[string]string, 9+len(req.RequestHeader))
	headers := req.RequestHeader

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(req.Protocol), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion1)] = strconv.FormatUint(uint64(req.Version1), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdType)] = strconv.FormatUint(uint64(req.CmdType), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(req.CmdCode), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion)] = strconv.FormatUint(uint64(req.Version), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(req.ReqID), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCodec)] = strconv.FormatUint(uint64(req.Codec), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderSwitchCode)] = strconv.FormatUint(uint64(req.SwitchCode), 10)

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderTimeout)] = strconv.FormatUint(uint64(req.Timeout), 10)

	// TODO: bypass length header
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassLen)] = strconv.FormatUint(uint64(req.ClassLen), 10)
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderHeaderLen)] = strconv.FormatUint(uint64(req.HeaderLen), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderContentLen)] = strconv.FormatUint(uint64(req.ContentLen), 10)

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassName)] = req.RequestClass

	// for conv
	headers[protocol.MosnHeaderDirection] = protocol.Request

	return headers, nil
}

func mapRespToFieldsV2(ctx context.Context, resp *sofarpc.BoltResponseV2) (map[string]string, error) {
	// TODO: map reuse
	//protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	//headers := make(map[string]string, 12)

	headers := resp.ResponseHeader

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(resp.Protocol), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion1)] = strconv.FormatUint(uint64(resp.Version1), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdType)] = strconv.FormatUint(uint64(resp.CmdType), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(resp.CmdCode), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion)] = strconv.FormatUint(uint64(resp.Version), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(resp.ReqID), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCodec)] = strconv.FormatUint(uint64(resp.Codec), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderSwitchCode)] = strconv.FormatUint(uint64(resp.SwitchCode), 10)

	// TODO: bypass length header
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassLen)] = strconv.FormatUint(uint64(resp.ClassLen), 10)
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderHeaderLen)] = strconv.FormatUint(uint64(resp.HeaderLen), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderContentLen)] = strconv.FormatUint(uint64(resp.ContentLen), 10)

	// FOR RESPONSE,ENCODE RESPONSE STATUS and RESPONSE TIME
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespStatus)] = strconv.FormatUint(uint64(resp.ResponseStatus), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespTimeMills)] = strconv.FormatUint(uint64(resp.ResponseTimeMillis), 10)

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassName)] = resp.ResponseClass

	// for conv
	headers[protocol.MosnHeaderDirection] = protocol.Response

	return headers, nil
}
