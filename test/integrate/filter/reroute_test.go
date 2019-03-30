package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/filter"
	_ "github.com/alipay/sofa-mosn/pkg/filter/network/proxy"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
)

// Test and demo for use stream filter to re route
// we configure a http route that only route http request with header(service: test)
// add we add a stream filter to auto inject header(service:test), so all http request will be routed
// we use POST request with body

// implementation of a new stream filter called inject
type injectConfigFactory struct{}

func (f *injectConfigFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	filter := &injectFilter{}
	callbacks.AddStreamReceiverFilter(filter, types.DownFilterAfterRoute)
}

func createInjectFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	return &injectConfigFactory{}, nil
}

func init() {
	filter.RegisterStream("inject", createInjectFactory)
}

type injectFilter struct {
	handler types.StreamReceiverFilterHandler
}

func (f *injectFilter) SetReceiveFilterHandler(handler types.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *injectFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) types.StreamFilterStatus {
	f.inject()
	return types.StreamFilterReMatchRoute
}

func (f *injectFilter) OnDestroy() {}

func (f *injectFilter) inject() {
	headers := f.handler.GetRequestHeaders()
	headers.Set("service", "test")
	f.handler.SetRequestHeaders(headers)
}

// mosn config with stream filter called inject
func createInjectProxyMesh(addr string, hosts []string, proto types.Protocol) *config.MOSNConfig {
	clusterName := "http_server"
	cmconfig := config.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			util.NewBasicCluster(clusterName, hosts),
		},
	}
	routers := []v2.Router{
		{
			RouterConfig: v2.RouterConfig{
				Match: v2.RouterMatch{
					Prefix: "/",
					Headers: []v2.HeaderMatcher{
						{
							Name:  "service",
							Value: "test",
						},
					},
				},
				Route: v2.RouteAction{
					RouterActionConfig: v2.RouterActionConfig{
						ClusterName: clusterName,
					},
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
			Type:   "inject",
			Config: map[string]interface{}{},
		},
	}
	cfg := util.NewMOSNConfig([]v2.Listener{listener}, cmconfig)
	return cfg
}

func TestReRoute(t *testing.T) {
	// NOTE: reroute maybe trigger panic and recover which still response a BadGateway
	// we should avoid the panic, so we needs to parse the output log
	lgfile := "/tmp/test_reroute.log"
	os.Remove(lgfile)
	// start a http server
	httpServer := util.NewHTTPServer(t, nil)
	httpServer.GoServe()
	httpAddr := httpServer.Addr()
	meshAddr := util.CurrentMeshAddr()
	cfg := createInjectProxyMesh(meshAddr, []string{httpAddr}, protocol.HTTP1)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	time.Sleep(2 * time.Second) // wait mosn start
	// reset the logger
	log.InitDefaultLogger(lgfile, log.ERROR)
	// make a http request
	resp, err := http.Post("http://"+meshAddr, "application/x-www-form-urlencoded", bytes.NewBufferString("testdata"))
	if err != nil {
		t.Error("rerquest reroute mosn error: ", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		t.Error("request reroute mosn is not OK, ", resp.StatusCode)
	}
	resp.Body.Close() // release
	// stop the server and make a request, expected get a error response from mosn
	httpServer.Close()
	errResp, err := http.Post("http://"+meshAddr, "application/x-www-form-urlencoded", bytes.NewBufferString("testdata"))
	if err != nil {
		t.Error("rerquest reroute mosn error: ", err)
		return
	}
	if errResp.StatusCode != http.StatusBadGateway {
		t.Error("request reroute mosn is not 502, ", errResp.StatusCode)
	}
	errResp.Body.Close()
	data, err := ioutil.ReadFile(lgfile)
	if err != nil {
		t.Error("read log file failed")
		return
	}
	if strings.Contains(string(data), "panic") {
		t.Error("mosn have panic log")
	}

}
