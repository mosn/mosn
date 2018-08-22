package subprotocol

import (
	"github.com/alipay/sofa-mosn/pkg/stream/xprotocol"
	"context"
	"github.com/alipay/sofa-mosn/pkg/types"
)
func init(){
	xprotocol.Register("rpcExample",&rpcExampleFactory{})
}

type rpcExampleFactory struct{}

func (ref *rpcExampleFactory) CreateSubProtocolCodec(context context.Context) types.Multiplexing{
	return newRpcExample()
}

type rpcExample struct{}

func newRpcExample() types.Multiplexing{
	return &rpcExample{}
}

func (re *rpcExample) SplitRequest(data []byte) [][]byte{
	return nil
}

func (re *rpcExample) GetStreamId(data []byte) string{
	return ""
}

func (re *rpcExample) SetStreamId(data []byte,streamId string) []byte{
	return nil
}
