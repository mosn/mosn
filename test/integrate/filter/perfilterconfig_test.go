package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"mosn.io/pkg/buffer"

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

// Test and demo for use stream filter with `per_filter_config`
// register create filter factory func
func init() {
	api.RegisterStream("filterConfig", createConfigFilterFactory)
}

func createConfigFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	b, _ := json.Marshal(conf)
	m := make(map[string]string)
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &filterConfigFilterFactory{
		config: m,
	}, nil
}

// An implementation of api.StreamFilterChainFactory
type filterConfigFilterFactory struct {
	config map[string]string
}

func (f *filterConfigFilterFactory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewFilterConfigFilter(ctx, f.config)
	// ReceiverFilter, run the filter when receive a request from downstream
	// The FilterPhase can be BeforeRoute or AfterRoute, we use BeforeRoute in this demo
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
	// SenderFilter, run the filter when receive a response from upstream
	// In the demo, we are not implement this filter type
	// callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
}

// What filterConfigFilter do:
// the request will be passed only if the request headers contains key&value matched in the config
type filterConfigFilter struct {
	config  map[string]string
	handler api.StreamReceiverFilterHandler
}

// NewFilterConfigFilter returns a filterConfigFilter, the filterConfigFilter is an implementation of api.StreamReceiverFilter
// A Filter can implement both api.StreamReceiverFilter and api.StreamSenderFilter.
func NewFilterConfigFilter(ctx context.Context, config map[string]string) *filterConfigFilter {
	return &filterConfigFilter{
		config: config,
	}
}

func (f *filterConfigFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if route := f.handler.Route(); route != nil {
		f.ReadPerRouteConfig(route)
	}
	passed := true
CHECK:
	for k, v := range f.config {
		value, ok := headers.Get(k)
		if !ok || value != v {
			passed = false
			break CHECK
		}
	}
	if !passed {
		f.handler.SendHijackReply(403, headers)
		return api.StreamFilterStop
	}
	return api.StreamFilterContinue
}

func (f *filterConfigFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *filterConfigFilter) OnDestroy() {}

func (f *filterConfigFilter) ReadPerRouteConfig(route api.Route) {
	config, ok := route.RouteRule().PerFilterConfig()["filterConfig"]
	if !ok {
		config, ok = route.RouteRule().VirtualHost().PerFilterConfig()["filterConfig"]
		if !ok {
			return
		}
	}

	b, _ := json.Marshal(config)
	m := make(map[string]string)
	if err := json.Unmarshal(b, &m); err != nil {
		return
	}
	f.config = m
}

// mosn config with stream filter called inject
func createConfigFilterProxyMesh(addr string, hosts []string, proto types.ProtocolName) *v2.MOSNConfig {
	clusterName := "http_server"
	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			util.NewBasicCluster(clusterName, hosts),
		},
	}

	routerCfgName := "proxyVirtualHost"
	routerConfig := v2.RouterConfiguration{
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: routerCfgName,
		},
		VirtualHosts: []v2.VirtualHost{
			v2.VirtualHost{
				Name:    "test1",
				Domains: []string{"domain1.example"},
				Routers: []v2.Router{
					{
						RouterConfig: v2.RouterConfig{
							Match: v2.RouterMatch{
								Prefix: "/router",
							},
							Route: v2.RouteAction{
								RouterActionConfig: v2.RouterActionConfig{
									ClusterName: clusterName,
								},
							},
							PerFilterConfig: map[string]interface{}{
								"filterConfig": map[string]interface{}{
									"User": "routerUser",
								},
							},
						},
					},
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
						},
					},
				},
				PerFilterConfig: map[string]interface{}{
					"filterConfig": map[string]interface{}{
						"User": "vhostUser",
					},
				},
			},
			v2.VirtualHost{
				Name:    "test2",
				Domains: []string{"*"},
				Routers: []v2.Router{
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
						},
					},
				},
			},
		},
	}
	proxy := util.NewProxyFilter(routerCfgName, proto, proto)

	chainsProxy := make(map[string]interface{})
	b, _ := json.Marshal(proxy)
	json.Unmarshal(b, &chainsProxy)

	chainsConnMng := make(map[string]interface{})
	b2, _ := json.Marshal(routerConfig)
	json.Unmarshal(b2, &chainsConnMng)

	chains := []v2.FilterChain{
		v2.FilterChain{
			FilterChainConfig: v2.FilterChainConfig{
				Filters: []v2.Filter{
					v2.Filter{Type: "proxy", Config: chainsProxy},
					{Type: "connection_manager", Config: chainsConnMng},
				},
			},
		},
	}
	listener := util.NewListener("proxyListener", addr, chains)
	listener.StreamFilters = []v2.Filter{
		{
			Type: "filterConfig",
			Config: map[string]interface{}{
				"User": "globalUser",
			},
		},
	}
	cfg := util.NewMOSNConfig([]v2.Listener{listener}, cmconfig)
	return cfg
}

func TestConfigFilter(t *testing.T) {
	// NOTE: reroute maybe trigger panic and recover which still response a BadGateway
	// we should avoid the panic, so we needs to parse the output log
	lgfile := "/tmp/test_perfilterconfig.log"
	os.Remove(lgfile)
	// start a http server
	httpServer := util.NewHTTPServer(t, nil)
	httpServer.GoServe()
	httpAddr := httpServer.Addr()
	meshAddr := util.CurrentMeshAddr()
	cfg := createConfigFilterProxyMesh(meshAddr, []string{httpAddr}, protocol.HTTP1)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	defer mesh.Close()
	time.Sleep(2 * time.Second) // wait mosn start
	// reset the logger
	log.InitDefaultLogger(lgfile, log.DEBUG)

	// test User:globalUser
	testCase(t, "http://"+meshAddr, "any", "globalUser")
	// test User:vhostUser
	testCase(t, "http://"+meshAddr, "domain1.example", "vhostUser")
	// test User:routerUser
	testCase(t, "http://"+meshAddr+"/router", "domain1.example", "routerUser")

	httpServer.Close()
}

func testCase(t *testing.T, url, host, user string) {
	client := http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Host = host
	req.Header.Set("User", user)
	resp, err := client.Do(req)
	if err != nil {
		t.Error("test per filter config mosn error: ", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected http status code was success 200, but got %d", resp.StatusCode)
	}
	resp.Body.Close() // release
}
