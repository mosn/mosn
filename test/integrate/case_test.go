package integrate

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/cmd/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
	"golang.org/x/net/http2"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network/buffer"
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

	// TODO : add xprotocol test
	case protocol.Xprotocol:
		stopChan := make(chan struct{})
		remoteAddr, _ := net.ResolveTCPAddr("tcp", c.clientMeshAddr)
		cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
		if err := cc.Connect(true); err != nil {
			c.C <- fmt.Errorf("xprotocol connect fail %v\n",err)
		}
		call = func() error {
			testData := []byte{14,1,0,8,0,0,3,0,0,0,0,0,0,112,32,0,0,7,208,0,0,0,45,0,0,0,8,0,0,0,1,0,0,0,16,0,0,0,6,0,0,0,143,99,111,109,46,116,101,115,116,46,112,97,110,100,111,114,97,46,104,115,102,46,72,101,108,108,111,83,101,114,118,105,99,101,58,49,46,48,46,48,46,100,97,105,108,121,115,97,121,72,101,108,108,111,106,97,118,97,46,108,97,110,103,46,83,116,114,105,110,103,5,119,111,114,108,100,72,16,67,111,110,115,117,109,101,114,45,65,112,112,78,97,109,101,15,104,115,102,45,115,101,114,118,101,114,45,100,101,109,111,12,116,97,114,103,101,116,95,103,114,111,117,112,4,72,83,70,49,4,95,84,73,68,78,16,101,97,103,108,101,101,121,101,95,99,111,110,116,101,120,116,72,7,116,114,97,99,101,73,100,30,48,97,57,55,54,51,54,55,49,53,51,53,49,48,48,54,52,50,52,54,56,53,54,57,57,100,48,48,54,50,5,114,112,99,73,100,1,57,16,101,97,103,108,101,69,121,101,85,115,101,114,68,97,116,97,78,90,90}
			cc.Write(buffer.NewIoBufferBytes(testData))
			for {
				now := time.Now()
				cc.RawConn().SetReadDeadline(now.Add(30 * time.Second))
				buf := make([]byte, 10*1024)
				bytesRead, err := cc.RawConn().Read(buf)
				if err != nil {
					if err, ok := err.(net.Error); ok && err.Timeout() {
						continue
					}
				}
				if bytesRead > 0 {
					fmt.Printf("server response %v\n",string(buf))
				}
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
