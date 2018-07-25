/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tests

import (
	"fmt"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/internal/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// use different mesh port to avoid "port in used" error
var meshIndex uint32

func CurrentMeshAddr() string {
	var basic uint32 = 2044
	atomic.AddUint32(&meshIndex, 1)
	return fmt.Sprintf("127.0.0.1:%d", basic+meshIndex)
}

func CreateMeshConfig(addr string, filterChains []config.FilterChain, clusterManager config.ClusterManagerConfig) *config.MOSNConfig {
	listenerCfg := config.ListenerConfig{
		Name:         "testListener",
		Address:      addr,
		BindToPort:   true,
		LogPath:      "stdout",
		LogLevel:     "WARN",
		FilterChains: filterChains,
	}
	serverCfg := config.ServerConfig{
		DefaultLogPath:  "stdout",
		DefaultLogLevel: "WARN",
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
		var vhosts []config.HostConfig
		for _, addr := range c.hosts {
			vhosts = append(vhosts, config.HostConfig{
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

//simple mesh config, based on protocol only
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
	
	routerV2_http2 := v2.Router{
		Match: v2.RouterMatch{Prefix:"/"},
		Route: v2.RouteAction{ClusterName: clusterName},
	}
	
	p := &v2.Proxy{
		DownstreamProtocol: string(downstream),
		UpstreamProtocol:   string(upstream),
		VirtualHosts: []*v2.VirtualHost{
			&v2.VirtualHost{Name: "testHost", Domains: []string{"*"}, Routers: []v2.Router{routerV2,routerV2_http2}},
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

//HTTP router mesh config
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
	routerV2Header := v2.Router{
		Match: v2.RouterMatch{Headers: []v2.HeaderMatcher{header2}},
		Route: v2.RouteAction{ClusterName: clusters[1].name},
	}
	routerV2Path := v2.Router{
		Match: v2.RouterMatch{
			Headers: []v2.HeaderMatcher{header1},
			Path:    "/test.htm",
		},
		Route: v2.RouteAction{ClusterName: clusters[1].name},
	}
	p := &v2.Proxy{
		DownstreamProtocol: string(protocol.HTTP1),
		UpstreamProtocol:   string(protocol.HTTP1),
		VirtualHosts: []*v2.VirtualHost{
			//Notice that the order of router, the first successful matched is used
			&v2.VirtualHost{Name: "testHost", Domains: []string{"*"}, Routers: []v2.Router{routerV2Path, routerV2, routerV2Header}},
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
