package example

import (
	"context"
	"encoding/binary"

	"mosn.io/mosn/pkg/buffer"
	"mosn.io/mosn/pkg/types"
)

func decodeRequest(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	bytesLen := data.Len()
	bytes := data.Bytes()

	// 1. least bytes to decode header is RequestHeaderLen
	if bytesLen < RequestHeaderLen {
		return
	}

	// 2. least bytes to decode whole frame
	payloadLen := binary.BigEndian.Uint32(bytes[7:])
	frameLen := RequestHeaderLen + int(payloadLen)
	if bytesLen < frameLen {
		return
	}
	data.Drain(frameLen)

	// 3. decode header
	request := &Request{
		Type:       bytes[1],
		RequestId:  binary.BigEndian.Uint32(bytes[RequestIdIndex:]),
		PayloadLen: payloadLen,
	}

	//4. copy data for io multiplexing
	request.Payload = *buffer.GetBytesByContext(ctx, int(payloadLen))
	copy(request.Payload, bytes[RequestHeaderLen:])

	return request, err
}

func decodeResponse(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	bytesLen := data.Len()
	bytes := data.Bytes()

	// 1. least bytes to decode header is ResponseHeaderLen
	if bytesLen < ResponseHeaderLen {
		return
	}

	// 2. least bytes to decode whole frame
	payloadLen := binary.BigEndian.Uint32(bytes[9:])
	frameLen := ResponseHeaderLen + int(payloadLen)
	if bytesLen < frameLen {
		return
	}
	data.Drain(frameLen)

	// 3. decode header
	response := &Response{
		Request: Request{
			Type:       bytes[1],
			RequestId:  binary.BigEndian.Uint32(bytes[RequestIdIndex:]),
			PayloadLen: payloadLen,
		},
		Status: binary.BigEndian.Uint16(bytes[7:]),
	}

	//4. copy data for io multiplexing
	response.Payload = *buffer.GetBytesByContext(ctx, int(payloadLen))
	copy(response.Payload, bytes[ResponseHeaderLen:])

	return response, nil
}
