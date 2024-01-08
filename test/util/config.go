package util

import (
	"fmt"
	"sync/atomic"

	jsoniter "github.com/json-iterator/go"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
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
func CreateProxyMesh(addr string, hosts []string, proto types.ProtocolName) *v2.MOSNConfig {
	clusterName := "proxyCluster"
	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			NewBasicCluster(clusterName, hosts),
		},
	}
	routers := []v2.Router{
		NewPrefixRouter(clusterName, "/"),
		NewHeaderRouter(clusterName, ".*"),
	}
	chains := []v2.FilterChain{
		NewFilterChain("proxyVirtualHost", proto, proto, routers),
	}
	listener := NewListener("proxyListener", addr, chains)
	return NewMOSNConfig([]v2.Listener{listener}, cmconfig)
}

// Mesh to Mesh
// clientaddr and serveraddr is mesh's addr
// appproto is client and server (not mesh) protocol
// meshproto is mesh's protocol
// hosts is server's addresses
func CreateMeshToMeshConfig(clientaddr string, serveraddr string, appproto types.ProtocolName, meshproto types.ProtocolName, hosts []string, tls bool) *v2.MOSNConfig {
	return CreateMeshToMeshConfigWithSub(clientaddr, serveraddr, appproto, meshproto, "", hosts, tls)
}

// Mesh to Mesh
// clientaddr and serveraddr is mesh's addr
// appproto is client and server (not mesh) protocol
// meshproto is mesh's protocol
// hosts is server's addresses
func CreateMeshToMeshConfigWithSub(clientaddr string, serveraddr string, appproto types.ProtocolName, meshproto types.ProtocolName, subproto types.ProtocolName, hosts []string, tls bool) *v2.MOSNConfig {
	downstreamCluster := "downstream"
	upstreamCluster := "upstream"
	downstreamRouters := []v2.Router{
		NewPrefixRouter(downstreamCluster, "/"),
		NewHeaderRouter(downstreamCluster, ".*"),
	}
	var clientChains []v2.FilterChain
	if subproto != "" {
		clientChains = []v2.FilterChain{
			NewFilterChainWithSub("downstreamFilter", appproto, meshproto, subproto, downstreamRouters),
		}
	} else {
		clientChains = []v2.FilterChain{
			NewFilterChain("downstreamFilter", appproto, meshproto, downstreamRouters),
		}
	}
	clientListener := NewListener("downstreamListener", clientaddr, clientChains)
	upstreamRouters := []v2.Router{
		NewPrefixRouter(upstreamCluster, "/"),
		NewHeaderRouter(upstreamCluster, ".*"),
	}
	// client mesh -> cluster need tls
	meshClusterConfig := NewBasicCluster(downstreamCluster, []string{serveraddr})
	//  server mesh listener need tls
	var meshServerChain v2.FilterChain
	if subproto != "" {
		meshServerChain = NewFilterChainWithSub("upstreamFilter", meshproto, appproto, subproto, upstreamRouters)
	} else {
		meshServerChain = NewFilterChain("upstreamFilter", meshproto, appproto, upstreamRouters)
	}
	if tls {
		tlsConf := v2.TLSConfig{
			Status:       true,
			CACert:       cacert,
			CertChain:    certchain,
			PrivateKey:   privatekey,
			EcdhCurves:   "P256",
			VerifyClient: true,
			//CipherSuites: "ECDHE-RSA-SM4-SM3:ECDHE-ECDSA-SM4-SM3",
			ServerName: "127.0.0.1",
		}
		meshClusterConfig.TLS = tlsConf
		meshServerChain.TLSContexts = []v2.TLSConfig{
			tlsConf,
		}
	}
	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			meshClusterConfig,
			NewBasicCluster(upstreamCluster, hosts),
		},
	}
	serverChains := []v2.FilterChain{meshServerChain}
	serverListener := NewListener("upstreamListener", serveraddr, serverChains)
	return NewMOSNConfig([]v2.Listener{
		clientListener, serverListener,
	}, cmconfig)
}

