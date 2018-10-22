package util

import (
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// can set
var (
	MeshLogPath  = "stdout"
	MeshLogLevel = "WARN"
	StartRetry   = false
)

// Create Mesh Config
func newProxyFilter(routerfgname string, downstream, upstream types.Protocol) *v2.Proxy {
	return &v2.Proxy{
		DownstreamProtocol: string(downstream),
		UpstreamProtocol:   string(upstream),
		RouterConfigName:   routerfgname,
	}
}
func makeFilterChain(proxy *v2.Proxy, routers []v2.Router, cfgName string) v2.FilterChain {
	chains := make(map[string]interface{})
	b, _ := json.Marshal(proxy)
	json.Unmarshal(b, &chains)

	routerConfig := v2.RouterConfiguration{
		RouterConfigName: cfgName,
		VirtualHosts: []*v2.VirtualHost{
			&v2.VirtualHost{
				Name:    "test",
				Domains: []string{"*"},
				Routers: routers,
			},
		},
	}

	chainsConnMng := make(map[string]interface{})
	b2, _ := json.Marshal(routerConfig)
	json.Unmarshal(b2, &chainsConnMng)

	return v2.FilterChain{
		Filters: []v2.Filter{
			v2.Filter{Type: "proxy", Config: chains},
			{Type: "connection_manager", Config: chainsConnMng},
		},
	}
}

func newFilterChain(routerConfigName string, downstream, upstream types.Protocol, routers []v2.Router) v2.FilterChain {
	proxy := newProxyFilter(routerConfigName, downstream, upstream)

	return makeFilterChain(proxy, routers, routerConfigName)
}

func newXProtocolFilterChain(name string, subproto string, routers []v2.Router) v2.FilterChain {

	routerConfigName := "xprotocol_test_router_config_name"

	proxy := newProxyFilter(name, protocol.Xprotocol, protocol.Xprotocol)
	extendConfig := &v2.XProxyExtendConfig{
		SubProtocol: subproto,
	}
	extendMap := make(map[string]interface{})
	data, _ := json.Marshal(extendConfig)
	json.Unmarshal(data, &extendMap)
	proxy.ExtendConfig = extendMap
	return makeFilterChain(proxy, routers, routerConfigName)
}

func newBasicCluster(name string, hosts []string) v2.Cluster {
	var vhosts []v2.Host
	for _, addr := range hosts {
		vhosts = append(vhosts, v2.Host{
			HostConfig: v2.HostConfig{
				Address: addr,
			},
		})
	}
	return v2.Cluster{
		Name:                 name,
		ClusterType:          v2.SIMPLE_CLUSTER,
		LbType:               v2.LB_ROUNDROBIN,
		MaxRequestPerConn:    1024,
		ConnBufferLimitBytes: 16 * 1026,
		Hosts:                vhosts,
	}
}

func newWeightedCluster(name string, hosts []*WeightHost) v2.Cluster {
	var vhosts []v2.Host
	for _, host := range hosts {
		vhosts = append(vhosts, v2.Host{
			HostConfig: v2.HostConfig{
				Address: host.Addr,
				Weight:  host.Weight,
			},
		})
	}
	return v2.Cluster{
		Name:                 name,
		ClusterType:          v2.SIMPLE_CLUSTER,
		LbType:               v2.LB_ROUNDROBIN,
		MaxRequestPerConn:    1024,
		ConnBufferLimitBytes: 16 * 1026,
		Hosts:                vhosts,
	}
}

func newListener(name, addr string, chains []v2.FilterChain) v2.Listener {
	return v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:           name,
			AddrConfig:     addr,
			BindToPort:     true,
			LogPath:        MeshLogPath,
			LogLevelConfig: MeshLogLevel,
			FilterChains:   chains,
		},
	}
}

func newMOSNConfig(listeners []v2.Listener, clusterManager config.ClusterManagerConfig) *config.MOSNConfig {
	return &config.MOSNConfig{
		Servers: []config.ServerConfig{
			config.ServerConfig{
				DefaultLogPath:  MeshLogPath,
				DefaultLogLevel: MeshLogLevel,
				Listeners:       listeners,
			},
		},
		ClusterManager: clusterManager,
	}
}

