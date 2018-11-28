package http

import (
	"time"

	"github.com/alipay/sofa-mosn/pkg/trace"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/valyala/fasthttp"
)

var spanBuilder = &SpanBuilder{}

type SpanBuilder struct {
}

func (spanBuilder *SpanBuilder) BuildSpan(args ...interface{}) types.Span {
	if len(args) == 0 {
		return nil
	}

	if requestCtx, ok := args[0].(fasthttp.RequestCtx); ok {
		span := trace.Tracer().Start(time.Now())
		span.SetTag(trace.PROTOCOL, "http")
		span.SetTag(trace.METHOD_NAME, string(requestCtx.Method()))
		span.SetTag(trace.REQUEST_URL, string(requestCtx.Host())+string(requestCtx.Path()))
		span.SetTag(trace.REQUEST_SIZE, "0") // TODO 如何获得
		return span
	}

	return nil
}
