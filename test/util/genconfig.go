package util

import (
	"net"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

// can set
var (
	MeshLogPath  = "stdout"
	MeshLogLevel = "INFO"
	StartRetry   = false
)

// Create Mesh Config
func NewProxyFilter(routerfgname string, downstream, upstream types.ProtocolName) *v2.Proxy {
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
		RouterConfigurationConfig: v2.RouterConfigurationConfig{
			RouterConfigName: cfgName,
		},
		VirtualHosts: []v2.VirtualHost{
			v2.VirtualHost{
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
		FilterChainConfig: v2.FilterChainConfig{
			Filters: []v2.Filter{
				v2.Filter{Type: "proxy", Config: chains},
				{Type: "connection_manager", Config: chainsConnMng},
			},
		},
	}
}

func NewFilterChain(routerConfigName string, downstream, upstream types.ProtocolName, routers []v2.Router) v2.FilterChain {
	proxy := NewProxyFilter(routerConfigName, downstream, upstream)

	return makeFilterChain(proxy, routers, routerConfigName)
}

func NewFilterChainWithSub(routerConfigName string, downstream, upstream types.ProtocolName, subProtocol types.ProtocolName, routers []v2.Router) v2.FilterChain {
	proxy := NewProxyFilter(routerConfigName, downstream, upstream)
	proxy.ExtendConfig = map[string]interface{}{
		"sub_protocol": string(subProtocol),
	}
	return makeFilterChain(proxy, routers, routerConfigName)
}

func NewXProtocolFilterChain(name string, subProtocol types.ProtocolName, routers []v2.Router) v2.FilterChain {
	proxy := NewProxyFilter(name, types.ProtocolName("X"), types.ProtocolName("X"))
	proxy.ExtendConfig = map[string]interface{}{
		"sub_protocol": string(subProtocol),
	}
	return makeFilterChain(proxy, routers, name)
}

func NewBasicCluster(name string, hosts []string) v2.Cluster {
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

func NewWeightedCluster(name string, hosts []*WeightHost) v2.Cluster {
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

func NewListener(name, addr string, chains []v2.FilterChain) v2.Listener {
	nw := "tcp"
	if _, err := net.ResolveTCPAddr(nw, addr); err != nil {
		nw = "unix"
	}
	return v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:         name,
			AddrConfig:   addr,
			BindToPort:   true,
			FilterChains: chains,
			Network:      nw,
		},
	}
}

func NewMOSNConfig(listeners []v2.Listener, clusterManager v2.ClusterManagerConfig) *v2.MOSNConfig {
	return &v2.MOSNConfig{
		Servers: []v2.ServerConfig{
			v2.ServerConfig{
				DefaultLogPath:  MeshLogPath,
				DefaultLogLevel: MeshLogLevel,
				Listeners:       listeners,
			},
		},
		ClusterManager: clusterManager,
	}
}

// weighted cluster case
func NewHeaderWeightedRouter(clusters []v2.WeightedCluster, value string) v2.Router {
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
func NewHeaderRouter(cluster string, value string) v2.Router {
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
func NewPrefixRouter(cluster string, prefix string) v2.Router {
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
func NewPathRouter(cluster string, path string) v2.Router {
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
