package xprotocol

import (
	"context"
	"mosn.io/mosn/pkg/buffer"
)

func init() {
	buffer.RegisterBuffer(&ins)
}

var ins = xStreamBufferCtx{}

type xStreamBufferCtx struct {
	buffer.TempBufferCtx
}

func (ctx xStreamBufferCtx) New() interface{} {
	return new(streamBuffers)
}

func (ctx xStreamBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*streamBuffers)
	*buf = streamBuffers{}
}

type streamBuffers struct {
	clientStream xStream
	serverStream xStream
}

func streamBuffersByContext(context context.Context) *streamBuffers {
	ctx := buffer.PoolContext(context)
	return ctx.Find(&ins, nil).(*streamBuffers)
}
