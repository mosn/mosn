package proxy

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/buffer"
)

type shadowBufferCtx struct{}

func (ctx shadowBufferCtx) Name() int {
	return buffer.Shadow
}

func (ctx shadowBufferCtx) New() interface{} {
	return new(shadowBuffers)
}

func (ctx shadowBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*shadowBuffers)
	*buf = shadowBuffers{}
}

type shadowBuffers struct {
	stream  shadowDownstream
	request shadowUpstreamRequest
}

func shadowBuffersByContext(ctx context.Context) *shadowBuffers {
	poolCtx := buffer.PoolContext(ctx)
	return poolCtx.Find(shadowBufferCtx{}, nil).(*shadowBuffers)
}
