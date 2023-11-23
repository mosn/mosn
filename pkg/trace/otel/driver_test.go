package otel

import (
	"context"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"testing"
	"time"
)

type mockTracer struct {
}

//goland:noinspection GoUnusedParameter
func (receiver *mockTracer) Start(ctx context.Context, request interface{}, startTime time.Time) api.Span {
	return nil
}

func TestNewOtelImpl(t *testing.T) {
	protocol := types.ProtocolName("test")

	otelDriver := NewOtelImpl()
	otelDriver.Register(protocol, func(config map[string]interface{}) (tracer api.Tracer, err error) {
		return &mockTracer{}, nil
	})

	_ = otelDriver.Init(nil)

	mockTracer := otelDriver.Get(protocol)
	assert.NotNil(t, mockTracer)

}
