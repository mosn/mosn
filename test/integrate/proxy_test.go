package integrate

import (
	"testing"
	"time"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/test/util"
)

// Proxy Mode
func TestProxy(t *testing.T) {
	testCases := []*TestCase{
		NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
		NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2WithAnyPort(t, nil)),
		NewTestCase(t, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServerWithAnyPort(t, util.Bolt1)),
		//TODO:
		//NewTestCase(T, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServerWithAnyPort(t, util.Bolt2)),
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartProxy()
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		tc.FinishCase()
	}
}
