package stream

import (
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"context"
	"github.com/alipay/sofa-mosn/pkg/trace"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// contextManager
type ContextManager struct {
	base context.Context
	curr context.Context
}

func (cm *ContextManager) Get() context.Context {
	return cm.curr
}

func (cm *ContextManager) Next() {
	// buffer context
	cm.curr = buffer.NewBufferPoolContext(cm.base)
}

func (cm *ContextManager) NextWithTrace() {
	// buffer context
	ctx := buffer.NewBufferPoolContext(cm.base)

	// trace context with default traceId, should be replaced if request already contained one.
	cm.curr = context.WithValue(ctx, types.ContextKeyTraceId, trace.IdGen().GenerateTraceId())
}

func NewContextManager(base context.Context) *ContextManager {
	return &ContextManager{
		base: base,
	}
}
