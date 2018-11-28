package functiontest

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
	"golang.org/x/net/http2"
)

// Test Direct Response
// TODO: fix direct response with body in rpc
// TODO: rpc status have something wrong

// Direct Response ignore upstream cluster information and route rule config
func CreateDirectMeshProxy(addr string, proto types.Protocol, response *v2.DirectResponseAction) *config.MOSNConfig {
	cmconfig := config.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			v2.Cluster{
				Name: "cluster",
			},
		},
	}
	routers := []v2.Router{
		NewDirectResponseHeaderRouter(".*", response),
		NewDirectResponsePrefixRouter("/", response),
	}
	chains := []v2.FilterChain{
		util.NewFilterChain("proxyVirtualHost", proto, proto, routers),
	}
	listener := util.NewListener("proxyListener", addr, chains)
	return util.NewMOSNConfig([]v2.Listener{listener}, cmconfig)
}
func NewDirectResponseHeaderRouter(value string, response *v2.DirectResponseAction) v2.Router {
	header := v2.HeaderMatcher{Name: "service", Value: value}
	return v2.Router{
		RouterConfig: v2.RouterConfig{
			Match:          v2.RouterMatch{Headers: []v2.HeaderMatcher{header}},
			DirectResponse: response,
		},
	}
}
func NewDirectResponsePrefixRouter(prefix string, response *v2.DirectResponseAction) v2.Router {
	return v2.Router{
		RouterConfig: v2.RouterConfig{
			Match:          v2.RouterMatch{Prefix: prefix},
			DirectResponse: response,
		},
	}
}

// Proxy Mode is OK
type DirectResponseCase struct {
	Protocol   types.Protocol
	RPCClient  *util.RPCClient
	C          chan error
	T          *testing.T
	ClientAddr string
	Stop       chan struct{}
	status     int
	body       string
}

func NewDirectResponseCase(t *testing.T, proto types.Protocol, status int, body string, client *util.RPCClient) *DirectResponseCase {
	return &DirectResponseCase{
		Protocol:  proto,
		RPCClient: client,
		C:         make(chan error),
		T:         t,
		Stop:      make(chan struct{}),
		status:    status,
		body:      body,
	}
}

func (c *DirectResponseCase) StartProxy() {
	addr := util.CurrentMeshAddr()
	c.ClientAddr = addr
	resp := &v2.DirectResponseAction{
		StatusCode: c.status,
		Body:       c.body,
	}
	cfg := CreateDirectMeshProxy(addr, c.Protocol, resp)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Stop
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

func (c *DirectResponseCase) RunCase(n int, interval time.Duration) {
	var call func() error
	switch c.Protocol {
	case protocol.HTTP1:
		call = func() error {
			url := fmt.Sprintf("http://%s/", c.ClientAddr)
			resp, err := http.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != c.status {
				return fmt.Errorf("response status: %d", resp.StatusCode)
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if string(b) != c.body {
				return fmt.Errorf("response body: %s", string(b))
			}
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
			url := fmt.Sprintf("http://%s/", c.ClientAddr)
			resp, err := httpClient.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != c.status {
				return fmt.Errorf("response status: %d", resp.StatusCode)
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if string(b) != c.body {
				return fmt.Errorf("response body: %s", string(b))
			}
			return nil
		}
	case protocol.SofaRPC:
		client := c.RPCClient
		if err := client.Connect(c.ClientAddr); err != nil {
			c.C <- err
			return
		}
		if c.status != 200 {
			client.ExpectedStatus = sofarpc.RESPONSE_STATUS_UNKNOWN
		}
		defer client.Close()
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
		time.Sleep(interval)
	}
	c.C <- nil
}

func TestDirectResponse(t *testing.T) {
	testCases := []*DirectResponseCase{
		NewDirectResponseCase(t, protocol.HTTP1, 500, "", nil),
		NewDirectResponseCase(t, protocol.HTTP2, 500, "", nil),
		NewDirectResponseCase(t, protocol.HTTP1, 500, "internal error", nil),
		NewDirectResponseCase(t, protocol.HTTP2, 500, "internal error", nil),
		NewDirectResponseCase(t, protocol.HTTP1, 200, "testdata", nil),
		NewDirectResponseCase(t, protocol.HTTP2, 200, "testdata", nil),
		// RPC
		// FIXME: RPC cannot direct response success, code will be transfer
		NewDirectResponseCase(t, protocol.SofaRPC, 500, "", util.NewRPCClient(t, "directfail", util.Bolt1)),
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartProxy()
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d protocol %v test failed, error: %v\n", i, tc.Protocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d protocol %v hang\n", i, tc.Protocol)
		}
		close(tc.Stop)
		time.Sleep(time.Second)
	}
}