// XProtocol mesh as a proxy , client and servre have same protocol
func CreateXProtocolProxyMesh(addr string, hosts []string, proto types.ProtocolName) *v2.MOSNConfig {
	clusterName := "proxyCluster"
	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			NewBasicCluster(clusterName, hosts),
		},
	}
	routers := []v2.Router{
		NewPrefixRouter(clusterName, "/"),
		NewHeaderRouter(clusterName, ".*"),
	}
	chains := []v2.FilterChain{
		NewXProtocolFilterChain("proxyVirtualHost", proto, routers),
	}
	listener := NewListener("proxyListener", addr, chains)
	return NewMOSNConfig([]v2.Listener{listener}, cmconfig)
}

// XProtocol mesh to mesh
// currently, support Path/Prefix is "/" only
func CreateXProtocolMesh(clientaddr string, serveraddr string, subProtocol types.ProtocolName, hosts []string, tls bool) *v2.MOSNConfig {
	downstreamCluster := "downstream"
	upstreamCluster := "upstream"
	downstreamRouters := []v2.Router{
		NewPrefixRouter(downstreamCluster, "/"),
		NewHeaderRouter(downstreamCluster, ".*"),
	}
	clientChains := []v2.FilterChain{
		NewXProtocolFilterChain("xprotocol_test_router_config_name", subProtocol, downstreamRouters),
	}
	clientListener := NewListener("downstreamListener", clientaddr, clientChains)
	upstreamRouters := []v2.Router{
		NewPrefixRouter(upstreamCluster, "/"),
		NewHeaderRouter(upstreamCluster, ".*"),
	}
	meshClusterConfig := NewBasicCluster(downstreamCluster, []string{serveraddr})
	meshServerChain := NewXProtocolFilterChain("upstreamFilter", subProtocol, upstreamRouters)
	if tls {
		tlsConf := v2.TLSConfig{
			Status:       true,
			CACert:       cacert,
			CertChain:    certchain,
			PrivateKey:   privatekey,
			EcdhCurves:   "P256",
			VerifyClient: true,
			//CipherSuites: "ECDHE-RSA-SM4-SM3:ECDHE-ECDSA-SM4-SM3",
			ServerName: "127.0.0.1",
		}
		meshClusterConfig.TLS = tlsConf
		meshServerChain.TLSContexts = []v2.TLSConfig{
			tlsConf,
		}
	}
	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			meshClusterConfig,
			NewBasicCluster(upstreamCluster, hosts),
		},
	}
	serverChains := []v2.FilterChain{meshServerChain}
	serverListener := NewListener("upstreamListener", serveraddr, serverChains)
	return NewMOSNConfig([]v2.Listener{
		clientListener, serverListener,
	}, cmconfig)
}

// TLS Extension
type ExtendVerifyConfig struct {
	ExtendType   string
	VerifyConfig map[string]interface{}
}

func CreateTLSExtensionConfig(clientaddr string, serveraddr string, appproto types.ProtocolName, meshproto types.ProtocolName, hosts []string, ext *ExtendVerifyConfig) *v2.MOSNConfig {
	downstreamCluster := "downstream"
	upstreamCluster := "upstream"
	downstreamRouters := []v2.Router{
		NewPrefixRouter(downstreamCluster, "/"),
		NewHeaderRouter(downstreamCluster, ".*"),
	}
	clientChains := []v2.FilterChain{
		NewFilterChain("downstreamFilter", appproto, meshproto, downstreamRouters),
	}
	clientListener := NewListener("downstreamListener", clientaddr, clientChains)
	upstreamRouters := []v2.Router{
		NewPrefixRouter(upstreamCluster, "/"),
		NewHeaderRouter(upstreamCluster, ".*"),
	}
	tlsConf := v2.TLSConfig{
		Status:       true,
		Type:         ext.ExtendType,
		VerifyClient: true,
		ExtendVerify: ext.VerifyConfig,
	}
	meshClusterConfig := NewBasicCluster(downstreamCluster, []string{serveraddr})
	meshClusterConfig.TLS = tlsConf
	meshServerChain := NewFilterChain("upstreamFilter", meshproto, appproto, upstreamRouters)
	meshServerChain.TLSContexts = []v2.TLSConfig{
		tlsConf,
	}
	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			meshClusterConfig,
			NewBasicCluster(upstreamCluster, hosts),
		},
	}
	serverChains := []v2.FilterChain{meshServerChain}
	serverListener := NewListener("upstreamListener", serveraddr, serverChains)
	return NewMOSNConfig([]v2.Listener{
		clientListener, serverListener,
	}, cmconfig)

}

