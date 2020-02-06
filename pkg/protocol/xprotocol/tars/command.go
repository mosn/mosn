package tars

import (
	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

type Request struct {
	cmd     *requestf.RequestPacket
	rawData []byte         // raw data
	data    types.IoBuffer // wrapper of data
	protocol.CommonHeader
}

// ~ XFrame
func (r *Request) GetRequestId() uint64 {
	return uint64(r.cmd.IRequestId)
}

func (r *Request) SetRequestId(id uint64) {
	r.cmd.IRequestId = int32(id)
}

func (r *Request) IsHeartbeatFrame() bool {
	// un support
	return false
}

func (r *Request) GetStreamType() xprotocol.StreamType {
	return xprotocol.Request
}

func (r *Request) GetHeader() types.HeaderMap {
	return r
}

func (r *Request) GetData() types.IoBuffer {
	return r.data
}

type Response struct {
	cmd     *requestf.ResponsePacket
	rawData []byte         // raw data
	data    types.IoBuffer // wrapper of data
	protocol.CommonHeader
}

// ~ XFrame
func (r *Response) GetRequestId() uint64 {
	return uint64(r.cmd.IRequestId)
}

func (r *Response) SetRequestId(id uint64) {
	r.cmd.IRequestId = int32(id)
}

func (r *Response) IsHeartbeatFrame() bool {
	// un support
	return false
}

func (r *Response) GetStreamType() xprotocol.StreamType {
	return xprotocol.Response
}

func (r *Response) GetHeader() types.HeaderMap {
	return r
}

func (r *Response) GetData() types.IoBuffer {
	return r.data
}
