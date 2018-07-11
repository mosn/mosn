package tests

import (
	"encoding/json"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
)

func CreateSofaRpcConfig(addr string, hosts []string) *config.MOSNConfig {
	clusterName := "testCluster"
	filterChains := make(map[string]interface{})
	header := v2.HeaderMatcher{Name: "service", Value: ".*"}
	routerV2 := v2.Router{
		Match: v2.RouterMatch{Headers: []v2.HeaderMatcher{header}},
		Route: v2.RouteAction{ClusterName: clusterName},
	}
	p := &v2.Proxy{
		DownstreamProtocol: string(protocol.SofaRpc),
		UpstreamProtocol:   string(protocol.SofaRpc),
		VirtualHosts: []*v2.VirtualHost{
			&v2.VirtualHost{Name: "testHost", Domains: []string{"*"}, Routers: []v2.Router{routerV2}},
		},
	}
	b, _ := json.Marshal(p)
	json.Unmarshal(b, &filterChains)
	//MSON Config
	listenerCfg := config.ListenerConfig{
		Name:       "testListener",
		Address:    addr,
		BindToPort: true,
		LogPath:    "stdout",
		LogLevel:   "DEBUG",
		FilterChains: []config.FilterChain{
			config.FilterChain{Filters: []config.FilterConfig{
				config.FilterConfig{Type: "proxy", Config: filterChains},
			}},
		},
	}
	serverCfg := config.ServerConfig{
		DefaultLogPath:  "stdout",
		DefaultLogLevel: "DEBUG",
		Listeners:       []config.ListenerConfig{listenerCfg},
	}
	var vhosts []v2.Host
	for _, addr := range hosts {
		vhosts = append(vhosts, v2.Host{
			Address: addr,
			Weight:  100,
		})
	}
	clusterCfg := config.ClusterManagerConfig{
		Clusters: []config.ClusterConfig{
			config.ClusterConfig{
				Name:                 clusterName,
				Type:                 "SIMPLE",
				LbType:               "LB_ROUNDROBIN",
				MaxRequestPerConn:    1024,
				ConnBufferLimitBytes: 16 * 1026,
				Hosts:                vhosts},
		},
	}
	return &config.MOSNConfig{
		Servers:        []config.ServerConfig{serverCfg},
		ClusterManager: clusterCfg,
	}
}
