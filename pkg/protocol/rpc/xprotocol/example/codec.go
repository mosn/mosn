package example

import (
	"context"
	"errors"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/xprotocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	codec := &exampleCodec{}
	xprotocol.RegisterEngine("rpc-example", rpc.NewEngine(codec, codec))
}

// dubbo codec
type exampleCodec struct{}

func (c *exampleCodec) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	if cmd, ok := model.(*exampleCmd); ok {
		return buffer.NewIoBufferBytes(cmd.data), nil
	}
	return nil, errors.New("fail to convert to dubbo cmd")
}

func (c *exampleCodec) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	bytes := data.Bytes()
	dataLen := data.Len()

	if dataLen >= ReqDataLen {
		data.Drain(ReqDataLen)

		cmd := &exampleCmd{
			data: bytes[:ReqDataLen],
		}
		return cmd, nil
	}
	return nil, nil
}
