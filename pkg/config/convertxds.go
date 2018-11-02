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

package config

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/filter"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/router"
	xdsxproxy "github.com/alipay/sofa-mosn/pkg/xds-config-model/filter/network/x_proxy/v2"
	"github.com/alipay/sofa-mosn/pkg/xds/v2/rds"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xdsauth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	xdscluster "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	xdscore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	xdsendpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	xdslistener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	xdsroute "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	xdsaccesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	xdshttp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	xdstcp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	xdsutil "github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/types"
)

var supportFilter = map[string]bool{
	xdsutil.HTTPConnectionManager: true,
	v2.RPC_PROXY:                  true,
	v2.X_PROXY:                    true,
}

var httpBaseConfig = map[string]bool{
	xdsutil.HTTPConnectionManager: true,
	v2.RPC_PROXY:                  true,
}

// todo add streamfilters parse
func convertListenerConfig(xdsListener *xdsapi.Listener) *v2.Listener {
	if !isSupport(xdsListener) {
		return nil
	}

	listenerConfig := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       xdsListener.GetName(),
			BindToPort: convertBindToPort(xdsListener.GetDeprecatedV1()),
			Inspector:  true,
			HandOffRestoredDestinationConnections: xdsListener.GetUseOriginalDst().GetValue(),
			AccessLogs:                            convertAccessLogs(xdsListener),
			LogPath:                               "stdout",
		},
		Addr: convertAddress(&xdsListener.Address),
		PerConnBufferLimitBytes: xdsListener.GetPerConnectionBufferLimitBytes().GetValue(),
		LogLevel:                uint8(log.INFO),
	}

	// virtual listener need none filters
	if listenerConfig.Name == "virtual" {
		return listenerConfig
	}

	listenerConfig.FilterChains, listenerConfig.StreamFilters = convertFilterChains(xdsListener.GetFilterChains())
	listenerConfig.DisableConnIo = GetListenerDisableIO(&listenerConfig.FilterChains[0])

	return listenerConfig
}

func convertClustersConfig(xdsClusters []*xdsapi.Cluster) []*v2.Cluster {
	if xdsClusters == nil {
		return nil
	}
	clusters := make([]*v2.Cluster, 0, len(xdsClusters))
	for _, xdsCluster := range xdsClusters {
		cluster := &v2.Cluster{
			Name:                 xdsCluster.GetName(),
			ClusterType:          convertClusterType(xdsCluster.GetType()),
			LbType:               convertLbPolicy(xdsCluster.GetLbPolicy()),
			LBSubSetConfig:       convertLbSubSetConfig(xdsCluster.GetLbSubsetConfig()),
			MaxRequestPerConn:    xdsCluster.GetMaxRequestsPerConnection().GetValue(),
			ConnBufferLimitBytes: xdsCluster.GetPerConnectionBufferLimitBytes().GetValue(),
			HealthCheck:          convertHealthChecks(xdsCluster.GetHealthChecks()),
			CirBreThresholds:     convertCircuitBreakers(xdsCluster.GetCircuitBreakers()),
			OutlierDetection:     convertOutlierDetection(xdsCluster.GetOutlierDetection()),
			Hosts:                convertClusterHosts(xdsCluster.GetHosts()),
			Spec:                 convertSpec(xdsCluster),
			TLS:                  convertTLS(xdsCluster.GetTlsContext()),
		}

		clusters = append(clusters, cluster)
	}

	return clusters
}

