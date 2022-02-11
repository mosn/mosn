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

package conv

import (
	"fmt"
	"net"

	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
)

func ConvertClustersConfig(xdsClusters []*envoy_config_cluster_v3.Cluster) []*v2.Cluster {
	if xdsClusters == nil {
		return nil
	}
	clusters := make([]*v2.Cluster, 0, len(xdsClusters))
	for _, xdsCluster := range xdsClusters {
		// If tls is specified in DestinationRule, tls config will be in TransportSocket
		xdsTLSContext := xdsCluster.GetTransportSocket()
		// Otherwise tls config will be in TransportSocketMatches
		if xdsTLSContext == nil {
			for _, m := range xdsCluster.TransportSocketMatches {
				if m.GetTransportSocket().Name == wellknown.TransportSocketTls {
					xdsTLSContext = m.GetTransportSocket()
				}
			}
		}
		cluster := &v2.Cluster{
			Name:                 xdsCluster.GetName(),
			ClusterType:          convertClusterType(xdsCluster.GetType()),
			LbType:               convertLbPolicy(xdsCluster.GetType(), xdsCluster.GetLbPolicy()),
			LBSubSetConfig:       convertLbSubSetConfig(xdsCluster.GetLbSubsetConfig()),
			MaxRequestPerConn:    xdsCluster.GetMaxRequestsPerConnection().GetValue(),
			ConnBufferLimitBytes: xdsCluster.GetPerConnectionBufferLimitBytes().GetValue(),
			HealthCheck:          convertHealthChecks(xdsCluster.GetHealthChecks()),
			CirBreThresholds:     convertCircuitBreakers(xdsCluster.GetCircuitBreakers()),
			ConnectTimeout:       &api.DurationConfig{Duration: ConvertDuration(xdsCluster.GetConnectTimeout())},
			// OutlierDetection:     convertOutlierDetection(xdsCluster.GetOutlierDetection()),
			Spec:     convertSpec(xdsCluster),
			TLS:      convertTLS(xdsTLSContext),
			LbConfig: convertLbConfig(xdsCluster.LbConfig),
		}

		if ass := xdsCluster.GetLoadAssignment(); ass != nil {
			for _, endpoints := range ass.Endpoints {
				hosts := ConvertEndpointsConfig(endpoints)
				cluster.Hosts = append(cluster.Hosts, hosts...)
			}
		}

		clusters = append(clusters, cluster)
	}

	return clusters
}

// TODO support more LB converter
func convertLbConfig(config interface{}) v2.IsCluster_LbConfig {
	switch config.(type) {
	case *envoy_config_cluster_v3.Cluster_LeastRequestLbConfig:
		return &v2.LeastRequestLbConfig{ChoiceCount: config.(*envoy_config_cluster_v3.Cluster_LeastRequestLbConfig).ChoiceCount.GetValue()}
	default:
		return nil
	}
}

