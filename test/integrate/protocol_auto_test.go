package integrate

import (
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	_ "github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc/codec"
	_ "github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc/conv"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http2"
	_ "github.com/alipay/sofa-mosn/pkg/stream/mhttp2"
	_ "github.com/alipay/sofa-mosn/pkg/stream/sofarpc"
	_ "github.com/alipay/sofa-mosn/pkg/stream/xprotocol"
	"github.com/alipay/sofa-mosn/test/util"
)

func (c *TestCase) StartAuto(tls bool) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	serverMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateMeshToMeshConfig(clientMeshAddr, serverMeshAddr, protocol.Auto, protocol.Auto, []string{appAddr}, tls)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Stop
		c.AppServer.Close()
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

func TestAuto(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*TestCase{
		NewTestCase(t, protocol.MHTTP2, protocol.MHTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),

	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartAuto(false)
		go tc.RunCase(5, 0)
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

func TestAutoTLS(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*TestCase{
		NewTestCase(t, protocol.MHTTP2, protocol.MHTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),

	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartAuto(true)
		go tc.RunCase(5, 0)
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
