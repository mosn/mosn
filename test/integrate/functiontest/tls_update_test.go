package functiontest

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mosn"
	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/protocol/http/conv"
	_ "mosn.io/mosn/pkg/protocol/http2/conv"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/server"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/stream/http2"
	_ "mosn.io/mosn/pkg/stream/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/test/util"
)

const (
	caPEM = `-----BEGIN CERTIFICATE-----
MIIBVzCB/qADAgECAhBsIQij0idqnmDVIxbNRxRCMAoGCCqGSM49BAMCMBIxEDAO
BgNVBAoTB0FjbWUgQ28wIBcNNzAwMTAxMDAwMDAwWhgPMjA4NDAxMjkxNjAwMDBa
MBIxEDAOBgNVBAoTB0FjbWUgQ28wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARV
DG+YT6LzaR5r0Howj4/XxHtr3tJ+llqg9WtTJn0qMy3OEUZRfHdP9iuJ7Ot6rwGF
i6RXy1PlAurzeFzDqQY8ozQwMjAOBgNVHQ8BAf8EBAMCAqQwDwYDVR0TAQH/BAUw
AwEB/zAPBgNVHREECDAGhwR/AAABMAoGCCqGSM49BAMCA0gAMEUCIQDt9WA96LJq
VvKjvGhhTYI9KtbC0X+EIFGba9lsc6+ubAIgTf7UIuFHwSsxIVL9jI5RkNgvCA92
FoByjq7LS7hLSD8=
-----END CERTIFICATE-----
`
	certPEM = `-----BEGIN CERTIFICATE-----
MIIBdDCCARqgAwIBAgIQbCEIo9Inap5g1SMWzUcUQjAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYwMDAw
WjASMRAwDgYDVQQKEwdBY21lIENvMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
VQxvmE+i82kea9B6MI+P18R7a97SfpZaoPVrUyZ9KjMtzhFGUXx3T/Yriezreq8B
hYukV8tT5QLq83hcw6kGPKNQME4wDgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQG
CCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMA8GA1UdEQQIMAaHBH8A
AAEwCgYIKoZIzj0EAwIDSAAwRQIgO9xLIF1AoBsSMU6UgNE7svbelMAdUQgEVIhq
K3gwoeICIQCDC75Fa3XQL+4PZatS3OfG93XNFyno9koyn5mxLlDAAg==
-----END CERTIFICATE-----
`
	keyPEM = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICWksdaVL6sOu33VeohiDuQ3gP8xlQghdc+2FsWPSkrooAoGCCqGSM49
AwEHoUQDQgAEVQxvmE+i82kea9B6MI+P18R7a97SfpZaoPVrUyZ9KjMtzhFGUXx3
T/Yriezreq8BhYukV8tT5QLq83hcw6kGPA==
-----END EC PRIVATE KEY-----
`
)

// for test tls update, we use a client - tls -> mosn -> upstream server
// protocol include http/http2/sofarpc(boltv1)
// FIXME: mosn listen http2 with tls have bugs.(no alpn)
// case1 mosn listen a non-tls changed to tls (needs to create new connection)
// case2 mosn listen a tls (without inspector) changed to non-tls (needs to create new connection)
// case3 mosn listen a tls (without inspector) changed to inspector mode
type TLSUpdateCase struct {
	Protocol     types.ProtocolName
	AppServer    util.UpstreamServer
	MeshAddr     string
	ListenerName string
	C            chan error
	T            *testing.T
	Finish       chan bool
}

func NewTLSUpdateCase(t *testing.T, proto types.ProtocolName, server util.UpstreamServer) *TLSUpdateCase {
	return &TLSUpdateCase{
		Protocol:  proto,
		AppServer: server,
		C:         make(chan error),
		T:         t,
		Finish:    make(chan bool),
	}
}

type XExtendTLSUpdateCase struct {
	*TLSUpdateCase
}

func NewXExtendTLSUpdateCase(t *testing.T, proto types.ProtocolName, server util.UpstreamServer) *XExtendTLSUpdateCase {
	return &XExtendTLSUpdateCase{
		TLSUpdateCase: &TLSUpdateCase{
			Protocol:  proto,
			AppServer: server,
			C:         make(chan error),
			T:         t,
			Finish:    make(chan bool),
		},
	}
}

var DefaultTLSConfig = v2.TLSConfig{
	Status:     true,
	CACert:     caPEM,
	CertChain:  certPEM,
	PrivateKey: keyPEM,
	ServerName: "127.0.0.1",
}

func MakeProxyWithTLSConfig(listenerName string, addr string, hosts []string, proto types.ProtocolName, tls bool, xproto bool) *v2.MOSNConfig {
	clusterName := "upstream"
	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			util.NewBasicCluster(clusterName, hosts),
		},
	}
	routers := []v2.Router{
		util.NewPrefixRouter(clusterName, "/"),
		util.NewHeaderRouter(clusterName, ".*"),
	}
	filterChain := util.NewFilterChain("proxyVirtualHost", proto, proto, routers)
	if xproto {
		filterChain = util.NewXProtocolFilterChain("proxyVirtualHost", proto, routers)
	}

	if tls {
		filterChain.TLSContexts = []v2.TLSConfig{
			DefaultTLSConfig,
		}
	}
	chains := []v2.FilterChain{filterChain}
	lnCfg := util.NewListener(listenerName, addr, chains)
	lnCfg.Inspector = false
	mosnConfig := util.NewMOSNConfig([]v2.Listener{lnCfg}, cmconfig)
	mosnConfig.RawAdmin = json.RawMessage([]byte(`{
		 "address":{
			 "socket_address":{
				 "address": "127.0.0.1",
				 "port_value": 8888
			 }
		 }
	}`))
	return mosnConfig

}

func (c *TLSUpdateCase) Start(tls bool) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	c.MeshAddr = util.CurrentMeshAddr()
	c.ListenerName = "test_dynamic"
	cfg := MakeProxyWithTLSConfig(c.ListenerName, c.MeshAddr, []string{appAddr}, c.Protocol, tls, false)
	// for test, reset adapter
	server.ResetAdapter()
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		c.AppServer.Close()
		mesh.Close()
		time.Sleep(5 * time.Second)
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

func (c *TLSUpdateCase) FinishCase() {
	c.Finish <- true
	<-c.Finish
}

func (c *TLSUpdateCase) UpdateTLS(inspector bool, cfgs []v2.TLSConfig) error {
	adapter := server.GetListenerAdapterInstance()
	return adapter.UpdateListenerTLS("", c.ListenerName, inspector, cfgs)
}

// client do "n" times request, interval time (ms)
func (c *TLSUpdateCase) RequestTLS(isTLS bool, n int, interval int) {
	var call func() error
	switch c.Protocol {
	case protocol.HTTP1:
		tr := http.DefaultTransport.(*http.Transport)
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM([]byte(caPEM))
		tr.TLSClientConfig = &tls.Config{
			RootCAs: pool,
		}
		httpClient := http.Client{
			Transport: tr,
		}
		call = func() error {
			scheme := "http://"
			if isTLS {
				scheme = "https://"
			}
			resp, err := httpClient.Get(fmt.Sprintf("%s%s/", scheme, c.MeshAddr))
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
		var httpClient http.Client
		if !isTLS {
			tr := &http2.Transport{
				AllowHTTP: true,
				DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
					return net.Dial(netw, addr)
				},
			}
			httpClient = http.Client{Transport: tr}

		} else {
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM([]byte(caPEM))
			httpClient = http.Client{
				Transport: &http2.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs: pool,
					},
				},
			}
		}
		call = func() error {
			scheme := "http://"
			if isTLS {
				scheme = "https://"
			}
			resp, err := httpClient.Get(fmt.Sprintf("%s%s/", scheme, c.MeshAddr))
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

func (c *XExtendTLSUpdateCase) Start(tls bool) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	c.MeshAddr = util.CurrentMeshAddr()
	c.ListenerName = "test_dynamic"
	cfg := MakeProxyWithTLSConfig(c.ListenerName, c.MeshAddr, []string{appAddr}, c.Protocol, tls, true)
	// for test, reset adapter
	server.ResetAdapter()
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		c.AppServer.Close()
		mesh.Close()
		time.Sleep(5 * time.Second)
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

// client do "n" times request, interval time (ms)
func (c *XExtendTLSUpdateCase) RequestTLS(isTLS bool, n int, interval int) {
	var call func() error
	switch c.Protocol {
	case bolt.ProtocolName:
		server, ok := c.AppServer.(*util.RPCServer)
		if !ok {
			c.C <- fmt.Errorf("need a sofa rpc server")
			return
		}
		client := server.Client
		if isTLS {
			tlsCfg := &v2.TLSConfig{
				Status:     true,
				CACert:     caPEM,
				ServerName: "127.0.0.1",
			}
			if err := client.ConnectTLS(c.MeshAddr, tlsCfg); err != nil {
				c.C <- err
				return
			}
		} else {
			if err := client.Connect(c.MeshAddr); err != nil {
				c.C <- err
				return
			}
		}
		defer client.Close() // close the connection
		call = func() error {
			client.SendRequest()
			if !util.WaitMapEmpty(&client.Waits, 2*time.Second) {
				return fmt.Errorf("request get no response")
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

// NoneToTLS
// first listen a non-tls listener, then update to tls
func TestUpdateTLS_NoneToTLS(t *testing.T) {
	testCases := []*TLSUpdateCase{
		NewTLSUpdateCase(t, protocol.HTTP1, util.NewHTTPServer(t, nil)),
		// NewTLSUpdateCase(t, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
	}
	for i, tc := range testCases {
		verify := func() {
			select {
			case err := <-tc.C:
				if err != nil {
					t.Errorf("request failed, case %d, protocol %v, error: %v", i, tc.Protocol, err)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("request hung up case %d, protocol %v", i, tc.Protocol)
			}
		}
		t.Logf("start case #%d\n", i)
		tc.Start(false)
		go tc.RequestTLS(false, 1, 0)
		t.Logf("verify non-tls")
		verify()
		// update to tls
		if err := tc.UpdateTLS(false, []v2.TLSConfig{DefaultTLSConfig}); err != nil {
			t.Fatal("update tls failed")
		}
		go tc.RequestTLS(true, 1, 0)
		t.Logf("verify tls")
		verify()
		tc.FinishCase()

	}
}

// TLSToNone
// first listen a tls listener, then update to non-tls
func TestUpdateTLS_TLSToNone(t *testing.T) {
	testCases := []*TLSUpdateCase{
		NewTLSUpdateCase(t, protocol.HTTP1, util.NewHTTPServer(t, nil)),
		// NewTLSUpdateCase(t, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
	}
	for i, tc := range testCases {
		verify := func() {
			select {
			case err := <-tc.C:
				if err != nil {
					t.Errorf("request failed, case %d, protocol %v, error: %v", i, tc.Protocol, err)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("request hung up case %d, protocol %v", i, tc.Protocol)
			}
		}
		t.Logf("start case #%d\n", i)
		tc.Start(true)
		go tc.RequestTLS(true, 1, 0)
		verify()
		// update to non-tls
		if err := tc.UpdateTLS(false, []v2.TLSConfig{v2.TLSConfig{}}); err != nil {
			t.Fatal("update tls failed")
		}
		go tc.RequestTLS(false, 1, 0)
		verify()
		// finish
		tc.FinishCase()
	}

}

// TLS to inspector
func TestUpdateTLS_TLSToInspector(t *testing.T) {
	testCases := []*TLSUpdateCase{
		NewTLSUpdateCase(t, protocol.HTTP1, util.NewHTTPServer(t, nil)),
		// NewTLSUpdateCase(t, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
	}
	for i, tc := range testCases {
		verify := func() {
			select {
			case err := <-tc.C:
				if err != nil {
					t.Errorf("request failed, case %d, protocol %v, error: %v", i, tc.Protocol, err)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("request hung up case %d, protocol %v", i, tc.Protocol)
			}
		}
		t.Logf("start case #%d\n", i)
		tc.Start(true)
		go tc.RequestTLS(true, 1, 0)
		verify()
		// update to inspector
		if err := tc.UpdateTLS(true, []v2.TLSConfig{DefaultTLSConfig}); err != nil {
			t.Fatal("update tls failed")
		}
		go tc.RequestTLS(false, 1, 0)
		verify()
		tc.FinishCase()
	}
}

// NoneToTLS
// first listen a non-tls listener, then update to tls
func TestXUpdateTLS_NoneToTLS(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*XExtendTLSUpdateCase{
		NewXExtendTLSUpdateCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName)),
	}
	for i, tc := range testCases {
		verify := func() {
			select {
			case err := <-tc.C:
				if err != nil {
					t.Errorf("request failed, case %d, protocol %v, error: %v", i, tc.Protocol, err)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("request hung up case %d, protocol %v", i, tc.Protocol)
			}
		}
		t.Logf("start case #%d\n", i)
		tc.Start(false)
		go tc.RequestTLS(false, 1, 0)
		t.Logf("verify non-tls")
		verify()
		// update to tls
		if err := tc.UpdateTLS(false, []v2.TLSConfig{DefaultTLSConfig}); err != nil {
			t.Fatal("update tls failed")
		}
		go tc.RequestTLS(true, 1, 0)
		t.Logf("verify tls")
		verify()
		tc.FinishCase()

	}
}

// xprotocol TLSToNone
// first listen a tls listener, then update to non-tls
func TestXUpdateTLS_TLSToNone(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*XExtendTLSUpdateCase{
		NewXExtendTLSUpdateCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName)),
	}
	for i, tc := range testCases {
		verify := func() {
			select {
			case err := <-tc.C:
				if err != nil {
					t.Errorf("request failed, case %d, protocol %v, error: %v", i, tc.Protocol, err)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("request hung up case %d, protocol %v", i, tc.Protocol)
			}
		}
		t.Logf("start case #%d\n", i)
		tc.Start(true)
		go tc.RequestTLS(true, 1, 0)
		verify()
		// update to non-tls
		if err := tc.UpdateTLS(false, []v2.TLSConfig{v2.TLSConfig{}}); err != nil {
			t.Fatal("update tls failed")
		}
		go tc.RequestTLS(false, 1, 0)
		verify()
		// finish
		tc.FinishCase()
	}

}

// TLS to inspector
func TestXUpdateTLS_TLSToInspector(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*XExtendTLSUpdateCase{
		NewXExtendTLSUpdateCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName)),
	}
	for i, tc := range testCases {
		verify := func() {
			select {
			case err := <-tc.C:
				if err != nil {
					t.Errorf("request failed, case %d, protocol %v, error: %v", i, tc.Protocol, err)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("request hung up case %d, protocol %v", i, tc.Protocol)
			}
		}
		t.Logf("start case #%d\n", i)
		tc.Start(true)
		go tc.RequestTLS(true, 1, 0)
		verify()
		// update to inspector
		if err := tc.UpdateTLS(true, []v2.TLSConfig{DefaultTLSConfig}); err != nil {
			t.Fatal("update tls failed")
		}
		go tc.RequestTLS(false, 1, 0)
		verify()
		tc.FinishCase()
	}
}
