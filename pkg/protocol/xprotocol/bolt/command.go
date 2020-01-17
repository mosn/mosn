package bolt

import (
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

type RequestHeader struct {
	Protocol   byte // meta fields
	CmdType    byte
	CmdCode    uint16
	Version    byte
	RequestId  uint32
	Codec      byte
	Timeout    int32
	ClassLen   uint16
	HeaderLen  uint16
	ContentLen uint32

	Class string // payload fields
	header
}

// ~ HeaderMap
func (h *RequestHeader) Clone() types.HeaderMap {
	clone := &RequestHeader{}
	*clone = *h

	// deep copy
	clone.header = *h.header.Clone()

	return clone
}

// Request is the cmd struct of bolt v1 request
type Request struct {
	RequestHeader

	rawData    *[]byte // raw data
	rawMeta    []byte  // sub slice of raw data, start from protocol code, ends to content length
	rawClass   []byte  // sub slice of raw data, class bytes
	rawHeader  []byte  // sub slice of raw data, header bytes
	rawContent []byte  // sub slice of raw data, content bytes

	Data    types.IoBuffer // wrapper of raw data
	Content types.IoBuffer // wrapper of raw content
}

// ~ XFrame
func (r *Request) GetRequestId() uint64 {
	return uint64(r.RequestHeader.RequestId)
}

func (r *Request) SetRequestId(id uint64) {
	r.RequestHeader.RequestId = uint32(id)
}

func (r *Request) IsHeartbeatFrame() bool {
	return r.RequestHeader.CmdCode == CmdCodeHeartbeat
}

func (r *Request) GetStreamType() xprotocol.StreamType {
	switch r.RequestHeader.CmdType {
	case CmdTypeRequest:
		return xprotocol.Request
	case CmdTypeRequestOneway:
		return xprotocol.RequestOneWay
	default:
		return xprotocol.Request
	}
}

func (r *Request) GetHeader() types.HeaderMap {
	return r
}

func (r *Request) GetData() types.IoBuffer {
	return r.Content
}

type ResponseHeader struct {
	Protocol       byte // meta fields
	CmdType        byte
	CmdCode        uint16
	Version        byte
	RequestId      uint32
	Codec          byte
	ResponseStatus uint16
	ClassLen       uint16
	HeaderLen      uint16
	ContentLen     uint32

	Class string // payload fields
	header
}

// ~ HeaderMap
func (h *ResponseHeader) Clone() types.HeaderMap {
	clone := &ResponseHeader{}
	*clone = *h

	// deep copy
	clone.header = *h.header.Clone()

	return clone
}

// Response is the cmd struct of bolt v1 response
type Response struct {
	ResponseHeader

	rawData    *[]byte // raw data
	rawMeta    []byte  // sub slice of raw data, start from protocol code, ends to content length
	rawClass   []byte  // sub slice of raw data, class bytes
	rawHeader  []byte  // sub slice of raw data, header bytes
	rawContent []byte  // sub slice of raw data, content bytes

	Data    types.IoBuffer // wrapper of raw data
	Content types.IoBuffer // wrapper of raw content
}

// ~ XFrame
func (r *Response) GetRequestId() uint64 {
	return uint64(r.ResponseHeader.RequestId)
}

func (r *Response) SetRequestId(id uint64) {
	r.ResponseHeader.RequestId = uint32(id)
}

func (r *Response) IsHeartbeatFrame() bool {
	return r.ResponseHeader.CmdCode == CmdCodeHeartbeat
}

func (r *Response) GetStreamType() xprotocol.StreamType {
	return xprotocol.Response
}

func (r *Response) GetHeader() types.HeaderMap {
	return r
}

func (r *Response) GetData() types.IoBuffer {
	return r.Content
}
