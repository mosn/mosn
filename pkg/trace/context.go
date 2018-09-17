package trace

import (
	"context"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type contextKey struct{}

var ActiveSpanKey = contextKey{}
var driver = &OpenTracingDriver{}

func SpanFromContext(ctx context.Context) types.Span {
	val := ctx.Value(ActiveSpanKey)
	if sp, ok := val.(types.Span); ok {
		return sp
	}
	return nil
}

func Driver() types.Driver {
	return driver
}
