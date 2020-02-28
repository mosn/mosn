package tars

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"mosn.io/mosn/pkg/buffer"
	"mosn.io/mosn/pkg/types"
)

func encodeRequest(ctx context.Context, request *Request) (types.IoBuffer, error) {
	sbuf := bytes.NewBuffer(nil)
	sbuf.Write(make([]byte, 4))
	os := codec.NewBuffer()
	err := request.cmd.WriteTo(os)
	if err != nil {
		return nil, err
	}
	bs := os.ToBytes()
	sbuf.Write(bs)
	len := sbuf.Len()
	binary.BigEndian.PutUint32(sbuf.Bytes(), uint32(len))
	data := sbuf.Bytes()
	return buffer.NewIoBufferBytes(data), nil
}
func encodeResponse(ctx context.Context, response *Response) (types.IoBuffer, error) {
	os := codec.NewBuffer()
	response.cmd.WriteTo(os)
	bs := os.ToBytes()
	sbuf := bytes.NewBuffer(nil)
	sbuf.Write(make([]byte, 4))
	sbuf.Write(bs)
	len := sbuf.Len()
	binary.BigEndian.PutUint32(sbuf.Bytes(), uint32(len))
	data := sbuf.Bytes()
	return buffer.NewIoBufferBytes(data), nil
}
