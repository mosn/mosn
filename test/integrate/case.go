package integrate

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbothrift"
	"mosn.io/mosn/pkg/protocol/xprotocol/tars"

	"golang.org/x/net/http2"
	_ "mosn.io/mosn/pkg/filter/network/proxy"
	_ "mosn.io/mosn/pkg/filter/network/streamproxy"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

// TODO: base interface
//type Case interface {
//	StartProxy()
//	Start(tls bool)
//	RunCase(n int, interval int)
//}

type TestCase struct {
	AppProtocol    types.ProtocolName
	MeshProtocol   types.ProtocolName
	C              chan error
	T              *testing.T
	AppServer      util.UpstreamServer
	ClientMeshAddr string
	ServerMeshAddr string
	Finish         chan bool
}

func NewTestCase(t *testing.T, app, mesh types.ProtocolName, server util.UpstreamServer) *TestCase {
	return &TestCase{
		AppProtocol:  app,
		MeshProtocol: mesh,
		C:            make(chan error),
		T:            t,
		AppServer:    server,
		Finish:       make(chan bool),
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
		<-c.Finish
		c.AppServer.Close()
		mesh.Close()
		c.Finish <- true
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
		<-c.Finish
		c.AppServer.Close()
		mesh.Close()
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

// Finish case and wait close returns
func (c *TestCase) FinishCase() {
	c.Finish <- true
	<-c.Finish
}

const HTTPTestPath = "test/path"

// mesh to mesh use tls if "istls" is true
// client do "n" times request, interval time (ms)
func (c *TestCase) RunCase(n int, interval int) {
	// Client Call
	var call func() error
	switch c.AppProtocol {
	case protocol.HTTP1:
		call = func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s/%s", c.ClientMeshAddr, HTTPTestPath))
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
			resp, err := httpClient.Get(fmt.Sprintf("http://%s/%s", c.ClientMeshAddr, HTTPTestPath))
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

// XProtocol case, should use sub-protocol
type XTestCase struct {
	TestCase
	SubProtocol types.ProtocolName
}

func NewXTestCase(t *testing.T, subProtocol types.ProtocolName, server util.UpstreamServer) *XTestCase {
	return &XTestCase{
		TestCase: TestCase{
			AppProtocol:  types.ProtocolName("X"),
			MeshProtocol: types.ProtocolName("X"),
			C:            make(chan error),
			T:            t,
			AppServer:    server,
			Finish:       make(chan bool),
		},
		SubProtocol: subProtocol,
	}
}

// client - mesh - server
// not support tls
// ignore parameter : mesh protocol
func (c *XTestCase) StartProxy() {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	cfg := util.CreateXProtocolProxyMesh(clientMeshAddr, []string{appAddr}, c.SubProtocol)
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

func (c *XTestCase) Start(tls bool) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	serverMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateXProtocolMesh(clientMeshAddr, serverMeshAddr, c.SubProtocol, []string{appAddr}, tls)
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

// mesh to mesh use tls if "istls" is true
// client do "n" times request, interval time (ms)
func (c *XTestCase) RunCase(n int, interval int) {
	// Client Call
	var call func() error
	switch c.SubProtocol {
	case bolt.ProtocolName, dubbo.ProtocolName, tars.ProtocolName, dubbothrift.ProtocolName:
		server, ok := c.AppServer.(*util.RPCServer)
		if !ok {
			c.C <- fmt.Errorf("need a xprotocol rpc server")
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
