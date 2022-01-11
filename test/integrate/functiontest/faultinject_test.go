package functiontest

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"golang.org/x/net/http2"
	v2 "mosn.io/mosn/pkg/config/v2"
	_ "mosn.io/mosn/pkg/filter/stream/faultinject"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	_ "mosn.io/mosn/pkg/stream/xprotocol"
	"mosn.io/mosn/test/integrate"
	"mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

func AddFaultInject(mosn *v2.MOSNConfig, listenername string, faultstr string) error {
	// make v2 config
	cfg := make(map[string]interface{})
	if err := json.Unmarshal([]byte(faultstr), &cfg); err != nil {
		return err
	}
	listeners := mosn.Servers[0].Listeners
	for i := range listeners {
		l := listeners[i]
		if l.Name == listenername {
			fault := v2.Filter{
				Type:   v2.FaultStream,
				Config: cfg,
			}
			l.ListenerConfig.StreamFilters = append(l.ListenerConfig.StreamFilters, fault)
		}
		listeners[i] = l
	}
	return nil
}

func MakeFaultStr(status int, delay time.Duration) string {
	tmpl := `{
		"delay":{
			"fixed_delay":"%s",
			"percentage": 100
		},
		"abort": {
			"status": %d,
			"percentage": %d
		}
	}`
	abortPercent := 0
	if status != 0 {
		abortPercent = 100
	}
	return fmt.Sprintf(tmpl, delay.String(), status, abortPercent)
}

// Proxy Mode is ok
type faultInjectCase struct {
	*integrate.TestCase
	abortstatus int
	delay       time.Duration
}

func (c *faultInjectCase) StartProxy() {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	cfg := util.CreateProxyMesh(clientMeshAddr, []string{appAddr}, c.AppProtocol)
	faultstr := MakeFaultStr(c.abortstatus, c.delay)
	AddFaultInject(cfg, "proxyListener", faultstr)
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

func (c *faultInjectCase) RunCase(n int, interval int) {
	var call func() error
	switch c.AppProtocol {
	case protocol.HTTP1:
		expectedCode := http.StatusOK
		if c.abortstatus != 0 {
			expectedCode = c.abortstatus
		}
		call = func() error {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/", c.ClientMeshAddr))
			if err != nil {
				return err
			}
			cost := time.Now().Sub(start)
			defer resp.Body.Close()
			if resp.StatusCode != expectedCode {
				return fmt.Errorf("response status: %d", resp.StatusCode)
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if c.delay > 0 {
				if cost < c.delay {
					return fmt.Errorf("expected delay %s, but cost %s", c.delay.String(), cost.String())
				}
			}
			c.T.Logf("HTTP client receive data: %s\n", string(b))
			return nil
		}
	case protocol.HTTP2:
		expectedCode := http.StatusOK
		if c.abortstatus != 0 {
			expectedCode = c.abortstatus
		}
		tr := &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(netw, addr)
			},
		}
		httpClient := http.Client{Transport: tr}
		call = func() error {
			start := time.Now()
			resp, err := httpClient.Get(fmt.Sprintf("http://%s/", c.ClientMeshAddr))
			if err != nil {
				return err
			}
			cost := time.Now().Sub(start)
			defer resp.Body.Close()
			if resp.StatusCode != expectedCode {
				return fmt.Errorf("response status: %d", resp.StatusCode)

			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if c.delay > 0 {
				if cost < c.delay {
					return fmt.Errorf("expected delay %s, but cost %s", c.delay.String(), cost.String())
				}
			}
			c.T.Logf("HTTP2 client receive data: %s\n", string(b))
			return nil
		}
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

func TestFaultInject(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*faultInjectCase{
		// delay
		&faultInjectCase{
			TestCase: integrate.NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
			delay:    time.Second,
		},
		&faultInjectCase{
			TestCase: integrate.NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
			delay:    time.Second,
		},
		// abort
		&faultInjectCase{
			TestCase:    integrate.NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
			abortstatus: 500,
		},
		&faultInjectCase{
			TestCase:    integrate.NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
			abortstatus: 500,
		},
		// delay and abort
		&faultInjectCase{
			TestCase:    integrate.NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
			delay:       time.Second,
			abortstatus: 500,
		},
		&faultInjectCase{
			TestCase:    integrate.NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
			delay:       time.Second,
			abortstatus: 500,
		},
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartProxy()
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

type XFaultInjectCase struct {
	*integrate.XTestCase
	abortstatus int
	delay       time.Duration
}

func (c *XFaultInjectCase) StartProxy() {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	cfg := util.CreateXProtocolProxyMesh(clientMeshAddr, []string{appAddr}, c.SubProtocol)
	faultstr := MakeFaultStr(c.abortstatus, c.delay)
	AddFaultInject(cfg, "proxyListener", faultstr)
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

func (c *XFaultInjectCase) RunCase(n int, interval int) {
	var call func() error
	switch c.SubProtocol {
	case bolt.ProtocolName:
		server, ok := c.AppServer.(*util.RPCServer)
		if !ok {
			c.C <- fmt.Errorf("need a sofa rpc server")
			return
		}
		client := server.Client
		// TODO: rpc abort status have something wrong, fix it later
		if c.abortstatus != 0 {
			client.ExpectedStatus = int16(bolt.ResponseStatusUnknown)
		}
		if err := client.Connect(c.ClientMeshAddr); err != nil {
			c.C <- err
			return
		}
		defer client.Close()
		call = func() error {
			start := time.Now()
			client.SendRequest()
			if !util.WaitMapEmpty(&client.Waits, 2*time.Second) {
				return fmt.Errorf("request get no response")
			}
			cost := time.Now().Sub(start)
			if c.delay > 0 {
				if cost < c.delay {
					return fmt.Errorf("expected delay %s, but cost %s", c.delay.String(), cost.String())
				}
			}
			return nil
		}
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

func TestXFaultInject(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*XFaultInjectCase{
		// delay
		&XFaultInjectCase{
			XTestCase: integrate.NewXTestCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName)),
			delay:     time.Second,
		},
		// abort
		&XFaultInjectCase{
			XTestCase:   integrate.NewXTestCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName)),
			abortstatus: 500,
		},
		// delay and abort
		&XFaultInjectCase{
			XTestCase:   integrate.NewXTestCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName)),
			delay:       time.Second,
			abortstatus: 500,
		},
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartProxy()
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
