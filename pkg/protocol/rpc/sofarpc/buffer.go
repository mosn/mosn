package sofarpc

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/buffer"
)

type SofaProtocolBufferCtx struct{}

func (ctx SofaProtocolBufferCtx) Name() int {
	return buffer.SofaProtocol
}

func (ctx SofaProtocolBufferCtx) New() interface{} {
	buffer := new(SofaProtocolBuffers)
	return buffer
}

func (ctx SofaProtocolBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*SofaProtocolBuffers)
	buf.BoltReq = BoltRequest{}
	buf.BoltRsp = BoltResponse{}
	buf.BoltEncodeReq = BoltRequest{}
	buf.BoltEncodeRsp = BoltResponse{}
}

type SofaProtocolBuffers struct {
	BoltReq       BoltRequest
	BoltRsp       BoltResponse
	BoltEncodeReq BoltRequest
	BoltEncodeRsp BoltResponse
}

func SofaProtocolBuffersByContext(ctx context.Context) *SofaProtocolBuffers {
	poolCtx := buffer.PoolContext(ctx)
	return poolCtx.Find(SofaProtocolBufferCtx{}, nil).(*SofaProtocolBuffers)
}
