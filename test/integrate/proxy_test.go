package integrate

import (
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/test/util"
)

// Prxoy Mode
func TestProxy(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*TestCase{
		NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
		NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServer(t, appaddr, util.Bolt1)),
		//TODO:
		//NewTestCase(T, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServer(T, appaddr, util.Bolt2)),
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
		close(tc.Stop)
		time.Sleep(time.Second)
	}
}
