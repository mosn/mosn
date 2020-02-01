package tars

import (
	"context"

	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"mosn.io/mosn/pkg/buffer"
	"mosn.io/mosn/pkg/types"
)

func encodeRequest(ctx context.Context, request *Request) (types.IoBuffer, error) {
	os := codec.NewBuffer()
	err := request.cmd.WriteTo(os)
	if err != nil {
		return nil, err
	}
	data := os.ToBytes()
	return buffer.NewIoBufferBytes(data), nil
}
func encodeResponse(ctx context.Context, response *Response) (types.IoBuffer, error) {
	os := codec.NewBuffer()
	err := response.cmd.WriteTo(os)
	if err != nil {
		return nil, err
	}
	data := os.ToBytes()
	return buffer.NewIoBufferBytes(data), nil
}
