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

// Protocols' facade used by Stream
type Protocols interface {
	//A encoder interface to extend various of protocols
	Encoder
	// Decode decodes data to headers-data-trailers by Stream
	// Stream register a DecodeFilter to receive decode event
	Decode(context context.Context, data IoBuffer, filter DecodeFilter)
}

// Filter used by Stream to receive decode events
type DecodeFilter interface {
	// Called on headers decoded
	OnDecodeHeader(streamId string, headers map[string]string) FilterStatus

	// Called on data decoded
	OnDecodeData(streamId string, data IoBuffer) FilterStatus

	// Called on trailers decoded
	OnDecodeTrailer(streamId string, trailers map[string]string) FilterStatus

	// Called when error occurs
	// When error occurring, filter status = stop
	OnDecodeError(err error, headers map[string]string)
}

// A encoder interface to extend various of protocols
type Encoder interface {
	// AppendHeaders encodes headers to buffer
	// return 1. headers bytes 2. stream id if have one
	EncodeHeaders(context context.Context, headers interface{}) (IoBuffer, error)

	// AppendData encodes data to buffer
	EncodeData(context context.Context, data IoBuffer) IoBuffer

	// AppendTrailers encodes trailers to buffer
	EncodeTrailers(context context.Context, trailers map[string]string) IoBuffer
}

// A decoder interface to extend various of protocols
type Decoder interface {
	// Decode decodes binary to a model
	// return 1. bytes decoded 2. decoded cmd
	Decode(context context.Context, data IoBuffer) (int, interface{})
}
