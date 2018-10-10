package trace

import "testing"

func TestSofaTracerPrintSpan(t *testing.T) {
	SofaTracerInstance.printSpan(&SofaTracerSpan{})
}
