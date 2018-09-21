package trace

import (
	"context"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type contextKey struct{}

type traceHolder struct {
	enableTracing bool
	driver        types.Driver
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

func SetDriver(driver types.Driver) {
	holder.driver = driver
}

func Driver() types.Driver {
	return holder.driver
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
