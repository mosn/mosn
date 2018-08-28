package integrate

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	_ "github.com/alipay/sofa-mosn/pkg/filter/network/proxy"
	_ "github.com/alipay/sofa-mosn/pkg/filter/network/tcpproxy"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/stream/xprotocol/subprotocol"
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
	// TODO : add xprotocol test
	case protocol.Xprotocol:
		stopChan := make(chan struct{})
		remoteAddr, _ := net.ResolveTCPAddr("tcp", c.clientMeshAddr)
		cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
		cc.SetReadDisable(true)
		if err := cc.Connect(true); err != nil {
			c.C <- fmt.Errorf("xprotocol connect fail %v\n", err)
		}
		call = func() error {
			testData := []byte{14, 1, 0, 8, 0, 0, 3, 0, 0, 0, 0, 0, 0, 112, 32, 0}
			exampleCodec := subprotocol.NewRPCExample()
			reqStreamID := exampleCodec.GetStreamID(testData)
			//fmt.Printf("client send request,stream id = %v\n", reqStreamID)
			cc.Write(buffer.NewIoBufferBytes(testData))
			for {
				now := time.Now()
				cc.RawConn().SetReadDeadline(now.Add(30 * time.Second))
				buf := make([]byte, 10*1024)
				bytesRead, err := cc.RawConn().Read(buf)
				//fmt.Printf("client try to read ,err = %v \n", err)
				if err != nil {
					if err, ok := err.(net.Error); ok && err.Timeout() {
						continue
					} else {
						//fmt.Errorf("client read error = %v", err)
						return err
					}
				}
				if bytesRead > 0 {
					//fmt.Printf("server response %v\n", string(buf[:bytesRead]))
					resStreamID := exampleCodec.GetStreamID(buf[:bytesRead])
					if resStreamID != reqStreamID {
						return fmt.Errorf("streamID not match")
					}
					break
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
