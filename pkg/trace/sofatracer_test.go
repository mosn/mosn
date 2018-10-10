package trace

import "testing"

func TestSofaTracerPrintSpan(t *testing.T) {
	SofaTracerInstance.printSpan(&SofaTracerSpan{})
}

func TestSofaTracerPrintIngressSpan(t *testing.T) {
	span := &SofaTracerSpan{
		tags: map[string]string{},
	}
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
