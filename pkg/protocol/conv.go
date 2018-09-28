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

package protocol

import (
	"context"
	"errors"

	"github.com/alipay/sofa-mosn/pkg/types"
)

var (
	protoConvFactory = make(map[types.Protocol]map[types.Protocol]ProtocolConv)

	ErrNotFound = errors.New("no convert function found for given protocol pair")
)

func init() {
	identity := new(identity)

	RegisterConv(HTTP1, HTTP2, identity)
	RegisterConv(HTTP2, HTTP1, identity)
}

// ProtocolConv extract common methods for protocol conversion(header, data, trailer)
type ProtocolConv interface {

	// ConvHeader convert header part represents in `types.HeaderMap`
	ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error)

	// ConvData convert data part represents in `types.IoBuffer`
	ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error)

	// ConvTrailer convert trailer part represents in `types.HeaderMap`
	ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error)
}

// RegisterConv register concrete protocol convert function for specified source protocol and destination protocol
func RegisterConv(src, dst types.Protocol, f ProtocolConv) {
	if _, subOk := protoConvFactory[src]; !subOk {
		protoConvFactory[src] = make(map[types.Protocol]ProtocolConv)
	}

	protoConvFactory[src][dst] = f
}

// ConvertHeader convert header from source protocol format to destination protocol format
func ConvertHeader(ctx context.Context, src, dst types.Protocol, srcHeader types.HeaderMap) (types.HeaderMap, error) {
	if sub, subOk := protoConvFactory[src]; subOk {
		if f, ok := sub[dst]; ok {
			return f.ConvHeader(ctx, srcHeader)
		}
	}
	return nil, ErrNotFound
}

// ConvertData convert data from source protocol format to destination protocol format
func ConvertData(ctx context.Context, src, dst types.Protocol, srcData types.IoBuffer) (types.IoBuffer, error) {
	if sub, subOk := protoConvFactory[src]; subOk {
		if f, ok := sub[dst]; ok {
			return f.ConvData(ctx, srcData)
		}
	}
	return nil, ErrNotFound
}

// ConvertTrailer convert trailer from source protocol format to destination protocol format
func ConvertTrailer(ctx context.Context, src, dst types.Protocol, srcTrailer types.HeaderMap) (types.HeaderMap, error) {
	if sub, subOk := protoConvFactory[src]; subOk {
		if f, ok := sub[dst]; ok {
			return f.ConvTrailer(ctx, srcTrailer)
		}
	}
	return nil, ErrNotFound
}

// identity conversion, keeps no change
type identity struct{}

func (c *identity) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}

func (c *identity) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *identity) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}
