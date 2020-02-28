package example

import (
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

type Request struct {
	Type       byte
	RequestId  uint32
	PayloadLen uint32
	Payload    []byte
	Content    types.IoBuffer
}

func (r *Request) GetStreamType() xprotocol.StreamType {
	return xprotocol.Request
}

func (r *Request) GetRequestId() uint64 {
	return uint64(r.RequestId)
}

func (r *Request) SetRequestId(id uint64) {
	r.RequestId = uint32(id)
}

type Response struct {
	Request
	Status uint16
}

func (r *Response) GetStreamType() xprotocol.StreamType {
	return xprotocol.Response
}

func (r *Response) GetRequestId() uint64 {
	return uint64(r.RequestId)
}

func (r *Response) SetRequestId(id uint64) {
	r.RequestId = uint32(id)
}


type MessageCommand struct {
	Request

	// TODO: pb deserialize target, extract from Payload
	Message interface{}
}

type MessageAckCommand struct {
	Response

	// TODO: pb deserialize target, extract from Payload
	Message interface{}
}
