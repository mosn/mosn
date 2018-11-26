package sofarpc

import (
	"context"
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/trace"
	"github.com/alipay/sofa-mosn/pkg/types"
	"testing"
)

func TestProtocols_BuildSpan_SpanIsNil(t *testing.T) {
	span := DefaultProtocols().BuildSpan(context.Background())

	if span != nil {
		t.Error("Span is not nil")
	}
}

func TestProtocols_BuildSpan_SpanIsNotNil(t *testing.T) {
	trace.CreateInstance()
	trace.SetTracer(trace.SofaTracerInstance)
	poolCtx := buffer.PoolContext(context.Background())
	ctx := context.WithValue(context.Background(), types.ContextKeyBufferPoolCtx, poolCtx)
	ctx = context.WithValue(ctx, types.ContextKeyListenerType, v2.INGRESS)
	buffers := SofaProtocolBuffersByContext(ctx)
	req := &buffers.BoltReq
	req.CmdCode = CMD_CODE_REQUEST
	span := DefaultProtocols().BuildSpan(ctx)

	if span == nil {
		t.Error("Span is not nil")
	}
}