func ConvertEndpointsConfig(xdsEndpoint *envoy_config_endpoint_v3.LocalityLbEndpoints) []v2.Host {
	if xdsEndpoint == nil {
		return nil
	}
	hosts := make([]v2.Host, 0, len(xdsEndpoint.GetLbEndpoints()))
	for _, xdsHost := range xdsEndpoint.GetLbEndpoints() {
		var address string
		xh, _ := xdsHost.GetHostIdentifier().(*envoy_config_endpoint_v3.LbEndpoint_Endpoint)
		if xdsAddress, ok := xh.Endpoint.GetAddress().Address.(*envoy_config_core_v3.Address_SocketAddress); ok {
			if xdsPort, ok := xdsAddress.SocketAddress.GetPortSpecifier().(*envoy_config_core_v3.SocketAddress_PortValue); ok {
				address = fmt.Sprintf("%s:%d", xdsAddress.SocketAddress.GetAddress(), xdsPort.PortValue)
			} else if xdsPort, ok := xdsAddress.SocketAddress.GetPortSpecifier().(*envoy_config_core_v3.SocketAddress_NamedPort); ok {
				address = fmt.Sprintf("%s:%s", xdsAddress.SocketAddress.GetAddress(), xdsPort.NamedPort)
			} else {
				log.DefaultLogger.Warnf("unsupported port type")
				continue
			}

		} else if xdsAddress, ok := xh.Endpoint.GetAddress().Address.(*envoy_config_core_v3.Address_Pipe); ok {
			// Unix Domain Socket path.
			address = "unix:" + xdsAddress.Pipe.GetPath()
		} else {
			log.DefaultLogger.Warnf("unsupported address type")
			continue
		}
		host := v2.Host{
			HostConfig: v2.HostConfig{
				Address: address,
			},
			MetaData: convertMeta(xdsHost.Metadata),
		}

		weight := xdsHost.GetLoadBalancingWeight().GetValue()
		if weight < configmanager.MinHostWeight {
			weight = configmanager.MinHostWeight
		} else if weight > configmanager.MaxHostWeight {
			weight = configmanager.MaxHostWeight
		}
		host.Weight = weight
		// enable tls only when transport_socket_match matched
		// TODO check match
		if _, ok := host.MetaData["envoy.transport_socket_match"]; !ok {
			host.TLSDisable = true
		}

		// enable tls only when transport_socket_match matched
		// TODO check match
		if _, ok := host.MetaData["envoy.transport_socket_match"]; !ok {
			host.TLSDisable = true
		}

		hosts = append(hosts, host)
	}
	return hosts
}

func convertWeightedClusters(xdsWeightedClusters *envoy_config_route_v3.WeightedCluster) []v2.WeightedCluster {
	if xdsWeightedClusters == nil {
		return nil
	}
	weightedClusters := make([]v2.WeightedCluster, 0, len(xdsWeightedClusters.GetClusters()))
	for _, cluster := range xdsWeightedClusters.GetClusters() {
		weightedCluster := v2.WeightedCluster{
			Cluster: convertWeightedCluster(cluster),
			//RuntimeKeyPrefix: xdsWeightedClusters.GetRuntimeKeyPrefix(),
		}
		weightedClusters = append(weightedClusters, weightedCluster)
	}
	return weightedClusters
}

func convertHashPolicy(hashPolicy []*envoy_config_route_v3.RouteAction_HashPolicy) []v2.HashPolicy {
	hpReturn := make([]v2.HashPolicy, 0, len(hashPolicy))
	for _, p := range hashPolicy {
		if header := p.GetHeader(); header != nil {
			hpReturn = append(hpReturn, v2.HashPolicy{
				Header: &v2.HeaderHashPolicy{
					Key: header.HeaderName,
				},
			})

			continue
		}

		if cookieConfig := p.GetCookie(); cookieConfig != nil {
			hpReturn = append(hpReturn, v2.HashPolicy{
				Cookie: &v2.CookieHashPolicy{
					Name: cookieConfig.Name,
					Path: cookieConfig.Path,
					TTL: api.DurationConfig{
						Duration: ConvertDuration(cookieConfig.Ttl),
					},
				},
			})

			continue
		}

		if ip := p.GetConnectionProperties(); ip != nil {
			hpReturn = append(hpReturn, v2.HashPolicy{
				SourceIP: &v2.SourceIPHashPolicy{},
			})

			continue
		}
	}

	return hpReturn
}

func convertWeightedCluster(xdsWeightedCluster *envoy_config_route_v3.WeightedCluster_ClusterWeight) v2.ClusterWeight {
	if xdsWeightedCluster == nil {
		return v2.ClusterWeight{}
	}
	return v2.ClusterWeight{
		ClusterWeightConfig: v2.ClusterWeightConfig{
			Name:   xdsWeightedCluster.GetName(),
			Weight: xdsWeightedCluster.GetWeight().GetValue(),
		},
		MetadataMatch: convertMeta(xdsWeightedCluster.GetMetadataMatch()),
	}
}

