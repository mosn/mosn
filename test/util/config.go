package util

import (
	"fmt"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
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

// mesh as a proxy , client and servre have same protocol
func CreateProxyMesh(addr string, hosts []string, proto types.Protocol) *config.MOSNConfig {
	clusterName := "proxyCluster"
	cmconfig := config.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			newBasicCluster(clusterName, hosts),
		},
	}
	routers := []v2.Router{
		newPrefixRouter(clusterName, "/"),
		newHeaderRouter(clusterName, ".*"),
	}
	chains := []v2.FilterChain{
		newFilterChain("proxyVirtualHost", proto, proto, routers),
	}
	listener := newListener("proxyListener", addr, chains)
	return newMOSNConfig([]v2.Listener{listener}, cmconfig)
}

// Mesh to Mesh
// clientaddr and serveraddr is mesh's addr
// appproto is client and server (not mesh) protocol
// meshproto is mesh's protocol
// hosts is server's addresses
func CreateMeshToMeshConfig(clientaddr string, serveraddr string, appproto types.Protocol, meshproto types.Protocol, hosts []string, tls bool) *config.MOSNConfig {
	downstreamCluster := "downstream"
	upstreamCluster := "upstream"
	downstreamRouters := []v2.Router{
		newPrefixRouter(downstreamCluster, "/"),
		newHeaderRouter(downstreamCluster, ".*"),
	}
	clientChains := []v2.FilterChain{
		newFilterChain("downstreamFilter", appproto, meshproto, downstreamRouters),
	}
	clientListener := newListener("downstreamListener", clientaddr, clientChains)
	upstreamRouters := []v2.Router{
		newPrefixRouter(upstreamCluster, "/"),
		newHeaderRouter(upstreamCluster, ".*"),
	}
	// client mesh -> cluster need tls
	meshClusterConfig := newBasicCluster(downstreamCluster, []string{serveraddr})
	//  server mesh listener need tls
	meshServerChain := newFilterChain("upstreamFilter", meshproto, appproto, upstreamRouters)
	if tls {
		tlsConf := v2.TLSConfig{
			Status:       true,
			CACert:       cacert,
			CertChain:    certchain,
			PrivateKey:   privatekey,
			EcdhCurves:   "P256",
			VerifyClient: true,
			ServerName:   "127.0.0.1",
		}
		meshClusterConfig.TLS = tlsConf
		meshServerChain.TLS = tlsConf
	}
	cmconfig := config.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			meshClusterConfig,
			newBasicCluster(upstreamCluster, hosts),
		},
	}
	serverChains := []v2.FilterChain{meshServerChain}
	serverListener := newListener("upstreamListener", serveraddr, serverChains)
	return newMOSNConfig([]v2.Listener{
		clientListener, serverListener,
	}, cmconfig)

}

// XProtocol must be mesh to mesh
// currently, support Path/Prefix is "/" only
func CreateXProtocolMesh(clientaddr string, serveraddr string, subprotocol string, hosts []string) *config.MOSNConfig {
	downstreamCluster := "downstream"
	upstreamCluster := "upstream"
	downstreamRouters := []v2.Router{
		newPrefixRouter(downstreamCluster, "/"),
	}
	clientChains := []v2.FilterChain{
		newXProtocolFilterChain("downstreamFilter", subprotocol, downstreamRouters),
	}
	clientListener := newListener("downstreamListener", clientaddr, clientChains)
	upstreamRouters := []v2.Router{
		newPrefixRouter(upstreamCluster, "/"),
	}
	meshClusterConfig := newBasicCluster(downstreamCluster, []string{serveraddr})
	meshServerChain := newXProtocolFilterChain("upstreamFilter", subprotocol, upstreamRouters)
	cmconfig := config.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			meshClusterConfig,
			newBasicCluster(upstreamCluster, hosts),
		},
	}
	serverChains := []v2.FilterChain{meshServerChain}
	serverListener := newListener("upstreamListener", serveraddr, serverChains)
	return newMOSNConfig([]v2.Listener{
		clientListener, serverListener,
	}, cmconfig)
}

// TLS Extension
type ExtendVerifyConfig struct {
	ExtendType   string
	VerifyConfig map[string]interface{}
}

