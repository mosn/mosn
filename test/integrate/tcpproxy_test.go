package integrate

import (
	"testing"
	"time"

	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	testutil "mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

type tcpExtendCase struct {
	*TestCase
}

func (c *tcpExtendCase) Start(isRouteEntryMode bool) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	meshAddr := testutil.CurrentMeshAddr()
	c.ClientMeshAddr = meshAddr
	cfg := testutil.CreateTCPProxyConfig(meshAddr, []string{appAddr}, isRouteEntryMode)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		c.AppServer.Close()
		mesh.Close()
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

const _NIL types.ProtocolName = "null"

func TestTCPProxy(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*tcpExtendCase{
		&tcpExtendCase{NewTestCase(t, protocol.HTTP1, _NIL, testutil.NewHTTPServer(t, nil))},
		&tcpExtendCase{NewTestCase(t, protocol.HTTP2, _NIL, testutil.NewUpstreamHTTP2(t, appaddr, nil))},
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(false)
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d tcp proxy test failed, protocol: %s, error: %v\n", i, tc.AppProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d tcp proxy hang, protocol: %s\n", i, tc.AppProtocol)
		}
		tc.FinishCase()
	}
}
func TestTCPProxyRouteEntry(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*tcpExtendCase{
		&tcpExtendCase{NewTestCase(t, protocol.HTTP1, _NIL, testutil.NewHTTPServer(t, nil))},
		&tcpExtendCase{NewTestCase(t, protocol.HTTP2, _NIL, testutil.NewUpstreamHTTP2(t, appaddr, nil))},
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(true)
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d tcp proxy route entry test failed, protocol: %s, error: %v\n", i, tc.AppProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d tcp proxy route entry hang, protocol: %s\n", i, tc.AppProtocol)
		}
		tc.FinishCase()
	}
}

type tcpXExtendCase struct {
	*XTestCase
}

func (c *tcpXExtendCase) Start(isRouteEntryMode bool) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	meshAddr := testutil.CurrentMeshAddr()
	c.ClientMeshAddr = meshAddr
	cfg := testutil.CreateTCPProxyConfig(meshAddr, []string{appAddr}, isRouteEntryMode)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		c.AppServer.Close()
		mesh.Close()
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

func TestXTCPProxy(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*tcpXExtendCase{
		&tcpXExtendCase{NewXTestCase(t, bolt.ProtocolName, testutil.NewRPCServer(t, appaddr, bolt.ProtocolName))},
		//TODO: boltv2, dubbo, tars
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(false)
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d tcp proxy test failed, protocol: %s, error: %v\n", i, tc.SubProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d tcp proxy hang, protocol: %s\n", i, tc.SubProtocol)
		}
		tc.FinishCase()
	}
}
func TestXTCPProxyRouteEntry(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*tcpXExtendCase{
		&tcpXExtendCase{NewXTestCase(t, bolt.ProtocolName, testutil.NewRPCServer(t, appaddr, bolt.ProtocolName))},
		//TODO: boltv2, dubbo, tars
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(true)
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d tcp proxy route entry test failed, protocol: %s, error: %v\n", i, tc.SubProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d tcp proxy route entry hang, protocol: %s\n", i, tc.SubProtocol)
		}
		tc.FinishCase()
	}
}
