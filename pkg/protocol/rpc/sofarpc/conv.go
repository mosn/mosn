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

	"reflect"
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var (
	sofaConvFactory = make(map[byte]SofaConv)
)

func init() {
	http2sofa := new(http2sofa)
	sofa2http := new(sofa2http)

	protocol.RegisterConv(protocol.HTTP1, protocol.SofaRPC, http2sofa)
	protocol.RegisterConv(protocol.HTTP2, protocol.SofaRPC, http2sofa)

	protocol.RegisterConv(protocol.SofaRPC, protocol.HTTP1, sofa2http)
	protocol.RegisterConv(protocol.SofaRPC, protocol.HTTP2, sofa2http)
}

// SofaConv extract common methods for protocol conversion between sofarpc protocols(bolt/boltv2/tr) and others
// This is special because the 'SofaRpc' directive actually contains multi sub protocols.Listener specified with 'SofaRpc' downstream
// protocol could handle different sub protocols in different connections. So the real sub protocol could only be determined
// at runtime, using protocol code recognition. And that's the exact job done by SofaConv.
type SofaConv interface {
	// MapToCmd maps given header map(must contains necessary sofarpc protocol fields) to corresponding sofarpc command struct
	MapToCmd(ctx context.Context, headerMap map[string]string) (SofaRpcCmd, error)

	// MapToFields maps given sofarpc command struct to corresponding key-value header map(contains necessary sofarpc protocol fields)
	MapToFields(ctx context.Context, cmd SofaRpcCmd) (map[string]string, error)
}

// RegisterConv for sub protocol registry
func RegisterConv(protocol byte, conv SofaConv) {
	sofaConvFactory[protocol] = conv
}

// MapToCmd  expect src header data type as `protocol.CommonHeader`
func MapToCmd(ctx context.Context, headerMap map[string]string) (SofaRpcCmd, error) {

	// TODO: temporary use bolt.HeaderProtocolCode, need to use common definition
	if proto, exist := headerMap[SofaPropertyHeader(HeaderProtocolCode)]; exist {
		protoValue := ConvertPropertyValueUint8(proto)
		protocolCode := protoValue

		if conv, ok := sofaConvFactory[protocolCode]; ok {
			return conv.MapToCmd(ctx, headerMap)
		}
		return nil, rpc.ErrUnrecognizedCode
	}
	return nil, rpc.ErrNoProtocolCode
}

// MapToFields expect src header data type as `ProtoBasicCmd`
func MapToFields(ctx context.Context, cmd SofaRpcCmd) (map[string]string, error) {
	protocol := cmd.ProtocolCode()

	if conv, ok := sofaConvFactory[protocol]; ok {
		return conv.MapToFields(ctx, cmd)
	}
	return nil, rpc.ErrUnrecognizedCode
}

// http/x -> sofarpc converter
type http2sofa struct{}

func (c *http2sofa) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	if header, ok := headerMap.(protocol.CommonHeader); ok {
		return MapToCmd(ctx, header)
	}
	return nil, errors.New("header type not supported")
}

func (c *http2sofa) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *http2sofa) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}

// sofarpc -> http/x converter
type sofa2http struct{}

func (c *sofa2http) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	if cmd, ok := headerMap.(SofaRpcCmd); ok {
		header, err := MapToFields(ctx, cmd)
		return protocol.CommonHeader(header), err
	}
	return nil, errors.New("header type not supported")
}

func (c *sofa2http) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *sofa2http) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}

// ~~ convert utility ~~
func SofaPropertyHeader(name string) string {
	return name
}

func GetPropertyValue(properHeaders map[string]reflect.Kind, headers map[string]string, name string) string {
	propertyHeaderName := SofaPropertyHeader(name)

	if value, ok := headers[propertyHeaderName]; ok {
		delete(headers, propertyHeaderName)

		return value
	}

	if value, ok := headers[name]; ok {

		return value
	}

	return ""
}

func ConvertPropertyValueUint8(strValue string) byte {
	value, _ := strconv.ParseUint(strValue, 10, 8)
	return byte(value)
}

func ConvertPropertyValueUint16(strValue string) uint16 {
	value, _ := strconv.ParseUint(strValue, 10, 16)
	return uint16(value)
}

func ConvertPropertyValueUint32(strValue string) uint32 {
	value, _ := strconv.ParseUint(strValue, 10, 32)
	return uint32(value)
}

func ConvertPropertyValueUint64(strValue string) uint64 {
	value, _ := strconv.ParseUint(strValue, 10, 64)
	return uint64(value)
}

func ConvertPropertyValueInt8(strValue string) int8 {
	value, _ := strconv.ParseInt(strValue, 10, 8)
	return int8(value)
}

func ConvertPropertyValueInt16(strValue string) int16 {
	value, _ := strconv.ParseInt(strValue, 10, 16)
	return int16(value)
}

func ConvertPropertyValueInt(strValue string) int {
	value, _ := strconv.ParseInt(strValue, 10, 32)
	return int(value)
}

func ConvertPropertyValueInt64(strValue string) int64 {
	value, _ := strconv.ParseInt(strValue, 10, 64)
	return int64(value)
}
