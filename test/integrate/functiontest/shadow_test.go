package functiontest

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/codec"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
	"golang.org/x/net/http2"
)

// Shadow config create
func CreateShadowMeshProxy(addr string, server string, shadow string, proto types.Protocol) *config.MOSNConfig {
	clusterName := "server"
	shadowClusterName := "shadow"
	cmconfig := config.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			util.NewBasicCluster(clusterName, []string{server}),
			util.NewBasicCluster(shadowClusterName, []string{shadow}),
		},
	}
	routers := []v2.Router{
		NewShadowPrefixRouter(clusterName, shadowClusterName, "/"),
		NewShadowHeaderRouter(clusterName, shadowClusterName, ".*"),
	}
	chains := []v2.FilterChain{
		util.NewFilterChain("proxyVirtualHost", proto, proto, routers),
	}
	listener := util.NewListener("proxyListener", addr, chains)
	return util.NewMOSNConfig([]v2.Listener{listener}, cmconfig)
}

// only upstream mosn have shadow
func CreateShadowMesh(downaddr, upaddr string, server string, shadow string, appproto, meshproto types.Protocol) *config.MOSNConfig {
	// downstream
	downstreamCluster := "downstream"
	downstreamRouters := []v2.Router{
		util.NewPrefixRouter(downstreamCluster, "/"),
		util.NewHeaderRouter(downstreamCluster, ".*"),
	}
	clientChains := []v2.FilterChain{
		util.NewFilterChain("downstreamFilter", appproto, meshproto, downstreamRouters),
	}
	clientListener := util.NewListener("downstreamListener", downaddr, clientChains)
	clientCluster := util.NewBasicCluster(downstreamCluster, []string{upaddr})
	// upstream
	upstreamCluster := "upstream"
	shadowClusterName := "shadow"
	upstreamRouters := []v2.Router{
		NewShadowPrefixRouter(upstreamCluster, shadowClusterName, "/"),
		NewShadowHeaderRouter(upstreamCluster, shadowClusterName, ".*"),
	}
	serverChains := []v2.FilterChain{
		util.NewFilterChain("upstreamFilter", meshproto, appproto, upstreamRouters),
	}
	serverListener := util.NewListener("upstreamListener", upaddr, serverChains)
	cmconfig := config.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			clientCluster,
			util.NewBasicCluster(upstreamCluster, []string{server}),
			util.NewBasicCluster(shadowClusterName, []string{shadow}),
		},
	}
	return util.NewMOSNConfig([]v2.Listener{
		clientListener,
		serverListener,
	}, cmconfig)
}

func NewShadowPrefixRouter(cluster, shadow string, prefix string) v2.Router {
	return v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{Prefix: prefix},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					ClusterName: cluster,
					RetryPolicy: &v2.RetryPolicy{
						RetryPolicyConfig: v2.RetryPolicyConfig{
							RetryOn:    false,
							NumRetries: 3,
						},
						RetryTimeout: 5 * time.Second,
					},
					ShadowPolicy: &v2.ShadowPolicyConfig{
						ShadowClusterName: shadow,
						ShadowRatio:       100,
					},
				},
			},
		},
	}
}

func NewShadowHeaderRouter(cluster, shadow string, value string) v2.Router {
	header := v2.HeaderMatcher{Name: "service", Value: value}
	return v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{Headers: []v2.HeaderMatcher{header}},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					ClusterName: cluster,
					RetryPolicy: &v2.RetryPolicy{
						RetryPolicyConfig: v2.RetryPolicyConfig{
							RetryOn:    false,
							NumRetries: 3,
						},
						RetryTimeout: 5 * time.Second,
					},
					ShadowPolicy: &v2.ShadowPolicyConfig{
						ShadowClusterName: shadow,
						ShadowRatio:       100,
					},
				},
			},
		},
	}
}

type ShadowHTTPHandler struct {
	t *testing.T
}

const (
	queryPath = "/test_shadow"
	postData  = "test_shadow"
)

// Shadow server try to response an error, expected mosn ignore it
// Shadow server will verify receive data, expected same as post
func (h *ShadowHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusBadRequest)
	// verify data
	if r.URL.String() != queryPath {
		h.t.Error("shadow receive query path not expected")
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.t.Errorf("read body failed", err)
	}
	if string(b) != postData {
		h.t.Error("shadow receive data not expected")
	}
}

// Shadow serve try to response an error, expected mosn ignore it
// Shadow server will verify receive data, expected same as post
func ServeShadowBoltV1(t *testing.T, conn net.Conn) {
	response := func(iobuf types.IoBuffer) ([]byte, bool) {
		cmd, _ := codec.BoltV1.GetDecoder().Decode(nil, iobuf)
		if cmd == nil {
			return nil, false
		}
		if req, ok := cmd.(*sofarpc.BoltRequestCommand); ok {
			if string(req.Content) != postData {
				t.Error("shadow receive data not expected")
			}
			resp := util.BuildBoltV1Response(req)
			resp.ResponseStatus = sofarpc.RESPONSE_STATUS_SERVER_EXCEPTION
			iobufresp, err := codec.BoltV1.GetEncoder().EncodeHeaders(nil, resp)
			if err != nil {
				t.Errorf("Build response error: %v\n", err)
				return nil, true
			}
			return iobufresp.Bytes(), true
		}
		return nil, true
	}
	util.ServeSofaRPC(t, conn, response)

}

