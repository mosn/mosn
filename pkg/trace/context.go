package trace

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/types"
)

type contextKey struct{}

type traceHolder struct {
	enableTracing bool
	tracer        types.Tracer
}

var ActiveSpanKey = contextKey{}
var holder = traceHolder{}

func SpanFromContext(ctx context.Context) types.Span {
	val := ctx.Value(ActiveSpanKey)
	if sp, ok := val.(types.Span); ok {
		return sp
	}
	return nil
}

func SetTracer(tracer types.Tracer) {
	holder.tracer = tracer
}

func Tracer() types.Tracer {
	return holder.tracer
}

func EnableTracing() {
	holder.enableTracing = true
}

func DisableTracing() {
	holder.enableTracing = false
}

func IsTracingEnabled() bool {
	return holder.enableTracing
}
