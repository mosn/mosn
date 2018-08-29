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

package types

import (
	"context"
)

// 	 The bunch of interfaces are structure skeleton to build a extensible protocol engine.
//
//   In mosn, we have 4 layers to build a mesh, protocol is the core layer to do protocol related encode/decode.
//	 -----------------------
//   |        PROXY          |
//    -----------------------
//   |       STREAMING       |
//    -----------------------
//   |        PROTOCOL       |
//    -----------------------
//   |         NET/IO        |
//    -----------------------
//
//	 Stream layer leverages protocol's ability to do binary-model conversation. In detail, Stream uses Protocols's encode/decode facade method and DecodeFilter to receive decode event call.
//

// TODO: support error case: add error as return value in EncodeX method; add OnError(error) in DecodeFilter @wugou

type Protocol string

// Protocols is a protocols' facade used by Stream
type Protocols interface {
	// Encoder is a encoder interface to extend various of protocols
	Encoder
	// Decode decodes data to headers-data-trailers by Stream
	// Stream register a DecodeFilter to receive decode event
	Decode(context context.Context, data IoBuffer, filter DecodeFilter)
}

// DecodeFilter is a filter used by Stream to receive decode events
type DecodeFilter interface {
	// OnDecodeHeader is called on headers decoded
	OnDecodeHeader(streamID string, headers map[string]string) FilterStatus

	// OnDecodeData is called on data decoded
	OnDecodeData(streamID string, data IoBuffer) FilterStatus

	// OnDecodeTrailer is called on trailers decoded
	OnDecodeTrailer(streamID string, trailers map[string]string) FilterStatus

	// OnDecodeError is called when error occurs
	// When error occurring, filter status = stop
	OnDecodeError(err error, headers map[string]string)
}

// Encoder is a encoder interface to extend various of protocols
type Encoder interface {
	// EncodeHeaders encodes the headers based on it's protocol
	EncodeHeaders(context context.Context, headers interface{}) (IoBuffer, error)

	// EncodeData encodes the data based on it's protocol
	EncodeData(context context.Context, data IoBuffer) IoBuffer

	// EncodeTrailers encodes the trailers based on it's protocol
	EncodeTrailers(context context.Context, trailers map[string]string) IoBuffer
}

// Decoder is a decoder interface to extend various of protocols
type Decoder interface {
	// Decode decodes binary to a model
	// return 1. bytes decoded 2. decoded cmd
	Decode(context context.Context, data IoBuffer) (interface{}, error)
}

// SubProtocol Name
type SubProtocol string

// Multiplexing Accesslog Rate limit Curcuit Breakers
type Multiplexing interface {
	SplitFrame(data []byte) [][]byte
	GetStreamID(data []byte) string
	SetStreamID(data []byte, streamID string) []byte
}

// Tracing base on Multiplexing
type Tracing interface {
	Multiplexing
	GetServiceName(data []byte) string
	GetMethodName(data []byte) string
}

// RequestRouting RequestAccessControl RequesstFaultInjection base on Multiplexing
type RequestRouting interface {
	Multiplexing
	GetMetas(data []byte) map[string]string
}

// ProtocolConvertor change protocol base on Multiplexing
type ProtocolConvertor interface {
	Multiplexing
	Convert(data []byte) (map[string]string, []byte)
}
