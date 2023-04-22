package integrate

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	_ "mosn.io/mosn/pkg/filter/stream/mirror"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/stream/http2"
	"mosn.io/mosn/pkg/types"
	_ "mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

type MirrorCase struct {
	*TestCase
	Servers       []util.UpstreamServer
	upstreamAddrs []string
}

// Server Implement
type mirrorHandler struct {
	ReqCnt    int
	LocalAddr string
}

var (
	handlers = [3]mirrorHandler{}
)

func (h *mirrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.ReqCnt++
	w.WriteHeader(200)
}

func NewMirrorCase(t *testing.T, serverProto, meshProto types.ProtocolName) *MirrorCase {
	upstreamAddrs := [3]string{
		"127.0.0.1:9080",
		"127.0.0.1:9081",
		"127.0.0.1:9082",
	}
	var server []util.UpstreamServer
	switch serverProto {
	case protocol.HTTP1:
		for i := 0; i < len(upstreamAddrs); i++ {
			s := util.NewHTTPServer(t, &handlers[i])
			server = append(server, s)
		}
	case protocol.HTTP2:
		for i := 0; i < len(upstreamAddrs); i++ {
			s := util.NewUpstreamHTTP2(t, upstreamAddrs[i], &handlers[i])
			server = append(server, s)
		}
	}

	tc := NewTestCase(t, serverProto, meshProto, util.NewRPCServer(t, "", bolt.ProtocolName)) // empty server
	return &MirrorCase{
		TestCase: tc,
		Servers:  server,
	}
}

// mosn config with stream filter called inject
func createMirrorProxyMesh(addr string, hosts []string, proto types.ProtocolName) *v2.MOSNConfig {
	clusterName := "default_server"
	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			util.NewBasicCluster(clusterName, hosts),
		},
	}
	routers := []v2.Router{
		{
			RouterConfig: v2.RouterConfig{
				Match: v2.RouterMatch{
					Prefix: "/",
				},
				Route: v2.RouteAction{
					RouterActionConfig: v2.RouterActionConfig{
						ClusterName: clusterName,
					},
				},
				RequestMirrorPolicies: &v2.RequestMirrorPolicy{
					Cluster: clusterName,
					Percent: 100,
				},
			},
		},
	}
	chains := []v2.FilterChain{
		util.NewFilterChain("proxyVirtualHost", proto, proto, routers),
	}
	listener := util.NewListener("proxyListener", addr, chains)
	listener.StreamFilters = []v2.Filter{
		{
			Type: "mirror",
			Config: map[string]interface{}{
				"amplification": 1,
				"broadcast":     true,
			},
		},
	}
	cfg := util.NewMOSNConfig([]v2.Listener{listener}, cmconfig)
	return cfg
}

func (c *MirrorCase) StartProxy() {
	var addrs []string
	for i := 0; i < len(c.Servers); i++ {
		c.Servers[i].GoServe()
		addrs = append(addrs, c.Servers[i].Addr())
	}
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	fmt.Printf(" start mosn with protocol:%s, local addr:%s\n, upstreamaddr:%v",
		string(c.AppProtocol), clientMeshAddr, addrs)
	cfg := createMirrorProxyMesh(clientMeshAddr, addrs, c.AppProtocol)
	// cfg := util.CreateProxyMesh(clientMeshAddr, c.upstreamAddrs, c.AppProtocol)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		for i := 0; i < len(c.Servers); i++ {
			c.Servers[i].Close()
		}
		mesh.Close()
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

func TestMirror(t *testing.T) {
	testCases := []*MirrorCase{
		NewMirrorCase(t, protocol.HTTP1, protocol.HTTP1),
		//	NewMirrorCase(t, protocol.HTTP1, protocol.HTTP2),
		//	NewMirrorCase(t, protocol.HTTP2, protocol.HTTP1),
		NewMirrorCase(t, protocol.HTTP2, protocol.HTTP2),
	}
	caseCount := 2
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartProxy()
		// at least run twice
		go tc.RunCase(caseCount, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		time.Sleep(2 * time.Second)
		tc.FinishCase()
	}

	for i := 0; i < len(handlers); i++ {
		if handlers[i].ReqCnt != caseCount*len(testCases) {
			t.Errorf("broadcast request not received, reqcnt: %d, addr: %s", handlers[i].ReqCnt, handlers[i].LocalAddr)
		}
	}
}
