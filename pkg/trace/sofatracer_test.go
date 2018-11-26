package trace

import (
	"testing"
	"time"
)

func init() {
	CreateInstance()
}

func TestSofaTracerStartFinish(t *testing.T) {
	span := SofaTracerInstance.Start(time.Now())
	span.SetTag(TRACE_ID, IdGen().GenerateTraceId())
	span.FinishSpan()
}

func TestSofaTracerPrintSpan(t *testing.T) {
	SofaTracerInstance.printSpan(&SofaTracerSpan{})
}

func TestSofaTracerPrintIngressSpan(t *testing.T) {
	span := &SofaTracerSpan{
		tags: map[string]string{},
	}
	span.tags[DOWNSTEAM_HOST_ADDRESS] = "127.0.0.1:43"
	span.tags[SPAN_TYPE] = "ingress"
	SofaTracerInstance.printSpan(span)
}

func TestSofaTracerPrintEgressSpan(t *testing.T) {
	span := &SofaTracerSpan{
		tags: map[string]string{},
	}
	span.tags[SPAN_TYPE] = "egress"
	SofaTracerInstance.printSpan(span)
}