func convertAddress(xdsAddress *envoy_config_core_v3.Address) net.Addr {
	if xdsAddress == nil {
		return nil
	}
	var address string
	if addr, ok := xdsAddress.GetAddress().(*envoy_config_core_v3.Address_SocketAddress); ok {
		if xdsPort, ok := addr.SocketAddress.GetPortSpecifier().(*envoy_config_core_v3.SocketAddress_PortValue); ok {
			address = fmt.Sprintf("%s:%d", addr.SocketAddress.GetAddress(), xdsPort.PortValue)
		} else {
			log.DefaultLogger.Warnf("only port value supported")
			return nil
		}
	} else {
		log.DefaultLogger.Errorf("only SocketAddress supported")
		return nil
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		log.DefaultLogger.Errorf("Invalid address: %v", err)
		return nil
	}
	return tcpAddr
}

func convertClusterType(xdsClusterType envoy_config_cluster_v3.Cluster_DiscoveryType) v2.ClusterType {
	switch xdsClusterType {
	case envoy_config_cluster_v3.Cluster_STATIC:
		return v2.SIMPLE_CLUSTER
	case envoy_config_cluster_v3.Cluster_STRICT_DNS:
		return v2.STRICT_DNS_CLUSTER
	case envoy_config_cluster_v3.Cluster_LOGICAL_DNS:
	case envoy_config_cluster_v3.Cluster_EDS:
		return v2.EDS_CLUSTER
	case envoy_config_cluster_v3.Cluster_ORIGINAL_DST:
		return v2.ORIGINALDST_CLUSTER
	}
	//log.DefaultLogger.Fatalf("unsupported cluster type: %s, exchange to SIMPLE_CLUSTER", xdsClusterType.String())
	return v2.SIMPLE_CLUSTER
}

// TODO: support CLUSTER_PROVIDED
func convertLbPolicy(clusterType envoy_config_cluster_v3.Cluster_DiscoveryType, xdsLbPolicy envoy_config_cluster_v3.Cluster_LbPolicy) v2.LbType {
	switch xdsLbPolicy {
	case envoy_config_cluster_v3.Cluster_ROUND_ROBIN:
		return v2.LB_ROUNDROBIN
	case envoy_config_cluster_v3.Cluster_LEAST_REQUEST:
		return v2.LB_LEAST_REQUEST
	case envoy_config_cluster_v3.Cluster_RANDOM:
		return v2.LB_RANDOM
	case envoy_config_cluster_v3.Cluster_CLUSTER_PROVIDED:
		// https://github.com/envoyproxy/envoy/issues/11664
		if clusterType == envoy_config_cluster_v3.Cluster_ORIGINAL_DST {
			return v2.LB_ORIGINAL_DST
		}
	case envoy_config_cluster_v3.Cluster_MAGLEV:
		return v2.LB_MAGLEV
	case envoy_config_cluster_v3.Cluster_RING_HASH:
		return v2.LB_MAGLEV
	}

	//log.DefaultLogger.Fatalf("unsupported lb policy: %s, exchange to LB_RANDOM", xdsLbPolicy.String())
	return v2.LB_RANDOM
}

func convertLbSubSetConfig(xdsLbSubsetConfig *envoy_config_cluster_v3.Cluster_LbSubsetConfig) v2.LBSubsetConfig {
	if xdsLbSubsetConfig == nil {
		return v2.LBSubsetConfig{}
	}
	return v2.LBSubsetConfig{
		FallBackPolicy:  uint8(xdsLbSubsetConfig.GetFallbackPolicy()),
		DefaultSubset:   convertTypesStruct(xdsLbSubsetConfig.GetDefaultSubset()),
		SubsetSelectors: convertSubsetSelectors(xdsLbSubsetConfig.GetSubsetSelectors()),
	}
}

