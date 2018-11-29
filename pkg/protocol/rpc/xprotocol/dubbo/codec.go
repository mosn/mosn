package dubbo

import (
	"context"
	"errors"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/xprotocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	codec := &dubboCodec{}
	xprotocol.RegisterEngine("dubbo", rpc.NewEngine(codec, codec))
}

// dubbo codec
type dubboCodec struct{}

func (c *dubboCodec) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	if cmd, ok := model.(*dubboCmd); ok {
		return buffer.NewIoBufferBytes(cmd.data), nil
	}
	return nil, errors.New("fail to convert to dubbo cmd")
}

func (c *dubboCodec) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	bytes := data.Bytes()
	dataLen := data.Len()

	// read frame length
	frameLen := getDubboLen(bytes)
	if frameLen > 0 && dataLen >= frameLen {
		data.Drain(frameLen)

		cmd := &dubboCmd{
			data: bytes[:frameLen],
		}
		return cmd, nil
	}
	return nil, nil
}
