package integrate

import (
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	_ "github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/codec"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http2"
	_ "github.com/alipay/sofa-mosn/pkg/stream/sofarpc"
	_ "github.com/alipay/sofa-mosn/pkg/stream/xprotocol"
	"github.com/alipay/sofa-mosn/test/util"
)

// Notice can'T use APP(HTTPX) to MESH(SofaRPC),
// because SofaRPC is a group of protocols,such as boltV1, boltV2.
func TestCommon(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*TestCase{
		NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
		NewTestCase(t, protocol.HTTP1, protocol.HTTP2, util.NewHTTPServer(t, nil)),
		NewTestCase(t, protocol.HTTP2, protocol.HTTP1, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.SofaRPC, protocol.HTTP1, util.NewRPCServer(t, appaddr, util.Bolt1)),
		NewTestCase(t, protocol.SofaRPC, protocol.HTTP2, util.NewRPCServer(t, appaddr, util.Bolt1)),
		NewTestCase(t, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServer(t, appaddr, util.Bolt1)),

		//TODO:
		//NewTestCase(T, protocol.SofaRPC, protocol.HTTP1, util.NewRPCServer(T, appaddr, util.Bolt2)),
		//NewTestCase(T, protocol.SofaRPC, protocol.HTTP2, util.NewRPCServer(T, appaddr, util.Bolt2)),
		//NewTestCase(T, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServer(T, appaddr, util.Bolt2)),
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
		close(tc.Stop)
		time.Sleep(time.Second)
	}
}

func TestTLS(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*TestCase{
		NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
		NewTestCase(t, protocol.HTTP1, protocol.HTTP2, util.NewHTTPServer(t, nil)),
		NewTestCase(t, protocol.HTTP2, protocol.HTTP1, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.SofaRPC, protocol.HTTP1, util.NewRPCServer(t, appaddr, util.Bolt1)),
		NewTestCase(t, protocol.SofaRPC, protocol.HTTP2, util.NewRPCServer(t, appaddr, util.Bolt1)),
		NewTestCase(t, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServer(t, appaddr, util.Bolt1)),
		//TODO:
		//NewTestCase(T, protocol.SofaRPC, protocol.HTTP1, util.NewRPCServer(T, appaddr, util.Bolt2)),
		//NewTestCase(T, protocol.SofaRPC, protocol.HTTP2, util.NewRPCServer(T, appaddr, util.Bolt2)),
		//NewTestCase(T, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServer(T, appaddr, util.Bolt2)),
		//NewTestCase(T, protocol.Xprotocol, protocol.Xprotocol, util.NewRPCServer(T, appaddr, util.Xprotocol)),

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
		close(tc.Stop)
		time.Sleep(time.Second)
	}

}

func TestXprotocol(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []struct {
		*TestCase
		subProtocol string
	}{
		{
			TestCase:    NewTestCase(t, protocol.Xprotocol, protocol.Xprotocol, util.NewXProtocolServer(t, appaddr, util.XExample)),
			subProtocol: util.XExample,
		},
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartX(tc.subProtocol)
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v xprotocol: %s test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, tc.subProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v xprotocol: %s hang\n", i, tc.AppProtocol, tc.MeshProtocol, tc.subProtocol)
		}
		close(tc.Stop)
		time.Sleep(time.Second)
	}
}
