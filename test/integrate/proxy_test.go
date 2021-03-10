package integrate

import (
	"testing"
	"time"

	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbothrift"
	"mosn.io/mosn/pkg/protocol/xprotocol/tars"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/test/util"
)

// Proxy Mode
func TestProxy(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*TestCase{
		NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
		NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
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

// Proxy Mode with xprotocol
func TestXProxy(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*XTestCase{
		NewXTestCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName)),
		NewXTestCase(t, dubbo.ProtocolName, util.NewRPCServer(t, appaddr, dubbo.ProtocolName)),
		NewXTestCase(t, tars.ProtocolName, util.NewRPCServer(t, appaddr, tars.ProtocolName)),
		NewXTestCase(t, dubbothrift.ProtocolName, util.NewRPCServer(t, appaddr, dubbothrift.ProtocolName)),
		//TODO: boltv2
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartProxy()
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v proxy to mesh %v xprotocol: %s test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, tc.SubProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v proxy to mesh %v xprotocol: %s hang\n", i, tc.AppProtocol, tc.SubProtocol, tc.MeshProtocol)
		}
		tc.FinishCase()
	}
}
