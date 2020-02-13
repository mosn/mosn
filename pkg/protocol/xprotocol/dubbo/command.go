package dubbo

import (
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

type Header struct {
	Magic   []byte
	Flag    byte
	Status  byte
	Id      uint64
	DataLen uint32

	Event           int // 1 mean ping
	TwoWay          int // 1 mean req & resp pair
	Direction       int // 1 mean req
	SerializationId int // 2 mean hessian
	protocol.CommonHeader
}

type Frame struct {
	Header
	rawData []byte // raw data
	payload []byte // raw payload

	data    types.IoBuffer // wrapper of data
	content types.IoBuffer // wrapper of payload
}

// ~ XFrame
func (r *Frame) GetRequestId() uint64 {
	return r.Header.Id
}

func (r *Frame) SetRequestId(id uint64) {
	r.Header.Id = id
}

func (r *Frame) IsHeartbeatFrame() bool {
	return r.Header.Event != 0
}

func (r *Frame) GetStreamType() xprotocol.StreamType {
	switch r.Direction {
	case EventRequest:
		return xprotocol.Request
	case EventResponse:
		return xprotocol.Response
	default:
		return xprotocol.Request
	}
}

func (r *Frame) GetHeader() types.HeaderMap {
	return r
}

func (r *Frame) GetData() types.IoBuffer {
	return r.content
}
