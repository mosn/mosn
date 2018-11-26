package integrate

import ( /**/
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/trace"
	"github.com/alipay/sofa-mosn/test/util"
	"testing"
)

type TracingCase struct {
	*TestCase
	US util.UpstreamServer
}

func NewTracingCase(t *testing.T) *TracingCase {
	us := util.NewRPCServer(t, "127.0.0.1:12200", util.Bolt1)
	tc := NewTestCase(t, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServer(t, "", util.Bolt1))
	return &TracingCase{
		TestCase: tc,
		US:       us,
	}
}

func TestTracing(t *testing.T) {
	trace.PrintLog = false
	tc := NewTracingCase(t)
	tc.ClientMeshAddr = "127.0.0.1:2045"
	tc.EnableTracing = true
	tc.Tracer = "SOFATracer"
	tc.StartProxy()
	go tc.RunCase(1, 0)
	_ = <-tc.C
	span := trace.SofaTracerInstance.GetSpan()
	if span == nil {
		t.Error("Can not get span.")
	}
	close(tc.Stop)
}
