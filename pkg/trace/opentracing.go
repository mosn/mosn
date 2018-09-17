package trace

import (
	"fmt"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/opentracing/opentracing-go"
	"time"
)

type OpenTracingSpan struct {
	span opentracing.Span
}

func (s *OpenTracingSpan) SetOperation(operation string) {
	s.span.SetOperationName(operation)
}

func (s *OpenTracingSpan) SetTag(key string, value string) {
	s.span.SetTag(key, value)
}

func (s *OpenTracingSpan) FinishSpan() {
	s.span.Finish()
}

func (s *OpenTracingSpan) InjectContext(requestHeaders map[string]string) {
}

func (s *OpenTracingSpan) SpawnChild(operationName string, startTime time.Time) types.Span {
	return nil
}

func (s *OpenTracingSpan) String() string {
	return fmt.Sprintf("%v", s.span)
}

type OpenTracingDriver struct {
}

func (driver *OpenTracingDriver) Start(requestHeaders map[string]string, operationName string, startTime time.Time) types.Span {
	span := &OpenTracingSpan{
		span: &SimpleOpenTracingSpan{
			startTime: startTime,
			tags:      map[string]interface{}{},
		},
	}

	if requestHeaders[TRACE_ID] == "" {
		span.SetTag(TRACE_ID, IdGen().GenerateTraceId())
	} else {
		span.SetTag(TRACE_ID, requestHeaders[TRACE_ID])
	}

	return span
}
