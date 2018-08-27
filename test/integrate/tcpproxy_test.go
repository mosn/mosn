package integrate

import (
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
	testutil "github.com/alipay/sofa-mosn/test/util"
)

type tcpExtendCase struct {
	*testCase
}

func (c *tcpExtendCase) Start() {
	c.appServer.GoServe()
	appAddr := c.appServer.Addr()
	meshAddr := testutil.CurrentMeshAddr()
	c.clientMeshAddr = meshAddr
	cfg := testutil.CreateTCPProxyConfig(meshAddr, []string{appAddr})
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.stop
		c.appServer.Close()
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

const _NIL types.Protocol = "null"

func TestTCPProxy(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*tcpExtendCase{
		&tcpExtendCase{newTestCase(t, protocol.HTTP1, _NIL, testutil.NewHTTPServer(t))},
		&tcpExtendCase{newTestCase(t, protocol.HTTP2, _NIL, testutil.NewUpstreamHTTP2(t, appaddr))},
		&tcpExtendCase{newTestCase(t, protocol.SofaRPC, _NIL, testutil.NewRPCServer(t, appaddr, testutil.Bolt1))},
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start()
		go tc.RunCase(1)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d tcp proxy test failed, protocol: %s, error: %v\n", i, tc.AppProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d tcp proxy hang, protocol: %s\n", i, tc.AppProtocol)
		}
		close(tc.stop)
		time.Sleep(time.Second)
	}
}
