package stream

import (
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"context"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/trace"
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

func (cm *ContextManager) InjectTrace(ctx context.Context, span types.Span) context.Context {
	if span != nil {
		return context.WithValue(ctx, types.ContextKeyTraceId, span.TraceId())
	}
	// generate traceId
	return context.WithValue(ctx, types.ContextKeyTraceId, trace.IdGen().GenerateTraceId())
}

func NewContextManager(base context.Context) *ContextManager {
	return &ContextManager{
		base: base,
	}
}
