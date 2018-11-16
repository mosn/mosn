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

type TestCase struct {
	AppProtocol    types.Protocol
	MeshProtocol   types.Protocol
	C              chan error
	T              *testing.T
	AppServer      util.UpstreamServer
	ClientMeshAddr string
	ServerMeshAddr string
	Stop           chan struct{}
}

func NewTestCase(t *testing.T, app, mesh types.Protocol, server util.UpstreamServer) *TestCase {
	return &TestCase{
		AppProtocol:  app,
		MeshProtocol: mesh,
		C:            make(chan error),
		T:            t,
		AppServer:    server,
		Stop:         make(chan struct{}),
	}
}

// client - mesh - server
// not support tls
// ignore parameter : mesh protocol
func (c *TestCase) StartProxy() {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	cfg := util.CreateProxyMesh(clientMeshAddr, []string{appAddr}, c.AppProtocol)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Stop
		c.AppServer.Close()
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

// client - mesh - mesh - server
func (c *TestCase) Start(tls bool) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	serverMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateMeshToMeshConfig(clientMeshAddr, serverMeshAddr, c.AppProtocol, c.MeshProtocol, []string{appAddr}, tls)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Stop
		c.AppServer.Close()
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

// XProtocol CASE
// should use subprotocol
func (c *TestCase) StartX(subprotocol string) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	serverMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateXProtocolMesh(clientMeshAddr, serverMeshAddr, subprotocol, []string{appAddr})
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Stop
		c.AppServer.Close()
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

// mesh to mesh use tls if "istls" is true
// client do "n" times request, interval time (ms)
func (c *TestCase) RunCase(n int, interval int) {
	// Client Call
	var call func() error
	switch c.AppProtocol {
	case protocol.HTTP1:
		call = func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s/", c.ClientMeshAddr))
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
			c.T.Logf("HTTP client receive data: %s\n", string(b))
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
			resp, err := httpClient.Get(fmt.Sprintf("http://%s/", c.ClientMeshAddr))
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
			c.T.Logf("HTTP2 client receive data: %s\n", string(b))
			return nil
		}
	case protocol.SofaRPC:
		server, ok := c.AppServer.(*util.RPCServer)
		if !ok {
			c.C <- fmt.Errorf("need a sofa rpc server")
			return
		}
		client := server.Client
		if err := client.Connect(c.ClientMeshAddr); err != nil {
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
	case protocol.Xprotocol:
		server, ok := c.AppServer.(*util.XProtocolServer)
		if !ok {
			c.C <- fmt.Errorf("need a xprotocol server")
			return
		}
		client := server.Client
		if err := client.Connect(c.ClientMeshAddr); err != nil {
			c.C <- err
			return
		}
		defer client.Close()
		call = client.RequestAndWaitReponse
	default:
		c.C <- fmt.Errorf("unsupported protocol: %v", c.AppProtocol)
		return
	}
	for i := 0; i < n; i++ {
		if err := call(); err != nil {
			c.C <- err
			return
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	c.C <- nil
}