func CreateTLSExtensionConfig(clientaddr string, serveraddr string, appproto types.Protocol, meshproto types.Protocol, hosts []string, ext *ExtendVerifyConfig) *config.MOSNConfig {
	downstreamCluster := "downstream"
	upstreamCluster := "upstream"
	downstreamRouters := []v2.Router{
		newPrefixRouter(downstreamCluster, "/"),
		newHeaderRouter(downstreamCluster, ".*"),
	}
	clientChains := []v2.FilterChain{
		newFilterChain("downstreamFilter", appproto, meshproto, downstreamRouters),
	}
	clientListener := newListener("downstreamListener", clientaddr, clientChains)
	upstreamRouters := []v2.Router{
		newPrefixRouter(upstreamCluster, "/"),
		newHeaderRouter(upstreamCluster, ".*"),
	}
	tlsConf := v2.TLSConfig{
		Status:       true,
		Type:         ext.ExtendType,
		VerifyClient: true,
		ExtendVerify: ext.VerifyConfig,
	}
	meshClusterConfig := newBasicCluster(downstreamCluster, []string{serveraddr})
	meshClusterConfig.TLS = tlsConf
	meshServerChain := newFilterChain("upstreamFilter", meshproto, appproto, upstreamRouters)
	meshServerChain.TLS = tlsConf
	cmconfig := config.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			meshClusterConfig,
			newBasicCluster(upstreamCluster, hosts),
		},
	}
	serverChains := []v2.FilterChain{meshServerChain}
	serverListener := newListener("upstreamListener", serveraddr, serverChains)
	return newMOSNConfig([]v2.Listener{
		clientListener, serverListener,
	}, cmconfig)

}

// TCP Proxy
func CreateTCPProxyConfig(meshaddr string, hosts []string, isRouteEntryMode bool) *config.MOSNConfig {
	clusterName := "cluster"
	cluster := clusterName
	if isRouteEntryMode {
		cluster = ""
	}
	tcpConfig := v2.TCPProxy{
		Cluster: cluster,
		Routes: []*v2.TCPRoute{
			&v2.TCPRoute{
				Cluster:          "cluster",
				SourceAddrs:      []v2.CidrRange{v2.CidrRange{Address: "127.0.0.1", Length: 24}},
				DestinationAddrs: []v2.CidrRange{v2.CidrRange{Address: "127.0.0.1", Length: 24}},
				SourcePort:       "1-65535",
				DestinationPort:  "1-65535",
			},
		},
	}
	chains := make(map[string]interface{})
	b, _ := json.Marshal(tcpConfig)
	json.Unmarshal(b, &chains)
	filterChains := []v2.FilterChain{
		{
			Filters: []v2.Filter{
				{Type: "tcp_proxy", Config: chains},
			},
		},
	}
	cmconfig := config.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			newBasicCluster(clusterName, hosts),
		},
	}
	listener := newListener("listener", meshaddr, filterChains)
	return newMOSNConfig([]v2.Listener{
		listener,
	}, cmconfig)
}

type WeightCluster struct {
	Name   string
	Hosts  []*WeightHost
	Weight uint32
}
type WeightHost struct {
	Addr   string
	Weight uint32
}

// mesh as a proxy , client and servre have same protocol
func CreateWeightProxyMesh(addr string, proto types.Protocol, clusters []*WeightCluster) *config.MOSNConfig {
	var clusterConfigs []v2.Cluster
	var weightClusters []v2.WeightedCluster
	for _, c := range clusters {
		clusterConfigs = append(clusterConfigs, newWeightedCluster(c.Name, c.Hosts))
		weightClusters = append(weightClusters, v2.WeightedCluster{
			Cluster: v2.ClusterWeight{
				ClusterWeightConfig: v2.ClusterWeightConfig{
					Name:   c.Name,
					Weight: c.Weight,
				},
			},
		})
	}
	cmconfig := config.ClusterManagerConfig{
		Clusters: clusterConfigs,
	}
	routers := []v2.Router{
		newHeaderWeightedRouter(weightClusters, ".*"),
	}
	chains := []v2.FilterChain{
		newFilterChain("proxyVirtualHost", proto, proto, routers),
	}
	listener := newListener("proxyListener", addr, chains)

	return newMOSNConfig([]v2.Listener{listener}, cmconfig)
}
