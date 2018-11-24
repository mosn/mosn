package mhttp2

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/buffer"
)

var ins = Mhttp2BufferCtx{}

type Mhttp2BufferCtx struct{}

func (ctx Mhttp2BufferCtx) Name() int {
	return buffer.MHTTP2
}

func (ctx Mhttp2BufferCtx) New() interface{} {
	buffer := new(Mhttp2Buffers)
	return buffer
}

func (ctx Mhttp2BufferCtx) Reset(i interface{}) {
}

type Mhttp2Buffers struct {
}

func Mhttp2BuffersByContext(ctx context.Context) *Mhttp2Buffers {
	poolCtx := buffer.PoolContext(ctx)
	return poolCtx.Find(ins, nil).(*Mhttp2Buffers)
}