func convertTypesStruct(s *structpb.Struct) map[string]string {
	if s == nil {
		return nil
	}
	meta := make(map[string]string, len(s.GetFields()))
	for key, value := range s.GetFields() {
		meta[key] = value.String()
	}
	return meta
}

func convertSubsetSelectors(xdsSubsetSelectors []*envoy_config_cluster_v3.Cluster_LbSubsetConfig_LbSubsetSelector) [][]string {
	if xdsSubsetSelectors == nil {
		return nil
	}
	subsetSelectors := make([][]string, 0, len(xdsSubsetSelectors))
	for _, xdsSubsetSelector := range xdsSubsetSelectors {
		subsetSelectors = append(subsetSelectors, xdsSubsetSelector.GetKeys())
	}
	return subsetSelectors
}

func convertHealthChecks(xdsHealthChecks []*envoy_config_core_v3.HealthCheck) v2.HealthCheck {
	if xdsHealthChecks == nil || len(xdsHealthChecks) == 0 || xdsHealthChecks[0] == nil {
		return v2.HealthCheck{}
	}

	return v2.HealthCheck{
		HealthCheckConfig: v2.HealthCheckConfig{
			HealthyThreshold:   xdsHealthChecks[0].GetHealthyThreshold().GetValue(),
			UnhealthyThreshold: xdsHealthChecks[0].GetUnhealthyThreshold().GetValue(),
		},
		Timeout:        ConvertDuration(xdsHealthChecks[0].GetTimeout()),
		Interval:       ConvertDuration(xdsHealthChecks[0].GetInterval()),
		IntervalJitter: ConvertDuration(xdsHealthChecks[0].GetIntervalJitter()),
	}
}

func convertCircuitBreakers(xdsCircuitBreaker *envoy_config_cluster_v3.CircuitBreakers) v2.CircuitBreakers {
	if xdsCircuitBreaker == nil || proto.Size(xdsCircuitBreaker) == 0 {
		return v2.CircuitBreakers{}
	}
	thresholds := make([]v2.Thresholds, 0, len(xdsCircuitBreaker.GetThresholds()))
	for _, xdsThreshold := range xdsCircuitBreaker.GetThresholds() {
		threshold := v2.Thresholds{
			MaxConnections:     xdsThreshold.GetMaxConnections().GetValue(),
			MaxPendingRequests: xdsThreshold.GetMaxPendingRequests().GetValue(),
			MaxRequests:        xdsThreshold.GetMaxRequests().GetValue(),
			MaxRetries:         xdsThreshold.GetMaxRetries().GetValue(),
		}
		thresholds = append(thresholds, threshold)
	}
	return v2.CircuitBreakers{
		Thresholds: thresholds,
	}
}

func convertSpec(xdsCluster *envoy_config_cluster_v3.Cluster) v2.ClusterSpecInfo {
	if xdsCluster == nil || xdsCluster.GetEdsClusterConfig() == nil {
		return v2.ClusterSpecInfo{}
	}
	specs := make([]v2.SubscribeSpec, 0, 1)
	spec := v2.SubscribeSpec{
		ServiceName: xdsCluster.GetEdsClusterConfig().GetServiceName(),
	}
	specs = append(specs, spec)
	return v2.ClusterSpecInfo{
		Subscribes: specs,
	}
}

func convertClusterHosts(xdsHosts []*envoy_config_core_v3.Address) []v2.Host {
	if xdsHosts == nil {
		return nil
	}
	hostsWithMetaData := make([]v2.Host, 0, len(xdsHosts))
	for _, xdsHost := range xdsHosts {
		hostWithMetaData := v2.Host{
			HostConfig: v2.HostConfig{
				Address: convertAddress(xdsHost).String(),
			},
		}
		hostsWithMetaData = append(hostsWithMetaData, hostWithMetaData)
	}
	return hostsWithMetaData
}
