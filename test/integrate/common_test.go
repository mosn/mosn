package integrate

import (
	"testing"
	"time"

	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/protocol/xprotocol/tars"

	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/stream/http2"
	_ "mosn.io/mosn/pkg/stream/xprotocol"
	"mosn.io/mosn/test/util"
)

// Notice can'T use APP(HTTPX) to MESH(SofaRPC),
// because SofaRPC is a group of protocols,such as boltV1, boltV2.
func TestCommon(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*TestCase{
		NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
		// http convert case use transcoder filter instead
		//	NewTestCase(t, protocol.HTTP1, protocol.HTTP2, util.NewHTTPServer(t, nil)),
		//	NewTestCase(t, protocol.HTTP2, protocol.HTTP1, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),

		//Protocol-auto
		NewTestCase(t, protocol.HTTP2, protocol.Auto, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.HTTP1, protocol.Auto, util.NewHTTPServer(t, nil)),
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(false)
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

func TestTLS(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*TestCase{
		NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
		// NewTestCase(t, protocol.HTTP1, protocol.HTTP2, util.NewHTTPServer(t, nil)),
		// NewTestCase(t, protocol.HTTP2, protocol.HTTP1, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),

		//Protocol-auto
		NewTestCase(t, protocol.HTTP2, protocol.Auto, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.HTTP1, protocol.Auto, util.NewHTTPServer(t, nil)),
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(true)
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v tls test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		tc.FinishCase()
	}

}

func TestXprotocol(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*XTestCase{
		NewXTestCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName)),
		NewXTestCase(t, dubbo.ProtocolName, util.NewRPCServer(t, appaddr, dubbo.ProtocolName)),
		NewXTestCase(t, tars.ProtocolName, util.NewRPCServer(t, appaddr, tars.ProtocolName)),
		//TODO: boltv2
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(false)
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v xprotocol: %s test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, tc.SubProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v xprotocol: %s hang\n", i, tc.AppProtocol, tc.MeshProtocol, tc.SubProtocol)
		}
		tc.FinishCase()
	}
}

func TestXprotocolTLS(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*XTestCase{
		NewXTestCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName)),
		NewXTestCase(t, dubbo.ProtocolName, util.NewRPCServer(t, appaddr, dubbo.ProtocolName)),
		NewXTestCase(t, tars.ProtocolName, util.NewRPCServer(t, appaddr, tars.ProtocolName)),
		//TODO: boltv2
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(true)
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v xprotocol: %s test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, tc.SubProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v xprotocol: %s hang\n", i, tc.AppProtocol, tc.MeshProtocol, tc.SubProtocol)
		}
		tc.FinishCase()
	}
}
