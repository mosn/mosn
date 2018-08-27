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
	return newRPCExample()
}

type rpcExample struct{}

func newRPCExample() types.Multiplexing {
	return &rpcExample{}
}

func (re *rpcExample) SplitFrame(data []byte) [][]byte {
	return nil
}

func (re *rpcExample) GetStreamID(data []byte) string {
	return ""
}

func (re *rpcExample) SetStreamID(data []byte, streamID string) []byte {
	return nil
}
