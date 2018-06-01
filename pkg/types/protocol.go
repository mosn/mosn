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
	// Synchronously encodes headers/data/trailers to buffer by Stream

	// EncodeHeaders encodes headers to binary buffer
	// return 1. stream id if have one 2. headers bytes
	EncodeHeaders(context context.Context, headers interface{}) (error, IoBuffer)

	// EncodeData encodes data to binary buffer
	EncodeData(context context.Context, data IoBuffer) IoBuffer

	// EncodeTrailers encodes trailers to binary buffer
	EncodeTrailers(context context.Context, trailers map[string]string) IoBuffer

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
	// EncodeHeaders encodes headers to buffer
	// return 1. stream id if have one 2. headers bytes
	EncodeHeaders(context context.Context, headers interface{}) (error, IoBuffer)

	// EncodeData encodes data to buffer
	EncodeData(context context.Context, data IoBuffer) IoBuffer

	// EncodeTrailers encodes trailers to buffer
	EncodeTrailers(context context.Context, trailers map[string]string) IoBuffer
}

// A decoder interface to extend various of protocols
type Decoder interface {
	// Decode decodes binary to a model
	// return 1. bytes decoded 2. decoded cmd
	Decode(context context.Context, data IoBuffer) (int, interface{})
}
