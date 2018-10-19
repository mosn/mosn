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

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var (
	sofaConvFactory = make(map[byte]SofaConv)

	//TODO universe definition
	ErrUnsupportedProtocol = errors.New(types.UnSupportedProCode)
	ErrNoProtocol          = errors.New(NoProCodeInHeader)
)

func init() {
	http2sofa := new(http2sofa)
	sofa2http := new(sofa2http)

	protocol.RegisterConv(protocol.HTTP1, SofaRPC, http2sofa)
	protocol.RegisterConv(protocol.HTTP2, SofaRPC, http2sofa)

	protocol.RegisterConv(SofaRPC, protocol.HTTP1, sofa2http)
	protocol.RegisterConv(SofaRPC, protocol.HTTP2, sofa2http)
}

// SofaConv extract common methods for protocol conversion between sofarpc protocols(bolt/boltv2/tr) and others
type SofaConv interface {

	// MapToCmd maps given header map(must contains necessary sofarpc protocol fields) to corresponding sofarpc command struct
	MapToCmd(ctx context.Context, headerMap map[string]string) (ProtoBasicCmd, error)

	// MapToFields maps given sofarpc command struct to corresponding key-value header map(contains necessary sofarpc protocol fields)
	MapToFields(ctx context.Context, cmd ProtoBasicCmd) (map[string]string, error)
}

// RegisterConv for sub protocol registry
func RegisterConv(protocol byte, conv SofaConv) {
	sofaConvFactory[protocol] = conv
}

// MapToCmd  expect src header data type as `protocol.CommonHeader`
func MapToCmd(ctx context.Context, headerMap map[string]string) (ProtoBasicCmd, error) {

	if proto, exist := headerMap[SofaPropertyHeader(HeaderProtocolCode)]; exist {
		protoValue := ConvertPropertyValueUint8(proto)
		protocolCode := protoValue

		if conv, ok := sofaConvFactory[protocolCode]; ok {
			//TODO:  delete this copy
			// proxy downstream maybe retry, use the headerMap many times
			// if proxy downstream can keep a encoded header(iobuf)
			// map to cmd will be called only once, so we can modify the map without a copy
			headerCopy := make(map[string]string, len(headerMap))
			for k, v := range headerMap {
				headerCopy[k] = v
			}
			return conv.MapToCmd(ctx, headerCopy)
		}
		return nil, ErrUnsupportedProtocol
	}
	return nil, ErrNoProtocol
}

// MapToFields expect src header data type as `ProtoBasicCmd`
func MapToFields(ctx context.Context, cmd ProtoBasicCmd) (map[string]string, error) {
	protocol := cmd.GetProtocol()

	if conv, ok := sofaConvFactory[protocol]; ok {
		return conv.MapToFields(ctx, cmd)
	}
	return nil, ErrUnsupportedProtocol
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
	if cmd, ok := headerMap.(ProtoBasicCmd); ok {
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
