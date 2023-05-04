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

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	_ "mosn.io/mosn/pkg/filter/network/proxy"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/stream/http"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

// Test and demo for use stream filter to re route
// we configure a http route that only route http request with header(service: test)
// add we add a stream filter to auto inject header(service:test), so all http request will be routed
// we use POST request with body

// implementation of a new stream filter called inject
type injectConfigFactory struct{}

func (f *injectConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := &injectFilter{}
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
}

func createInjectFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	return &injectConfigFactory{}, nil
}

func init() {
	api.RegisterStream("inject", createInjectFactory)
}

type injectFilter struct {
	status  api.StreamFilterStatus
	handler api.StreamReceiverFilterHandler
}

func (f *injectFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *injectFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {
	if f.status == api.StreamFilterReMatchRoute {
		return api.StreamFilterContinue
	} else {
		f.inject()
		f.status = api.StreamFilterReMatchRoute
		return api.StreamFilterReMatchRoute
	}
}

func (f *injectFilter) OnDestroy() {}

func (f *injectFilter) inject() {
	headers := f.handler.GetRequestHeaders()
	headers.Set("service", "test")
	f.handler.SetRequestHeaders(headers)
}

// mosn config with stream filter called inject
func createInjectProxyMesh(addr string, hosts []string, proto types.ProtocolName) *v2.MOSNConfig {
	clusterName := "http_server"
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
	defer mesh.Close()
	time.Sleep(2 * time.Second) // wait mosn start
	// reset the logger
	log.InitDefaultLogger(lgfile, log.DEBUG)
	// make a http request
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post("http://"+meshAddr, "application/x-www-form-urlencoded", bytes.NewBufferString("testdata"))
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
	client = http.Client{Timeout: 5 * time.Second}
	errResp, err := client.Post("http://"+meshAddr, "application/x-www-form-urlencoded", bytes.NewBufferString("testdata"))
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