func NewShadowBoltServer(t *testing.T, shadowaddr string) util.UpstreamServer {
	return util.RPCServer{
		Client:         util.NewRPCClient(t, "rpcClient", util.Bolt1),
		Name:           "shadow",
		UpstreamServer: util.NewUpstreamServer(t, shadowaddr, ServeShadowBoltV1),
	}
}

type ShadowCase struct {
	AppProtocol  types.Protocol
	MeshProtocol types.Protocol
	C            chan error
	T            *testing.T
	AppServer    util.UpstreamServer
	ShadowServer util.UpstreamServer
	ClientAddr   string
	Stop         chan struct{}
}

func NewShadowCase(t *testing.T, app, mesh types.Protocol, server, shadow util.UpstreamServer) *ShadowCase {
	return &ShadowCase{
		AppProtocol:  app,
		MeshProtocol: mesh,
		C:            make(chan error),
		T:            t,
		AppServer:    server,
		ShadowServer: shadow,
		Stop:         make(chan struct{}),
	}
}

func (c *ShadowCase) run(cfg *config.MOSNConfig) {
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Stop
		c.AppServer.Close()
		c.ShadowServer.Close()
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start

}

func StartProxy(c *ShadowCase) {
	c.AppServer.GoServe()
	c.ShadowServer.GoServe()
	appAddr := c.AppServer.Addr()
	shadowAddr := c.ShadowServer.Addr()
	c.ClientAddr = util.CurrentMeshAddr()
	cfg := CreateShadowMeshProxy(c.ClientAddr, appAddr, shadowAddr, c.AppProtocol)
	c.run(cfg)
}

func Start(c *ShadowCase) {
	c.AppServer.GoServe()
	c.ShadowServer.GoServe()
	appAddr := c.AppServer.Addr()
	shadowAddr := c.ShadowServer.Addr()
	c.ClientAddr = util.CurrentMeshAddr()
	cfg := CreateShadowMesh(c.ClientAddr, util.CurrentMeshAddr(), appAddr, shadowAddr, c.AppProtocol, c.MeshProtocol)
	c.run(cfg)
}

func (c *ShadowCase) RunCase(n int, interval time.Duration) {
	var call func() error
	switch c.AppProtocol {
	case protocol.HTTP1:
		call = func() error {
			url := fmt.Sprintf("http://%s%s", c.ClientAddr, queryPath)
			resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(postData))
			//resp, err := http.Get(url)
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
			url := fmt.Sprintf("http://%s%s", c.ClientAddr, queryPath)
			resp, err := httpClient.Post(url, "application/x-www-form-urlencoded", strings.NewReader(postData))
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
		if err := client.Connect(c.ClientAddr); err != nil {
			c.C <- err
			return
		}
		defer client.Close()
		call = func() error {
			client.SendRequestWithData(postData)
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

func testCases(t *testing.T) []*ShadowCase {
	serverAddr := "127.0.0.1:8080"
	shadowAddr := "127.0.0.1:8081"
	return []*ShadowCase{
		NewShadowCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil), util.NewHTTPServer(t, &ShadowHTTPHandler{t})),
		NewShadowCase(t, protocol.HTTP1, protocol.HTTP2, util.NewHTTPServer(t, nil), util.NewHTTPServer(t, &ShadowHTTPHandler{t})),
		NewShadowCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, serverAddr, nil), util.NewUpstreamHTTP2(t, shadowAddr, &ShadowHTTPHandler{t})),
		NewShadowCase(t, protocol.HTTP2, protocol.HTTP1, util.NewUpstreamHTTP2(t, serverAddr, nil), util.NewUpstreamHTTP2(t, shadowAddr, &ShadowHTTPHandler{t})),
		NewShadowCase(t, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServer(t, serverAddr, util.Bolt1), NewShadowBoltServer(t, shadowAddr)),
		NewShadowCase(t, protocol.SofaRPC, protocol.HTTP1, util.NewRPCServer(t, serverAddr, util.Bolt1), NewShadowBoltServer(t, shadowAddr)),
		NewShadowCase(t, protocol.SofaRPC, protocol.HTTP2, util.NewRPCServer(t, serverAddr, util.Bolt1), NewShadowBoltServer(t, shadowAddr)),
	}
}

func runCase(t *testing.T, cases []*ShadowCase, foo func(tc *ShadowCase)) {
	for i, tc := range cases {
		t.Logf("start case #%d\n", i)
		foo(tc)
		go tc.RunCase(10, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("%v [ERROR MESSAGE] #%d %v to mesh %v test failed, error: %v\n", time.Now(), i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(60 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		close(tc.Stop)
		time.Sleep(time.Second)
	}
}

func TestShadowProxy(t *testing.T) {
	runCase(t, testCases(t), StartProxy)
}

func TestShadow(t *testing.T) {
	runCase(t, testCases(t), Start)
}