// weighted cluster case
func newHeaderWeightedRouter(clusters []v2.WeightedCluster, value string) v2.Router {
	header := v2.HeaderMatcher{Name: "service", Value: value}
	return v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{Headers: []v2.HeaderMatcher{header}},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					WeightedClusters: clusters,
					RetryPolicy: &v2.RetryPolicy{
						RetryPolicyConfig: v2.RetryPolicyConfig{
							RetryOn:    StartRetry,
							NumRetries: 3,
						},
						RetryTimeout: 5 * time.Second,
					},
				},
			},
		},
	}
}

// common case
func newHeaderRouter(cluster string, value string) v2.Router {
	header := v2.HeaderMatcher{Name: "service", Value: value}
	return v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{Headers: []v2.HeaderMatcher{header}},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					ClusterName: cluster,
					RetryPolicy: &v2.RetryPolicy{
						RetryPolicyConfig: v2.RetryPolicyConfig{
							RetryOn:    StartRetry,
							NumRetries: 3,
						},
						RetryTimeout: 5 * time.Second,
					},
				},
			},
		},
	}
}
func newPrefixRouter(cluster string, prefix string) v2.Router {
	return v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{Prefix: prefix},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					ClusterName: cluster,
					RetryPolicy: &v2.RetryPolicy{
						RetryPolicyConfig: v2.RetryPolicyConfig{
							RetryOn:    StartRetry,
							NumRetries: 3,
						},
						RetryTimeout: 5 * time.Second,
					},
				},
			},
		},
	}
}
func newPathRouter(cluster string, path string) v2.Router {
	return v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{Path: path},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					ClusterName: cluster,
					RetryPolicy: &v2.RetryPolicy{
						RetryPolicyConfig: v2.RetryPolicyConfig{
							RetryOn:    StartRetry,
							NumRetries: 3,
						},
						RetryTimeout: 5 * time.Second,
					},
				},
			},
		},
	}
}

//tls config

const (
	cacert string = `-----BEGIN CERTIFICATE-----
MIIBVzCB/qADAgECAhBsIQij0idqnmDVIxbNRxRCMAoGCCqGSM49BAMCMBIxEDAO
BgNVBAoTB0FjbWUgQ28wIBcNNzAwMTAxMDAwMDAwWhgPMjA4NDAxMjkxNjAwMDBa
MBIxEDAOBgNVBAoTB0FjbWUgQ28wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARV
DG+YT6LzaR5r0Howj4/XxHtr3tJ+llqg9WtTJn0qMy3OEUZRfHdP9iuJ7Ot6rwGF
i6RXy1PlAurzeFzDqQY8ozQwMjAOBgNVHQ8BAf8EBAMCAqQwDwYDVR0TAQH/BAUw
AwEB/zAPBgNVHREECDAGhwR/AAABMAoGCCqGSM49BAMCA0gAMEUCIQDt9WA96LJq
VvKjvGhhTYI9KtbC0X+EIFGba9lsc6+ubAIgTf7UIuFHwSsxIVL9jI5RkNgvCA92
FoByjq7LS7hLSD8=
-----END CERTIFICATE-----
`
	certchain string = `-----BEGIN CERTIFICATE-----
MIIBdDCCARqgAwIBAgIQbCEIo9Inap5g1SMWzUcUQjAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYwMDAw
WjASMRAwDgYDVQQKEwdBY21lIENvMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
VQxvmE+i82kea9B6MI+P18R7a97SfpZaoPVrUyZ9KjMtzhFGUXx3T/Yriezreq8B
hYukV8tT5QLq83hcw6kGPKNQME4wDgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQG
CCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMA8GA1UdEQQIMAaHBH8A
AAEwCgYIKoZIzj0EAwIDSAAwRQIgO9xLIF1AoBsSMU6UgNE7svbelMAdUQgEVIhq
K3gwoeICIQCDC75Fa3XQL+4PZatS3OfG93XNFyno9koyn5mxLlDAAg==
-----END CERTIFICATE-----
`
	privatekey string = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICWksdaVL6sOu33VeohiDuQ3gP8xlQghdc+2FsWPSkrooAoGCCqGSM49
AwEHoUQDQgAEVQxvmE+i82kea9B6MI+P18R7a97SfpZaoPVrUyZ9KjMtzhFGUXx3
T/Yriezreq8BhYukV8tT5QLq83hcw6kGPA==
-----END EC PRIVATE KEY-----
`
)