func CreateXprotocolTLSExtensionConfig(clientaddr string, serveraddr string, subProtocol types.ProtocolName, hosts []string, ext *ExtendVerifyConfig) *v2.MOSNConfig {
	downstreamCluster := "downstream"
	upstreamCluster := "upstream"
	downstreamRouters := []v2.Router{
		NewPrefixRouter(downstreamCluster, "/"),
		NewHeaderRouter(downstreamCluster, ".*"),
	}
	clientChains := []v2.FilterChain{
		NewXProtocolFilterChain("downstreamFilter", subProtocol, downstreamRouters),
	}
	clientListener := NewListener("downstreamListener", clientaddr, clientChains)
	upstreamRouters := []v2.Router{
		NewPrefixRouter(upstreamCluster, "/"),
		NewHeaderRouter(upstreamCluster, ".*"),
	}
	tlsConf := v2.TLSConfig{
		Status:       true,
		Type:         ext.ExtendType,
		VerifyClient: true,
		ExtendVerify: ext.VerifyConfig,
	}
	meshClusterConfig := NewBasicCluster(downstreamCluster, []string{serveraddr})
	meshClusterConfig.TLS = tlsConf
	meshServerChain := NewXProtocolFilterChain("upstreamFilter", subProtocol, upstreamRouters)
	meshServerChain.TLSContexts = []v2.TLSConfig{
		tlsConf,
	}
	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			meshClusterConfig,
			NewBasicCluster(upstreamCluster, hosts),
		},
	}
	serverChains := []v2.FilterChain{meshServerChain}
	serverListener := NewListener("upstreamListener", serveraddr, serverChains)
	return NewMOSNConfig([]v2.Listener{
		clientListener, serverListener,
	}, cmconfig)

}

// TCP Proxy
func CreateTCPProxyConfig(meshaddr string, hosts []string, isRouteEntryMode bool) *v2.MOSNConfig {
	clusterName := "cluster"
	cluster := clusterName
	if isRouteEntryMode {
		cluster = ""
	}
	tcpConfig := v2.StreamProxy{
		Cluster: cluster,
		Routes: []*v2.StreamRoute{
			&v2.StreamRoute{
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
			FilterChainConfig: v2.FilterChainConfig{
				Filters: []v2.Filter{
					{Type: "tcp_proxy", Config: chains},
				},
			},
		},
	}
	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			NewBasicCluster(clusterName, hosts),
		},
	}
	listener := NewListener("listener", meshaddr, filterChains)
	return NewMOSNConfig([]v2.Listener{
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
func CreateXWeightProxyMesh(addr string, proto types.ProtocolName, clusters []*WeightCluster) *v2.MOSNConfig {
	var clusterConfigs []v2.Cluster
	var weightClusters []v2.WeightedCluster
	for _, c := range clusters {
		clusterConfigs = append(clusterConfigs, NewWeightedCluster(c.Name, c.Hosts))
		weightClusters = append(weightClusters, v2.WeightedCluster{
			Cluster: v2.ClusterWeight{
				ClusterWeightConfig: v2.ClusterWeightConfig{
					Name:   c.Name,
					Weight: c.Weight,
				},
			},
		})
	}
	cmconfig := v2.ClusterManagerConfig{
		Clusters: clusterConfigs,
	}
	routers := []v2.Router{
		NewHeaderWeightedRouter(weightClusters, ".*"),
	}
	chains := []v2.FilterChain{
		NewXProtocolFilterChain("proxyVirtualHost", proto, routers),
	}
	listener := NewListener("proxyListener", addr, chains)

	return NewMOSNConfig([]v2.Listener{listener}, cmconfig)
}
