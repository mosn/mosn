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

	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

var (
	protoConvFactory = make(map[api.Protocol]map[api.Protocol]ProtocolConv)

	ErrNotFound = errors.New("no convert function found for given protocol pair")

	ErrHeaderDirection = errors.New("no header direction info")

	// Only for internal usage. This protocol is represent for those non-protocol-related data structure, like CommonHeader.
	//
	// For those protocols which have shared properties, like http1 and http2, we prefer to use an common way
	// instead of writing specific impl-aware convert functions.So we can avoid too many convert functions.
	// If we have N protocols, and each need to be able to convert from/to each other, then we need N*(N-1)*2
	// convert functions(N protocols, each need to consider N-1 targets with from/to).
	// But if we use 'common' protocol as bridge, we can decrease the function count to N*2.(N protocols, each one
	// define from/to with 'common' protocol).
	//
	// More complicated, supposed that we have 4 protocols here, http1/http2/sofarpc/dubbo. All of them
	// have their own header impl, and some properties are necessary while convert from/to CommonHeader(e.g.
	// 'protocol' for sofarpc).We abstract the protocol-properties just as 'a','b','c' and so on.
	// http1 <-> 'a', 'b'
	// http2 <-> 'a', 'b'
	// sofarpc <-> 'c', 'd', 'e'
	// dubbo <-> 'c', 'd', 'f'
	//
	// The common way not works if we want to convert from sofarpc to dubbo. We get ['c', 'd', 'e'] properties while
	// convert from 'sofarpc' to 'common', and need ['c', 'd', 'f'] for converting from 'common' to 'dubbo'. In this case,
	// write specific impl-aware convert functions for src and dst protocol.
	common api.Protocol = "common"
)

// ProtocolConv extract common methods for protocol conversion(header, data, trailer)
type ProtocolConv interface {
	// ConvHeader convert header part represents in `api.HeaderMap`
	ConvHeader(ctx context.Context, headerMap api.HeaderMap) (api.HeaderMap, error)

	// ConvData convert data part represents in `buffer.IoBuffer`
	ConvData(ctx context.Context, buffer buffer.IoBuffer) (buffer.IoBuffer, error)

	// ConvTrailer convert trailer part represents in `api.HeaderMap`
	ConvTrailer(ctx context.Context, headerMap api.HeaderMap) (api.HeaderMap, error)
}

// RegisterConv register concrete protocol convert function for specified source protocol and destination protocol
func RegisterConv(src, dst api.Protocol, f ProtocolConv) {
	if _, subOk := protoConvFactory[src]; !subOk {
		protoConvFactory[src] = make(map[api.Protocol]ProtocolConv)
	}

	protoConvFactory[src][dst] = f
}

// RegisterCommonConv register concrete protocol convert function for specified protocol and common representation.
// e.g. SofaRpcCmd <-> CommonHeader, which both implements the types.HeaderMap interface
func RegisterCommonConv(protocol api.Protocol, from, to ProtocolConv) {
	RegisterConv(common, protocol, from)
	RegisterConv(protocol, common, to)
}

// ConvertHeader convert header from source protocol format to destination protocol format
func ConvertHeader(ctx context.Context, src, dst api.Protocol, srcHeader api.HeaderMap) (api.HeaderMap, error) {
	// 1. try direct path
	if sub, subOk := protoConvFactory[src]; subOk {
		if f, ok := sub[dst]; ok {
			return f.ConvHeader(ctx, srcHeader)
		}
	}

	// 2. try common path
	if src != common && dst != common {
		if src2Common, serr := ConvertHeader(ctx, src, common, srcHeader); serr == nil {
			if common2dst, derr := ConvertHeader(ctx, common, dst, src2Common); derr == nil {
				return common2dst, derr
			} else {
				return nil, derr
			}
		} else {
			return nil, serr
		}
	}
	return nil, ErrNotFound
}

// ConvertData convert data from source protocol format to destination protocol format
func ConvertData(ctx context.Context, src, dst api.Protocol, srcData buffer.IoBuffer) (buffer.IoBuffer, error) {
	// 1. try direct path
	if sub, subOk := protoConvFactory[src]; subOk {
		if f, ok := sub[dst]; ok {
			return f.ConvData(ctx, srcData)
		}
	}

	// 2. try common path
	if src != common && dst != common {
		if src2Common, serr := ConvertData(ctx, src, common, srcData); serr == nil {
			if common2dst, derr := ConvertData(ctx, common, dst, src2Common); derr == nil {
				return common2dst, derr
			}
		}
	}

	return nil, ErrNotFound
}

// ConvertTrailer convert trailer from source protocol format to destination protocol format
func ConvertTrailer(ctx context.Context, src, dst api.Protocol, srcTrailer api.HeaderMap) (api.HeaderMap, error) {
	// 1. try direct path
	if sub, subOk := protoConvFactory[src]; subOk {
		if f, ok := sub[dst]; ok {
			return f.ConvTrailer(ctx, srcTrailer)
		}
	}

	// 2. try common path
	if src != common && dst != common {
		if src2Common, serr := ConvertTrailer(ctx, src, common, srcTrailer); serr == nil {
			if common2dst, derr := ConvertTrailer(ctx, common, dst, src2Common); derr == nil {
				return common2dst, derr
			}
		}
	}

	return nil, ErrNotFound
}
