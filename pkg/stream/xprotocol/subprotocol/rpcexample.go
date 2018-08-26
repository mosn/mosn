package subprotocol

import (
	"context"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	Register("rpc-example", &rpcExampleFactory{})
}

type rpcExampleFactory struct{}

func (ref *rpcExampleFactory) CreateSubProtocolCodec(context context.Context) types.Multiplexing {
	return newRpcExample()
}

type rpcExample struct{}

func newRpcExample() types.Multiplexing {
	return &rpcExample{}
}

func (re *rpcExample) SplitFrame(data []byte) [][]byte {
	return nil
}

func (re *rpcExample) GetStreamId(data []byte) string {
	return ""
}

func (re *rpcExample) SetStreamId(data []byte, streamId string) []byte {
	return nil
}
