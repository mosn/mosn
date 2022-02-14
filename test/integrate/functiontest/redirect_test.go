package functiontest

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"golang.org/x/net/http2"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

// Test Redirect

// Redirect ignores upstream cluster information and route rule config
func CreateRedirectMeshProxy(addr string, proto types.ProtocolName, redirect *v2.RedirectAction) *v2.MOSNConfig {
	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			{
				Name: "cluster",
			},
		},
	}
	routers := []v2.Router{
		NewRedirectResponsePrefixRouter("/", redirect),
	}
	chains := []v2.FilterChain{
		util.NewFilterChain("proxyVirtualHost", proto, proto, routers),
	}
	listener := util.NewListener("proxyListener", addr, chains)
	return util.NewMOSNConfig([]v2.Listener{listener}, cmconfig)
}

func NewRedirectResponsePrefixRouter(prefix string, redirect *v2.RedirectAction) v2.Router {
	return v2.Router{
		RouterConfig: v2.RouterConfig{
			Match:    v2.RouterMatch{Prefix: prefix},
			Redirect: redirect,
		},
	}
}

// Proxy Mode is OK
type RedirectResponseCase struct {
	Protocol   types.ProtocolName
	C          chan error
	T          *testing.T
	ClientAddr string
	Finish     chan bool
	redirect   *v2.RedirectAction
}

func NewRedirectResponseCase(t *testing.T, proto types.ProtocolName, redirect *v2.RedirectAction) *RedirectResponseCase {
	return &RedirectResponseCase{
		Protocol: proto,
		C:        make(chan error),
		T:        t,
		Finish:   make(chan bool),
		redirect: redirect,
	}
}

func (c *RedirectResponseCase) StartProxy() {
	addr := util.CurrentMeshAddr()
	c.ClientAddr = addr
	cfg := CreateRedirectMeshProxy(addr, c.Protocol, c.redirect)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		mesh.Close()
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) // wait server and mesh start
}

func (c *RedirectResponseCase) FinishCase() {
	c.Finish <- true
	<-c.Finish
}

func (c *RedirectResponseCase) RunCase(n int, interval time.Duration) {
	var (
		call func() error

		scheme = "http"
		path   = "/"
		host   = c.ClientAddr
	)
	if len(c.redirect.SchemeRedirect) != 0 {
		scheme = c.redirect.SchemeRedirect
	}
	if len(c.redirect.PathRedirect) != 0 {
		path = c.redirect.PathRedirect
	}
	if len(c.redirect.HostRedirect) != 0 {
		host = c.redirect.HostRedirect
	}
	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}
	expectedLocation := u.String()
	expectedCode := c.redirect.ResponseCode
	if expectedCode == 0 {
		expectedCode = http.StatusMovedPermanently
	}

	checkRedirect := func(req *http.Request, via []*http.Request) error {
		// do not follow redirects
		return http.ErrUseLastResponse
	}
	http1Client := http.Client{
		CheckRedirect: checkRedirect,
	}
	http2Client := http.Client{
		CheckRedirect: checkRedirect,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}

	switch c.Protocol {
	case protocol.HTTP1:
		call = func() error {
			url := fmt.Sprintf("http://%s/", c.ClientAddr)
			resp, err := http1Client.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			_, _ = io.Copy(ioutil.Discard, resp.Body)
			if resp.StatusCode != expectedCode {
				return fmt.Errorf("unexpected response status: %d", resp.StatusCode)
			}
			gotLocation := resp.Header.Get("location")
			if gotLocation != expectedLocation {
				return fmt.Errorf("unexpected location: %s", gotLocation)
			}
			return nil
		}
	case protocol.HTTP2:
		call = func() error {
			url := fmt.Sprintf("http://%s/", c.ClientAddr)
			resp, err := http2Client.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			_, _ = io.Copy(ioutil.Discard, resp.Body)
			if resp.StatusCode != expectedCode {
				return fmt.Errorf("unexpected response status: %d", resp.StatusCode)
			}
			gotLocation := resp.Header.Get("location")
			if gotLocation != expectedLocation {
				return fmt.Errorf("unexpected location: %s", gotLocation)
			}
			return nil
		}
	default:
		c.C <- fmt.Errorf("unknown protocol: %s", c.Protocol)
		return
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

func TestRedirect(t *testing.T) {
	testCases := []*RedirectResponseCase{
		NewRedirectResponseCase(t, protocol.HTTP1, &v2.RedirectAction{
			PathRedirect: "/foo",
		}),
		NewRedirectResponseCase(t, protocol.HTTP1, &v2.RedirectAction{
			ResponseCode: http.StatusTemporaryRedirect,
			HostRedirect: "foo.com",
		}),
		NewRedirectResponseCase(t, protocol.HTTP1, &v2.RedirectAction{
			SchemeRedirect: "https",
		}),
		NewRedirectResponseCase(t, protocol.HTTP2, &v2.RedirectAction{
			PathRedirect: "/foo",
		}),
		NewRedirectResponseCase(t, protocol.HTTP2, &v2.RedirectAction{
			ResponseCode: http.StatusTemporaryRedirect,
			HostRedirect: "foo.com",
		}),
		NewRedirectResponseCase(t, protocol.HTTP2, &v2.RedirectAction{
			SchemeRedirect: "https",
		}),
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
		case <-time.After(5 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d protocol %v hang\n", i, tc.Protocol)
		}
		tc.FinishCase()
	}
}
