package otel

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"mosn.io/mosn/pkg/protocol/http"
	"testing"
	"time"
)

func TestSpan(t *testing.T) {
	tracer, err := NewTracer(nil)
	assert.NoError(t, err)
	assert.NotNil(t, tracer)

	request := http.RequestHeader{
		RequestHeader: &fasthttp.RequestHeader{},
	}

	span := tracer.Start(context.Background(), request, time.Now())
	assert.NotEqual(t, span.SpanId(), "")
	assert.NotEqual(t, span.TraceId(), "")
	assert.Equal(t, span.ParentSpanId(), "")

	span.FinishSpan()
}
