package integrate

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	_ "github.com/alipay/sofa-mosn/pkg/filter/network/proxy"
	_ "github.com/alipay/sofa-mosn/pkg/filter/network/tcpproxy"
	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
	"golang.org/x/net/http2"
)

type testCase struct {
	AppProtocol    types.Protocol
	MeshProtocol   types.Protocol
	C              chan error
	t              *testing.T
	appServer      util.UpstreamServer
	clientMeshAddr string
	stop           chan struct{}
}

func newTestCase(t *testing.T, app, mesh types.Protocol, server util.UpstreamServer) *testCase {
	return &testCase{
		AppProtocol:  app,
		MeshProtocol: mesh,
		C:            make(chan error),
		t:            t,
		appServer:    server,
		stop:         make(chan struct{}),
	}
}

// client - mesh - server
// not support tls
// ignore parameter : mesh protocol
func (c *testCase) StartProxy() {
	c.appServer.GoServe()
	appAddr := c.appServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.clientMeshAddr = clientMeshAddr
	cfg := util.CreateProxyMesh(clientMeshAddr, []string{appAddr}, c.AppProtocol)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.stop
		c.appServer.Close()
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

// client - mesh - mesh - server
func (c *testCase) Start(tls bool) {
	c.appServer.GoServe()
	appAddr := c.appServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.clientMeshAddr = clientMeshAddr
	serverMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateMeshToMeshConfig(clientMeshAddr, serverMeshAddr, c.AppProtocol, c.MeshProtocol, []string{appAddr}, tls)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.stop
		c.appServer.Close()
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

// mesh to mesh use tls if "istls" is true
// client do "n" times request
func (c *testCase) RunCase(n int) {
	// Client Call
	var call func() error
	switch c.AppProtocol {
	case protocol.HTTP1:
		call = func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s/", c.clientMeshAddr))
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("response status: %d", resp.StatusCode)
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			c.t.Logf("HTTP client receive data: %s\n", string(b))
			return nil
		}
	case protocol.HTTP2:
		tr := &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(netw, addr)
			},
		}
		httpClient := http.Client{Transport: tr}
		call = func() error {
			resp, err := httpClient.Get(fmt.Sprintf("http://%s/", c.clientMeshAddr))
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("response status: %d", resp.StatusCode)

			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			c.t.Logf("HTTP2 client receive data: %s\n", string(b))
			return nil
		}
	case protocol.SofaRPC:
		server, ok := c.appServer.(*util.RPCServer)
		if !ok {
			c.C <- fmt.Errorf("need a sofa rpc server")
			return
		}
		client := server.Client
		if err := client.Connect(c.clientMeshAddr); err != nil {
			c.C <- err
			return
		}
		defer client.Close()
		call = func() error {
			client.SendRequest()
			if !util.WaitMapEmpty(&client.Waits, 2*time.Second) {
				return fmt.Errorf("request get no response")
			}
			return nil
		}
	default:
		c.C <- fmt.Errorf("unsupported protocol: %v", c.AppProtocol)
		return
	}
	for i := 0; i < n; i++ {
		if err := call(); err != nil {
			c.C <- err
			return
		}
	}
	c.C <- nil
}