func convertEndpointsConfig(xdsEndpoint *xdsendpoint.LocalityLbEndpoints) []v2.Host {
	if xdsEndpoint == nil {
		return nil
	}
	hosts := make([]v2.Host, 0, len(xdsEndpoint.GetLbEndpoints()))
	for _, xdsHost := range xdsEndpoint.GetLbEndpoints() {
		var address string
		if xdsAddress, ok := xdsHost.GetEndpoint().GetAddress().GetAddress().(*xdscore.Address_SocketAddress); ok {
			if xdsPort, ok := xdsAddress.SocketAddress.GetPortSpecifier().(*xdscore.SocketAddress_PortValue); ok {
				address = fmt.Sprintf("%s:%d", xdsAddress.SocketAddress.GetAddress(), xdsPort.PortValue)
			} else if xdsPort, ok := xdsAddress.SocketAddress.GetPortSpecifier().(*xdscore.SocketAddress_NamedPort); ok {
				address = fmt.Sprintf("%s:%s", xdsAddress.SocketAddress.GetAddress(), xdsPort.NamedPort)
			} else {
				log.DefaultLogger.Warnf("unsupported port type")
				continue
			}

		} else if xdsAddress, ok := xdsHost.GetEndpoint().GetAddress().GetAddress().(*xdscore.Address_Pipe); ok {
			address = xdsAddress.Pipe.GetPath()
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

		if weight := xdsHost.GetLoadBalancingWeight().GetValue(); weight < MinHostWeight {
			host.Weight = MinHostWeight
		} else if weight > MaxHostWeight {
			host.Weight = MaxHostWeight
		}

		hosts = append(hosts, host)
	}
	return hosts
}

// todo: more filter type support
func isSupport(xdsListener *xdsapi.Listener) bool {
	if xdsListener == nil {
		return false
	}
	if xdsListener.Name == "virtual" {
		return true
	}
	for _, filterChain := range xdsListener.GetFilterChains() {
		for _, filter := range filterChain.GetFilters() {
			if value, ok := supportFilter[filter.GetName()]; !ok || !value {
				return false
			}
		}
	}
	return true
}

func convertBindToPort(xdsDeprecatedV1 *xdsapi.Listener_DeprecatedV1) bool {
	if xdsDeprecatedV1 == nil || xdsDeprecatedV1.GetBindToPort() == nil {
		return true
	}
	return xdsDeprecatedV1.GetBindToPort().GetValue()
}

// todo: more filter config support
func convertAccessLogs(xdsListener *xdsapi.Listener) []v2.AccessLog {
	if xdsListener == nil {
		return nil
	}

	accessLogs := make([]v2.AccessLog, 0)
	for _, xdsFilterChain := range xdsListener.GetFilterChains() {
		for _, xdsFilter := range xdsFilterChain.GetFilters() {
			if value, ok := httpBaseConfig[xdsFilter.GetName()]; ok && value {
				filterConfig := &xdshttp.HttpConnectionManager{}
				xdsutil.StructToMessage(xdsFilter.GetConfig(), filterConfig)
				for _, accConfig := range filterConfig.GetAccessLog() {
					if accConfig.Name == xdsutil.FileAccessLog {
						als := &xdsaccesslog.FileAccessLog{}
						xdsutil.StructToMessage(accConfig.GetConfig(), als)
						accessLog := v2.AccessLog{
							Path:   als.GetPath(),
							Format: als.GetFormat(),
						}
						accessLogs = append(accessLogs, accessLog)
					}
				}
			} else if xdsFilter.GetName() == xdsutil.TCPProxy {
				filterConfig := &xdstcp.TcpProxy{}
				xdsutil.StructToMessage(xdsFilter.GetConfig(), filterConfig)
				for _, accConfig := range filterConfig.GetAccessLog() {
					if accConfig.Name == xdsutil.FileAccessLog {
						als := &xdsaccesslog.FileAccessLog{}
						xdsutil.StructToMessage(accConfig.GetConfig(), als)
						accessLog := v2.AccessLog{
							Path:   als.GetPath(),
							Format: als.GetFormat(),
						}
						accessLogs = append(accessLogs, accessLog)
					}
				}

			} else if xdsFilter.GetName() == v2.X_PROXY {
				filterConfig := &xdsxproxy.XProxy{}
				xdsutil.StructToMessage(xdsFilter.GetConfig(), filterConfig)
				for _, accConfig := range filterConfig.GetAccessLog() {
					if accConfig.Name == xdsutil.FileAccessLog {
						als := &xdsaccesslog.FileAccessLog{}
						xdsutil.StructToMessage(accConfig.GetConfig(), als)
						accessLog := v2.AccessLog{
							Path:   als.GetPath(),
							Format: als.GetFormat(),
						}
						accessLogs = append(accessLogs, accessLog)
					}
				}
			} else {
				log.DefaultLogger.Errorf("unsupported filter config type, filter name: %s", xdsFilter.GetName())
			}
		}
	}
	return accessLogs
}

func convertFilterChains(xdsFilterChains []xdslistener.FilterChain) ([]v2.FilterChain, []v2.Filter) {
	if xdsFilterChains == nil {
		return nil, nil
	}
	filterChains := make([]v2.FilterChain, 0, len(xdsFilterChains))
	streamFilters := make([]v2.Filter, 0)

	for _, xdsFilterChain := range xdsFilterChains {
		filterChain := v2.FilterChain{
			FilterChainMatch: xdsFilterChain.GetFilterChainMatch().String(),
			TLS:              convertTLS(xdsFilterChain.GetTlsContext()),
		}
		f, streamFilter := convertFilters(xdsFilterChain.GetFilters())

		filterChain.Filters = f
		streamFilters = append([]v2.Filter(streamFilters), []v2.Filter(streamFilter)...)

		filterChains = append(filterChains, filterChain)
	}
	return filterChains, streamFilters
}

func convertFilters(xdsFilters []xdslistener.Filter) ([]v2.Filter, []v2.Filter) {
	if xdsFilters == nil {
		return nil, nil
	}

	filters := make([]v2.Filter, 0, len(xdsFilters))
	streamFilters := make([]v2.Filter, 0)

	for _, xdsFilter := range xdsFilters {
		filterMaps, streamFilter := convertFilterConfig(xdsFilter.GetName(), xdsFilter.GetConfig())
		streamFilters = append([]v2.Filter(streamFilters), []v2.Filter(streamFilter)...)

		for typeKey, configValue := range filterMaps {
			filters = append(filters, v2.Filter{
				typeKey,
				configValue,
			})
		}
	}

	return filters, streamFilters
}

func toMap(in interface{}) map[string]interface{} {
	var out map[string]interface{}
	data, _ := json.Marshal(in)
	json.Unmarshal(data, &out)
	return out
}

// TODO: more filter config support
func convertFilterConfig(name string, s *types.Struct) (map[string]map[string]interface{}, []v2.Filter) {
	if s == nil {
		return nil, nil
	}

	filtersConfigParsed := make(map[string]map[string]interface{})
	streamFilters := make([]v2.Filter, 0)

	var proxyConfig v2.Proxy
	var routerConfig *v2.RouterConfiguration
	var isRds bool

	if name == xdsutil.HTTPConnectionManager || name == v2.RPC_PROXY {
		filterConfig := &xdshttp.HttpConnectionManager{}
		xdsutil.StructToMessage(s, filterConfig)
		routerConfig, isRds = convertRouterConf(filterConfig.GetRds().GetRouteConfigName(), filterConfig.GetRouteConfig())

		if name == xdsutil.HTTPConnectionManager {
			proxyConfig = v2.Proxy{
				DownstreamProtocol: string(protocol.HTTP1),
				UpstreamProtocol:   string(protocol.HTTP1),
			}

			// add http filter parse in HTTPConnectionManager case
			for _, httpFilter := range filterConfig.HttpFilters {
				f, err := filter.CreateNamedHTTPFilterFactory(httpFilter.Name, httpFilter.Config)
				if err != nil {
					continue
				}
				filter := f.CreateFilter()
				streamFilters = append(streamFilters, filter)
			}
		} else {
			proxyConfig = v2.Proxy{
				DownstreamProtocol: string(protocol.SofaRPC),
				UpstreamProtocol:   string(protocol.SofaRPC),
			}
		}
	} else if name == v2.X_PROXY {
		filterConfig := &xdsxproxy.XProxy{}
		xdsutil.StructToMessage(s, filterConfig)
		routerConfig, isRds = convertRouterConf(filterConfig.GetRds().GetRouteConfigName(), filterConfig.GetRouteConfig())

		proxyConfig = v2.Proxy{
			DownstreamProtocol: string(protocol.Xprotocol),
			UpstreamProtocol:   string(protocol.Xprotocol),
			ExtendConfig:       convertXProxyExtendConfig(filterConfig),
		}

	} else {
		log.DefaultLogger.Errorf("unsupported filter config, filter name: %s", name)
		return nil, nil
	}

	var routerConfigName string

	// get connection manager filter for rds
	if routerConfig != nil {
		routerConfigName = routerConfig.RouterConfigName
		if isRds {
			rds.AppendRouterName(routerConfigName)
		}
		if routersMngIns := router.GetRoutersMangerInstance(); routersMngIns == nil {
			log.DefaultLogger.Errorf("xds AddOrUpdateRouters error: router manager in nil")
		} else {
			routersMngIns.AddOrUpdateRouters(routerConfig)
		}
		filtersConfigParsed[v2.CONNECTION_MANAGER] = toMap(routerConfig)
	} else {
		log.DefaultLogger.Errorf("no router config found, filter name: %s", name)
	}

	// get proxy
	proxyConfig.RouterConfigName = routerConfigName
	filtersConfigParsed[v2.DEFAULT_NETWORK_FILTER] = toMap(proxyConfig)
	return filtersConfigParsed, streamFilters
}

func convertXProxyExtendConfig(config *xdsxproxy.XProxy) map[string]interface{} {
	extendConfig := &v2.XProxyExtendConfig{
		SubProtocol: config.XProtocol,
	}
	return toMap(extendConfig)
}

func convertRouterConf(routeConfigName string, xdsRouteConfig *xdsapi.RouteConfiguration) (*v2.RouterConfiguration, bool) {
	if routeConfigName != "" {
		return &v2.RouterConfiguration{
			RouterConfigName: routeConfigName,
		}, true
	}

	if xdsRouteConfig == nil {
		return nil, false
	}

	virtualHosts := make([]*v2.VirtualHost, 0)

	for _, xdsVirtualHost := range xdsRouteConfig.GetVirtualHosts() {
		virtualHost := &v2.VirtualHost{
			Name:                    xdsVirtualHost.GetName(),
			Domains:                 xdsVirtualHost.GetDomains(),
			Routers:                 convertRoutes(xdsVirtualHost.GetRoutes()),
			RequireTLS:              xdsVirtualHost.GetRequireTls().String(),
			VirtualClusters:         convertVirtualClusters(xdsVirtualHost.GetVirtualClusters()),
			RequestHeadersToAdd:     convertHeadersToAdd(xdsVirtualHost.GetRequestHeadersToAdd()),
			ResponseHeadersToAdd:    convertHeadersToAdd(xdsVirtualHost.GetResponseHeadersToAdd()),
			ResponseHeadersToRemove: xdsVirtualHost.GetResponseHeadersToRemove(),
		}
		virtualHosts = append(virtualHosts, virtualHost)
	}

	return &v2.RouterConfiguration{
		RouterConfigName:        xdsRouteConfig.GetName(),
		VirtualHosts:            virtualHosts,
		RequestHeadersToAdd:     convertHeadersToAdd(xdsRouteConfig.GetRequestHeadersToAdd()),
		ResponseHeadersToAdd:    convertHeadersToAdd(xdsRouteConfig.GetResponseHeadersToAdd()),
		ResponseHeadersToRemove: xdsRouteConfig.GetResponseHeadersToRemove(),
	}, false
}

func convertRoutes(xdsRoutes []xdsroute.Route) []v2.Router {
	if xdsRoutes == nil {
		return nil
	}
	routes := make([]v2.Router, 0, len(xdsRoutes))
	for _, xdsRoute := range xdsRoutes {
		if xdsRouteAction := xdsRoute.GetRoute(); xdsRouteAction != nil {
			route := v2.Router{
				RouterConfig: v2.RouterConfig{
					Match:     convertRouteMatch(xdsRoute.GetMatch()),
					Route:     convertRouteAction(xdsRouteAction),
					Decorator: v2.Decorator(xdsRoute.GetDecorator().String()),
				},
				Metadata: convertMeta(xdsRoute.GetMetadata()),
			}
			route.PerFilterConfig = convertPerRouteConfig(xdsRoute.PerFilterConfig)
			routes = append(routes, route)
		} else if xdsRouteAction := xdsRoute.GetRedirect(); xdsRouteAction != nil {
			route := v2.Router{
				RouterConfig: v2.RouterConfig{
					Match:     convertRouteMatch(xdsRoute.GetMatch()),
					Redirect:  convertRedirectAction(xdsRouteAction),
					Decorator: v2.Decorator(xdsRoute.GetDecorator().String()),
				},
				Metadata: convertMeta(xdsRoute.GetMetadata()),
			}
			route.PerFilterConfig = convertPerRouteConfig(xdsRoute.PerFilterConfig)
			routes = append(routes, route)
		} else {
			log.DefaultLogger.Errorf("unsupported route actin, just Route and Redirect support yet, ignore this route")
			continue
		}
	}
	return routes
}

func convertPerRouteConfig(xdsPerRouteConfig map[string]*types.Struct) map[string]*v2.PerRouterConfig {
	perRouteConfig := make(map[string]*v2.PerRouterConfig, 0)

	for key, config := range xdsPerRouteConfig {

		perRouteConfig[key] = &v2.PerRouterConfig{
			Struct:config,
		}
	}

	return perRouteConfig
}

func convertRouteMatch(xdsRouteMatch xdsroute.RouteMatch) v2.RouterMatch {
	return v2.RouterMatch{
		Prefix:        xdsRouteMatch.GetPrefix(),
		Path:          xdsRouteMatch.GetPath(),
		Regex:         xdsRouteMatch.GetRegex(),
		CaseSensitive: xdsRouteMatch.GetCaseSensitive().GetValue(),
		Runtime:       convertRuntime(xdsRouteMatch.GetRuntime()),
		Headers:       convertHeaders(xdsRouteMatch.GetHeaders()),
	}
}

func convertRuntime(xdsRuntime *xdscore.RuntimeUInt32) v2.RuntimeUInt32 {
	if xdsRuntime == nil {
		return v2.RuntimeUInt32{}
	}
	return v2.RuntimeUInt32{
		DefaultValue: xdsRuntime.GetDefaultValue(),
		RuntimeKey:   xdsRuntime.GetRuntimeKey(),
	}
}

func convertHeaders(xdsHeaders []*xdsroute.HeaderMatcher) []v2.HeaderMatcher {
	if xdsHeaders == nil {
		return nil
	}
	headerMatchers := make([]v2.HeaderMatcher, 0, len(xdsHeaders))
	for _, xdsHeader := range xdsHeaders {
		headerMatcher := v2.HeaderMatcher{
			Name:  xdsHeader.GetName(),
			Value: xdsHeader.GetExactMatch(),
			Regex: xdsHeader.GetRegex().GetValue(),
		}

		// as pseudo headers not support when Http1.x upgrade to Http2, change pseudo headers to normal headers
		// this would be fix soon
		if strings.HasPrefix(headerMatcher.Name, ":") {
			headerMatcher.Name = headerMatcher.Name[1:]
		}
		headerMatchers = append(headerMatchers, headerMatcher)
	}
	return headerMatchers
}

func convertMeta(xdsMeta *xdscore.Metadata) v2.Metadata {
	if xdsMeta == nil {
		return nil
	}
	meta := make(map[string]string, len(xdsMeta.GetFilterMetadata()))
	for key, value := range xdsMeta.GetFilterMetadata() {
		meta[key] = value.String()
	}
	return meta
}

func convertRouteAction(xdsRouteAction *xdsroute.RouteAction) v2.RouteAction {
	if xdsRouteAction == nil {
		return v2.RouteAction{}
	}
	return v2.RouteAction{
		RouterActionConfig: v2.RouterActionConfig{
			ClusterName:             xdsRouteAction.GetCluster(),
			ClusterHeader:           xdsRouteAction.GetClusterHeader(),
			WeightedClusters:        convertWeightedClusters(xdsRouteAction.GetWeightedClusters()),
			RetryPolicy:             convertRetryPolicy(xdsRouteAction.GetRetryPolicy()),
			PrefixRewrite:           xdsRouteAction.GetPrefixRewrite(),
			HostRewrite:             xdsRouteAction.GetHostRewrite(),
			AutoHostRewrite:         xdsRouteAction.GetAutoHostRewrite().GetValue(),
			RequestHeadersToAdd:     convertHeadersToAdd(xdsRouteAction.GetRequestHeadersToAdd()),
			ResponseHeadersToAdd:    convertHeadersToAdd(xdsRouteAction.GetResponseHeadersToAdd()),
			ResponseHeadersToRemove: xdsRouteAction.GetResponseHeadersToRemove(),
		},
		MetadataMatch: convertMeta(xdsRouteAction.GetMetadataMatch()),
		Timeout:       convertTimeDurPoint2TimeDur(xdsRouteAction.GetTimeout()),
	}
}

func convertHeadersToAdd(headerValueOption []*xdscore.HeaderValueOption) []*v2.HeaderValueOption {
	if len(headerValueOption) < 1 {
		return nil
	}
	valueOptions := make([]*v2.HeaderValueOption, 0, len(headerValueOption))
	for _, opt := range headerValueOption {
		var isAppend *bool
		if opt.Append != nil {
			append := opt.GetAppend().GetValue()
			isAppend = &append
		}
		valueOptions = append(valueOptions, &v2.HeaderValueOption{
			Header: &v2.HeaderValue{
				Key:   opt.GetHeader().GetKey(),
				Value: opt.GetHeader().GetValue(),
			},
			Append: isAppend,
		})
	}
	return valueOptions
}

func convertTimeDurPoint2TimeDur(duration *time.Duration) time.Duration {
	if duration == nil {
		return time.Duration(0)
	}
	return *duration
}

func convertWeightedClusters(xdsWeightedClusters *xdsroute.WeightedCluster) []v2.WeightedCluster {
	if xdsWeightedClusters == nil {
		return nil
	}
	weightedClusters := make([]v2.WeightedCluster, 0, len(xdsWeightedClusters.GetClusters()))
	for _, cluster := range xdsWeightedClusters.GetClusters() {
		weightedCluster := v2.WeightedCluster{
			Cluster:          convertWeightedCluster(cluster),
			RuntimeKeyPrefix: xdsWeightedClusters.GetRuntimeKeyPrefix(),
		}
		weightedClusters = append(weightedClusters, weightedCluster)
	}
	return weightedClusters
}

func convertWeightedCluster(xdsWeightedCluster *xdsroute.WeightedCluster_ClusterWeight) v2.ClusterWeight {
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

func convertRetryPolicy(xdsRetryPolicy *xdsroute.RouteAction_RetryPolicy) *v2.RetryPolicy {
	if xdsRetryPolicy == nil {
		return &v2.RetryPolicy{}
	}
	return &v2.RetryPolicy{
		RetryPolicyConfig: v2.RetryPolicyConfig{
			RetryOn:    len(xdsRetryPolicy.GetRetryOn()) > 0,
			NumRetries: xdsRetryPolicy.GetNumRetries().GetValue(),
		},
		RetryTimeout: convertTimeDurPoint2TimeDur(xdsRetryPolicy.GetPerTryTimeout()),
	}
}

func convertRedirectAction(xdsRedirectAction *xdsroute.RedirectAction) v2.RedirectAction {
	if xdsRedirectAction == nil {
		return v2.RedirectAction{}
	}
	return v2.RedirectAction{
		HostRedirect: xdsRedirectAction.GetHostRedirect(),
		PathRedirect: xdsRedirectAction.GetPathRedirect(),
		ResponseCode: uint32(xdsRedirectAction.GetResponseCode()),
	}
}

func convertVirtualClusters(xdsVirtualClusters []*xdsroute.VirtualCluster) []v2.VirtualCluster {
	if xdsVirtualClusters == nil {
		return nil
	}
	virtualClusters := make([]v2.VirtualCluster, 0, len(xdsVirtualClusters))
	for _, xdsVirtualCluster := range xdsVirtualClusters {
		virtualCluster := v2.VirtualCluster{
			Pattern: xdsVirtualCluster.GetPattern(),
			Name:    xdsVirtualCluster.GetName(),
			Method:  xdsVirtualCluster.GetMethod().String(),
		}
		virtualClusters = append(virtualClusters, virtualCluster)
	}
	return virtualClusters
}

func convertAddress(xdsAddress *xdscore.Address) net.Addr {
	if xdsAddress == nil {
		return nil
	}
	var address string
	if addr, ok := xdsAddress.GetAddress().(*xdscore.Address_SocketAddress); ok {
		if xdsPort, ok := addr.SocketAddress.GetPortSpecifier().(*xdscore.SocketAddress_PortValue); ok {
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

func convertClusterType(xdsClusterType xdsapi.Cluster_DiscoveryType) v2.ClusterType {
	switch xdsClusterType {
	case xdsapi.Cluster_STATIC:
		return v2.SIMPLE_CLUSTER
	case xdsapi.Cluster_STRICT_DNS:
	case xdsapi.Cluster_LOGICAL_DNS:
	case xdsapi.Cluster_EDS:
		return v2.EDS_CLUSTER
	case xdsapi.Cluster_ORIGINAL_DST:
	}
	//log.DefaultLogger.Fatalf("unsupported cluster type: %s, exchange to SIMPLE_CLUSTER", xdsClusterType.String())
	return v2.SIMPLE_CLUSTER
}

func convertLbPolicy(xdsLbPolicy xdsapi.Cluster_LbPolicy) v2.LbType {
	switch xdsLbPolicy {
	case xdsapi.Cluster_ROUND_ROBIN:
		return v2.LB_ROUNDROBIN
	case xdsapi.Cluster_LEAST_REQUEST:
	case xdsapi.Cluster_RING_HASH:
	case xdsapi.Cluster_RANDOM:
		return v2.LB_RANDOM
	case xdsapi.Cluster_ORIGINAL_DST_LB:
	case xdsapi.Cluster_MAGLEV:
	}
	//log.DefaultLogger.Fatalf("unsupported lb policy: %s, exchange to LB_RANDOM", xdsLbPolicy.String())
	return v2.LB_RANDOM
}

func convertLbSubSetConfig(xdsLbSubsetConfig *xdsapi.Cluster_LbSubsetConfig) v2.LBSubsetConfig {
	if xdsLbSubsetConfig == nil {
		return v2.LBSubsetConfig{}
	}
	return v2.LBSubsetConfig{
		FallBackPolicy:  uint8(xdsLbSubsetConfig.GetFallbackPolicy()),
		DefaultSubset:   convertTypesStruct(xdsLbSubsetConfig.GetDefaultSubset()),
		SubsetSelectors: convertSubsetSelectors(xdsLbSubsetConfig.GetSubsetSelectors()),
	}
}

func convertTypesStruct(s *types.Struct) map[string]string {
	if s == nil {
		return nil
	}
	meta := make(map[string]string, len(s.GetFields()))
	for key, value := range s.GetFields() {
		meta[key] = value.String()
	}
	return meta
}

func convertSubsetSelectors(xdsSubsetSelectors []*xdsapi.Cluster_LbSubsetConfig_LbSubsetSelector) [][]string {
	if xdsSubsetSelectors == nil {
		return nil
	}
	subsetSelectors := make([][]string, 0, len(xdsSubsetSelectors))
	for _, xdsSubsetSelector := range xdsSubsetSelectors {
		subsetSelectors = append(subsetSelectors, xdsSubsetSelector.GetKeys())
	}
	return subsetSelectors
}

func convertHealthChecks(xdsHealthChecks []*xdscore.HealthCheck) v2.HealthCheck {
	if xdsHealthChecks == nil || len(xdsHealthChecks) == 0 || xdsHealthChecks[0] == nil {
		return v2.HealthCheck{}
	}

	return v2.HealthCheck{
		HealthCheckConfig: v2.HealthCheckConfig{
			HealthyThreshold:   xdsHealthChecks[0].GetHealthyThreshold().GetValue(),
			UnhealthyThreshold: xdsHealthChecks[0].GetUnhealthyThreshold().GetValue(),
		},
		Timeout:        *xdsHealthChecks[0].GetTimeout(),
		Interval:       *xdsHealthChecks[0].GetInterval(),
		IntervalJitter: convertDuration(xdsHealthChecks[0].GetIntervalJitter()),
	}
}

func convertCircuitBreakers(xdsCircuitBreaker *xdscluster.CircuitBreakers) v2.CircuitBreakers {
	if xdsCircuitBreaker == nil || xdsCircuitBreaker.Size() == 0 {
		return v2.CircuitBreakers{}
	}
	thresholds := make([]v2.Thresholds, 0, len(xdsCircuitBreaker.GetThresholds()))
	for _, xdsThreshold := range xdsCircuitBreaker.GetThresholds() {
		if xdsThreshold.Size() == 0 {
			continue
		}
		threshold := v2.Thresholds{
			Priority:           v2.RoutingPriority(xdsThreshold.GetPriority().String()),
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

func convertOutlierDetection(xdsOutlierDetection *xdscluster.OutlierDetection) v2.OutlierDetection {
	if xdsOutlierDetection == nil || xdsOutlierDetection.Size() == 0 {
		return v2.OutlierDetection{}
	}
	return v2.OutlierDetection{
		Consecutive5xx:                     xdsOutlierDetection.GetConsecutive_5Xx().GetValue(),
		Interval:                           convertDuration(xdsOutlierDetection.GetInterval()),
		BaseEjectionTime:                   convertDuration(xdsOutlierDetection.GetBaseEjectionTime()),
		MaxEjectionPercent:                 xdsOutlierDetection.GetMaxEjectionPercent().GetValue(),
		ConsecutiveGatewayFailure:          xdsOutlierDetection.GetEnforcingConsecutive_5Xx().GetValue(),
		EnforcingConsecutive5xx:            xdsOutlierDetection.GetConsecutive_5Xx().GetValue(),
		EnforcingConsecutiveGatewayFailure: xdsOutlierDetection.GetEnforcingConsecutiveGatewayFailure().GetValue(),
		EnforcingSuccessRate:               xdsOutlierDetection.GetEnforcingSuccessRate().GetValue(),
		SuccessRateMinimumHosts:            xdsOutlierDetection.GetSuccessRateMinimumHosts().GetValue(),
		SuccessRateRequestVolume:           xdsOutlierDetection.GetSuccessRateRequestVolume().GetValue(),
		SuccessRateStdevFactor:             xdsOutlierDetection.GetSuccessRateStdevFactor().GetValue(),
	}
}

func convertSpec(xdsCluster *xdsapi.Cluster) v2.ClusterSpecInfo {
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

func convertClusterHosts(xdsHosts []*xdscore.Address) []v2.Host {
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

func convertDuration(p *types.Duration) time.Duration {
	if p == nil {
		return time.Duration(0)
	}
	d := time.Duration(p.Seconds) * time.Second
	if p.Nanos != 0 {
		if dur := d + time.Duration(p.Nanos); (dur < 0) != (p.Nanos < 0) {
			log.DefaultLogger.Warnf("duration: %#v is out of range for time.Duration, ignore nanos", p)
		}
	}
	return d
}

func convertTLS(xdsTLSContext interface{}) v2.TLSConfig {
	var config v2.TLSConfig
	var isDownstream bool
	var common *xdsauth.CommonTlsContext

	if xdsTLSContext == nil {
		return config
	}
	if context, ok := xdsTLSContext.(*xdsauth.DownstreamTlsContext); ok {
		if context.GetRequireClientCertificate() != nil {
			config.VerifyClient = context.GetRequireClientCertificate().GetValue()
		}
		common = context.GetCommonTlsContext()
		isDownstream = true
	} else if context, ok := xdsTLSContext.(*xdsauth.UpstreamTlsContext); ok {
		config.ServerName = context.GetSni()
		common = context.GetCommonTlsContext()
		isDownstream = false
	}
	if common == nil {
		return config
	}
	// Currently only a single certificate is supported
	if common.GetTlsCertificates() != nil {
		for _, cert := range common.GetTlsCertificates() {
			if cert.GetCertificateChain() != nil && cert.GetPrivateKey() != nil {
				config.CertChain = cert.GetCertificateChain().String()
				config.PrivateKey = cert.GetPrivateKey().String()
			}
		}
	}

	if common.GetValidationContext() != nil && common.GetValidationContext().GetTrustedCa() != nil {
		config.CACert = common.GetValidationContext().GetTrustedCa().String()
	}
	if common.GetAlpnProtocols() != nil {
		config.ALPN = strings.Join(common.GetAlpnProtocols(), ",")
	}
	param := common.GetTlsParams()
	if param != nil {
		if param.GetCipherSuites() != nil {
			config.CipherSuites = strings.Join(param.GetCipherSuites(), ":")
		}
		if param.GetEcdhCurves() != nil {
			config.EcdhCurves = strings.Join(param.GetEcdhCurves(), ",")
		}
		config.MinVersion = xdsauth.TlsParameters_TlsProtocol_name[int32(param.GetTlsMinimumProtocolVersion())]
		config.MaxVersion = xdsauth.TlsParameters_TlsProtocol_name[int32(param.GetTlsMaximumProtocolVersion())]
	}

	if isDownstream && (config.CertChain == "" || config.PrivateKey == "") {
		log.DefaultLogger.Fatalf("tls_certificates are required in downstream tls_context")
		config.Status = false
		return config
	}

	config.Status = true
	return config
}
