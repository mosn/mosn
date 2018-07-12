package tests

import (
	"encoding/json"
	"fmt"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func CreateMeshConfig(addr string, filterChains []config.FilterChain, clusterManager config.ClusterManagerConfig) *config.MOSNConfig {
	listenerCfg := config.ListenerConfig{
		Name:         "testListener",
		Address:      addr,
		BindToPort:   true,
		LogPath:      "stdout",
		LogLevel:     "DEBUG",
		FilterChains: filterChains,
	}
	serverCfg := config.ServerConfig{
		DefaultLogPath:  "stdout",
		DefaultLogLevel: "DEBUG",
		Listeners:       []config.ListenerConfig{listenerCfg},
	}
	return &config.MOSNConfig{
		Servers:        []config.ServerConfig{serverCfg},
		ClusterManager: clusterManager,
	}
}

type cluster struct {
	name  string
	hosts []string
}

func CreateBasicClusterConfig(clusters []cluster) config.ClusterManagerConfig {
	clustersConfig := []config.ClusterConfig{}
	for _, c := range clusters {
		var vhosts []v2.Host
		for _, addr := range c.hosts {
			vhosts = append(vhosts, v2.Host{
				Address: addr,
				Weight:  100,
			})
		}
		cfg := config.ClusterConfig{
			Name:                 c.name,
			Type:                 "SIMPLE",
			LbType:               "LB_ROUNDROBIN",
			MaxRequestPerConn:    1024,
			ConnBufferLimitBytes: 16 * 1026,
			Hosts:                vhosts,
		}
		clustersConfig = append(clustersConfig, cfg)
	}
	return config.ClusterManagerConfig{
		Clusters: clustersConfig,
	}
}

//最简单的Mesh转发配置
//转发规则：DownStream和Upstream 匹配, Header有service
func CreateSimpleMeshConfig(addr string, hosts []string, downstream, upstream types.Protocol) *config.MOSNConfig {
	clusterName := "testCluster"
	cmconfig := CreateBasicClusterConfig([]cluster{
		cluster{name: clusterName, hosts: hosts},
	})
	//proxy
	header := v2.HeaderMatcher{Name: "service", Value: ".*"}
	routerV2 := v2.Router{
		Match: v2.RouterMatch{Headers: []v2.HeaderMatcher{header}},
		Route: v2.RouteAction{ClusterName: clusterName},
	}
	p := &v2.Proxy{
		DownstreamProtocol: string(downstream),
		UpstreamProtocol:   string(upstream),
		VirtualHosts: []*v2.VirtualHost{
			&v2.VirtualHost{Name: "testHost", Domains: []string{"*"}, Routers: []v2.Router{routerV2}},
		},
	}
	b, _ := json.Marshal(p)
	filterChains := make(map[string]interface{})
	json.Unmarshal(b, &filterChains)
	proxyconfig := []config.FilterChain{
		config.FilterChain{Filters: []config.FilterConfig{
			config.FilterConfig{Type: "proxy", Config: filterChains},
		}},
	}
	return CreateMeshConfig(addr, proxyconfig, cmconfig)
}

//HTTP 路由，基于HEADER和PATH
func CreateHTTPRouteConfig(addr string, hosts [][]string) *config.MOSNConfig {
	clusters := []cluster{}
	for idx, hh := range hosts {
		clusterName := fmt.Sprintf("cluster%d", idx)
		c := cluster{name: clusterName, hosts: hh}
		clusters = append(clusters, c)
	}
	cmconfig := CreateBasicClusterConfig(clusters)
	//proxy
	header1 := v2.HeaderMatcher{Name: "service", Value: "cluster1"}
	routerV2 := v2.Router{
		Match: v2.RouterMatch{Headers: []v2.HeaderMatcher{header1}},
		Route: v2.RouteAction{ClusterName: clusters[0].name},
	}
	header2 := v2.HeaderMatcher{Name: "service", Value: "cluster2"}
	routerV2_header := v2.Router{
		Match: v2.RouterMatch{Headers: []v2.HeaderMatcher{header2}},
		Route: v2.RouteAction{ClusterName: clusters[1].name},
	}
	routerV2_Path := v2.Router{
		Match: v2.RouterMatch{
			Headers: []v2.HeaderMatcher{header1},
			Path:    "/test",
		},
		Route: v2.RouteAction{ClusterName: clusters[1].name},
	}
	p := &v2.Proxy{
		DownstreamProtocol: string(protocol.Http1),
		UpstreamProtocol:   string(protocol.Http1),
		VirtualHosts: []*v2.VirtualHost{
			&v2.VirtualHost{Name: "testHost", Domains: []string{"*"}, Routers: []v2.Router{routerV2, routerV2_header, routerV2_Path}},
		},
	}
	b, _ := json.Marshal(p)
	filterChains := make(map[string]interface{})
	json.Unmarshal(b, &filterChains)
	proxyconfig := []config.FilterChain{
		config.FilterChain{Filters: []config.FilterConfig{
			config.FilterConfig{Type: "proxy", Config: filterChains},
		}},
	}
	return CreateMeshConfig(addr, proxyconfig, cmconfig)

}
