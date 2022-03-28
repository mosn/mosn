package conv

import (
	"testing"

	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/stretchr/testify/require"
)

func resetConfig() {
	envoyClusters = map[string]*envoy_config_cluster_v3.Cluster{}
	envoyListeners = map[string]*envoy_config_listener_v3.Listener{}
	envoyRoutes = map[string]*envoy_config_route_v3.RouteConfiguration{}
}

func TestEnvoyConfigUpdate(t *testing.T) {
	// Add Config

	clusters1 := []*envoy_config_cluster_v3.Cluster{
		{
			Name: "cluster1",
		},
		{
			Name: "cluster2",
		},
	}
	EnvoyConfigUpdateClusters(clusters1)
	require.Len(t, envoyClusters, 2)

	clusters2 := []*envoy_config_cluster_v3.Cluster{
		{
			Name: "cluster3",
		},
	}
	EnvoyConfigUpdateClusters(clusters2)
	require.Len(t, envoyClusters, 3)

	listeners1 := []*envoy_config_listener_v3.Listener{
		{
			// Without Name
			Address: &envoy_config_core_v3.Address{
				Address: &envoy_config_core_v3.Address_SocketAddress{
					SocketAddress: &envoy_config_core_v3.SocketAddress{
						Protocol: envoy_config_core_v3.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
							PortValue: 80,
						},
					},
				},
			},
		},
		{
			// With Name
			Name: "TestListenerName",
			Address: &envoy_config_core_v3.Address{
				Address: &envoy_config_core_v3.Address_SocketAddress{
					SocketAddress: &envoy_config_core_v3.SocketAddress{
						Protocol: envoy_config_core_v3.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
							PortValue: 8080,
						},
					},
				},
			},
		},
	}
	EnvoyConfigUpdateListeners(listeners1)
	require.Len(t, envoyListeners, 2)

	listeners2 := []*envoy_config_listener_v3.Listener{
		// Without Name
		{
			Address: &envoy_config_core_v3.Address{
				Address: &envoy_config_core_v3.Address_SocketAddress{
					SocketAddress: &envoy_config_core_v3.SocketAddress{
						Protocol: envoy_config_core_v3.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
							PortValue: 15090,
						},
					},
				},
			},
		},
	}
	EnvoyConfigUpdateListeners(listeners2)
	require.Len(t, envoyListeners, 3)

	routes1 := []*envoy_config_route_v3.RouteConfiguration{
		{
			Name: "route",
		},
	}
	EnvoyConfigUpdateRoutes(routes1)
	require.Len(t, envoyRoutes, 1)

	routes2 := []*envoy_config_route_v3.RouteConfiguration{
		{
			Name: "route2",
		},
	}
	EnvoyConfigUpdateRoutes(routes2)
	require.Len(t, envoyRoutes, 2)

	// Delete Config
	mosnClusters := ConvertClustersConfig(clusters1)
	for _, c := range mosnClusters {
		EnvoyConfigDeleteClusterByName(c.Name)
	}
	require.Len(t, envoyClusters, 1)

	EnvoyConfigDeleteListeners(listeners1)

	require.Len(t, envoyListeners, 1)

}
