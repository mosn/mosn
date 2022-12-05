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
	"bytes"
	"encoding/base64"
	"net"
	"strings"
	"time"

	udpa_type_v1 "github.com/cncf/udpa/go/udpa/type/v1"
	envoy_config_accesslog_v3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_filters_http_fault_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/fault/v3"
	envoy_extensions_filters_http_gzip_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/gzip/v3"
	envoy_extensions_filters_network_http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	envoy_extensions_filters_network_tcp_proxy_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoy_type_v3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/valyala/fasthttp"
	"mosn.io/api"
	iv2 "mosn.io/mosn/istio/istio1106/config/v2"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/featuregate"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls/extensions/sni"
	"mosn.io/mosn/pkg/protocol"
)

// DefaultPassthroughCluster: EGRESS_CLUSTER or INGRESS_CLUSTER, counterpart of v2.ListenerType
type DefaultPassthroughCluster string

const EGRESS_CLUSTER DefaultPassthroughCluster = "PassthroughCluster"
const INGRESS_CLUSTER DefaultPassthroughCluster = "InboundPassthroughClusterIpv4"

// support network filter list
var supportFilter = map[string]bool{
	wellknown.HTTPConnectionManager: true,
	wellknown.TCPProxy:              true,
	v2.RPC_PROXY:                    true,
	v2.X_PROXY:                      true,
}

// todo: more filter type support
func isSupport(xdsListener *envoy_config_listener_v3.Listener) bool {
	if xdsListener == nil {
		return false
	}
	if xdsListener.Name == "virtual" || xdsListener.Name == "virtualOutbound" || xdsListener.Name == "virtualInbound" {
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

// todo add support for rpc_proxy
var httpBaseConfig = map[string]bool{
	wellknown.HTTPConnectionManager: true,
}

// istio stream filter names, which is quite different from mosn
const (
	MosnPayloadLimit = "mosn.payload_limit"
)

func convertTrafficDirection(listener *envoy_config_listener_v3.Listener) v2.ListenerType {
	switch listener.TrafficDirection {
	case envoy_config_core_v3.TrafficDirection_INBOUND:
		return v2.INGRESS
	case envoy_config_core_v3.TrafficDirection_OUTBOUND:
		return v2.EGRESS
	default:
		return v2.EGRESS // TODO
	}
}

func ConvertListenerConfig(xdsListener *envoy_config_listener_v3.Listener, rh routeHandler) []*v2.Listener {
	if xdsListener == nil {
		return nil
	}
	addr := convertAddress(xdsListener.Address)
	if addr == nil {
		log.DefaultLogger.Errorf("[xds] convert listener without address %+v", xdsListener.Address)
		return nil
	}
	listenerName := xdsListener.GetName()
	if listenerName == "" {
		listenerName = addr.String()
	}
	listenerConfig := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       listenerName,
			BindToPort: convertBindToPort(xdsListener.GetDeprecatedV1()),
			Inspector:  false,
			AccessLogs: convertAccessLogsFromListener(xdsListener),
			Type:       convertTrafficDirection(xdsListener),
		},
		Addr:                    addr,
		PerConnBufferLimitBytes: xdsListener.GetPerConnectionBufferLimitBytes().GetValue(),
	}

	if xdsListener.GetUseOriginalDst().GetValue() {
		listenerConfig.OriginalDst = v2.REDIRECT
	} else if xdsListener.GetTransparent().GetValue() {
		listenerConfig.OriginalDst = v2.TPROXY
	}

	// convert listener filters.
	listenerFilters := make([]v2.Filter, 0, len(xdsListener.GetListenerFilters()))
	for _, lf := range xdsListener.GetListenerFilters() {
		lnf := convertListenerFilter(lf.GetName(), lf.GetTypedConfig())
		// convert success
		if lnf.Type != "" {
			// TODO: find a better way to handle it
			// virtualInbound should use local to match
			if listenerName == "virtualInbound" && lnf.Type == v2.ORIGINALDST_LISTENER_FILTER {
				lnf.Config = map[string]interface{}{
					"fallback_to_local": true,
				}
			}
			listenerFilters = append(listenerFilters, lnf)
		}
		// set use original dst flag if original dst filter is exists
		if lnf.Type == v2.ORIGINALDST_LISTENER_FILTER && listenerConfig.OriginalDst != v2.TPROXY {
			listenerConfig.OriginalDst = v2.REDIRECT
		}
	}
	listenerConfig.ListenerFilters = listenerFilters

	// use virtual listeners instead of multi filter chain.
	// TODO: support multi filter chain
	var virtualListeners []*v2.Listener
	listenerConfig.FilterChains, listenerConfig.StreamFilters, virtualListeners = convertFilterChains(xdsListener, listenerConfig.IsOriginalDst(), rh)
	mosnListeners := []*v2.Listener{listenerConfig}
	mosnListeners = append(mosnListeners, virtualListeners...)
	return mosnListeners
}

func convertListenerFilter(name string, s *any.Any) v2.Filter {
	filter := v2.Filter{}

	switch name {
	case wellknown.OriginalDestination:
		filter.Type = v2.ORIGINALDST_LISTENER_FILTER

	default:
		log.DefaultLogger.Errorf("not support %s listener filter.", name)
	}

	return filter
}

func convertBindToPort(xdsDeprecatedV1 *envoy_config_listener_v3.Listener_DeprecatedV1) bool {
	if xdsDeprecatedV1 == nil || xdsDeprecatedV1.GetBindToPort() == nil {
		return true
	}
	return xdsDeprecatedV1.BindToPort.GetValue()
}

func convertAccessLogsFromListener(xdsListener *envoy_config_listener_v3.Listener) []v2.AccessLog {
	if xdsListener == nil {
		return nil
	}

	accessLogs := make([]v2.AccessLog, 0)
	for _, accConfig := range xdsListener.GetAccessLog() {
		if accConfig.Name == wellknown.FileAccessLog {
			als, err := GetAccessLog(accConfig)
			if err != nil {
				log.DefaultLogger.Warnf("[convertxds] [accesslog] conversion is fail %s", err)
				continue
			}
			accessLog := v2.AccessLog{
				Path:   als.GetPath(),
				Format: als.GetFormat(),
			}
			accessLogs = append(accessLogs, accessLog)
		}
	}

	return accessLogs
}

func convertAccessLogsFromFilterChain(xdsFilterChain *envoy_config_listener_v3.FilterChain) []v2.AccessLog {
	if xdsFilterChain == nil {
		return nil
	}

	var envoyAccesslogs []*envoy_config_accesslog_v3.AccessLog
	accessLogs := make([]v2.AccessLog, 0)

	for _, xdsFilter := range xdsFilterChain.GetFilters() {
		if value, ok := httpBaseConfig[xdsFilter.GetName()]; ok && value {
			filterConfig := GetHTTPConnectionManager(xdsFilter)
			envoyAccesslogs = filterConfig.GetAccessLog()
		} else if xdsFilter.GetName() == wellknown.TCPProxy {
			filterConfig := GetTcpProxy(xdsFilter)
			envoyAccesslogs = filterConfig.GetAccessLog()
		}

		if envoyAccesslogs != nil {
			for _, accConfig := range envoyAccesslogs {
				if accConfig.Name == wellknown.FileAccessLog {
					als, err := GetAccessLog(accConfig)
					if err != nil {
						log.DefaultLogger.Warnf("[convertxds] [accesslog] conversion is fail %s", err)
						continue
					}
					accessLog := v2.AccessLog{
						Path:   als.GetPath(),
						Format: als.GetFormat(),
					}
					accessLogs = append(accessLogs, accessLog)
				}
			}
		} else {
			log.DefaultLogger.Infof("[xds] unsupported filter config type, filter name: %s", xdsFilter.GetName())
		}
	}
	return accessLogs

}

func convertStreamFilters(pack *filterPack) []v2.Filter {
	filters := make([]v2.Filter, 0)
	if filter := pack.connectionManager; filter != nil {
		filterConfig := GetHTTPConnectionManager(filter)
		for _, filter := range filterConfig.GetHttpFilters() {
			streamFilter := convertStreamFilter(filter.GetName(), filter.GetTypedConfig())
			if streamFilter.Type != "" {
				log.DefaultLogger.Debugf("add a new stream filter, %v", streamFilter.Type)
				filters = append(filters, streamFilter)
			}
		}
	}
	return filters
}

func convertStreamFilter(name string, s *any.Any) v2.Filter {
	filter := v2.Filter{}
	var err error
	switch name {
	case v2.FaultStream, wellknown.Fault:
		filter.Type = v2.FaultStream
		// istio maybe do not contain this config, but have configs in router
		// in this case, we create a fault inject filter that do nothing
		if s == nil {
			streamFault := &v2.StreamFaultInject{}
			filter.Config, err = makeJsonMap(streamFault)
			if err != nil {
				log.DefaultLogger.Errorf("convert fault inject config error: %v", err)
			}
		} else { // common case
			filter.Config, err = convertStreamFaultInjectConfig(s)
			if err != nil {
				log.DefaultLogger.Errorf("convert fault inject config error: %v", err)
			}
		}
	case v2.Gzip, wellknown.Gzip:
		filter.Type = v2.Gzip
		// istio maybe do not contain this config, but have configs in router
		// in this case, we create a gzip filter that do nothing
		if s == nil {
			streamGzip := &v2.StreamGzip{}
			filter.Config, err = makeJsonMap(streamGzip)
			if err != nil {
				log.DefaultLogger.Errorf("convert fault inject config error: %v", err)
			}
		} else { // common case
			filter.Config, err = convertStreamGzipConfig(s)
			if err != nil {
				log.DefaultLogger.Errorf("convert gzip config error: %v", err)
			}
		}
	case iv2.IstioStats:
		m, err := convertUdpaTypedStructConfig(s)
		if err != nil {
			log.DefaultLogger.Errorf("convert %s config error: %v", name, err)
		}
		filter.Type = name
		filter.Config = m
	case wellknown.HTTPRoleBasedAccessControl:
		filter.Type = iv2.RBAC
		filter.Config, err = convertStreamRbacConfig(s)
		if err != nil {
			// TODO: if rbac config is in PerRoute format, use empty config to make sure rbac filter will be initialized.
			log.DefaultLogger.Errorf("convertRbacConfig error: %v", err)
		}
	case MosnPayloadLimit:
		if featuregate.Enabled(featuregate.PayLoadLimitEnable) {
			filter.Type = v2.PayloadLimit
			if s == nil {
				payloadLimitInject := &v2.StreamPayloadLimit{}
				filter.Config, err = makeJsonMap(payloadLimitInject)
				if err != nil {
					log.DefaultLogger.Errorf("convert payload limit config error: %v", err)
				}
			} else {
				//filter.Config, err = convertStreamPayloadLimitConfig(s)
				if err != nil {
					log.DefaultLogger.Errorf("convert payload limit config error: %v", err)
				}
			}
		}
	default:
		config, err := api.HandleXDSConfig(name, s)
		if err != nil {
			log.DefaultLogger.Infof("[xds] convertStreamFilter, unsupported filter config, name: %s,err=%v", name, err)
			break
		}
		filter.Type = name
		filter.Config = config
	}
	return filter
}

func convertStreamFaultInjectConfig(s *any.Any) (map[string]interface{}, error) {
	faultConfig := &envoy_extensions_filters_http_fault_v3.HTTPFault{}
	if err := ptypes.UnmarshalAny(s, faultConfig); err != nil {
		return nil, err
	}

	var fixedDelaygo time.Duration
	if d := faultConfig.Delay.GetFixedDelay(); d != nil {
		fixedDelaygo = ConvertDuration(d)
	}

	// convert istio percentage to mosn percent
	delayPercent := convertIstioPercentage(faultConfig.Delay.GetPercentage())
	abortPercent := convertIstioPercentage(faultConfig.Abort.GetPercentage())

	streamFault := &v2.StreamFaultInject{
		Delay: &v2.DelayInject{
			DelayInjectConfig: v2.DelayInjectConfig{
				Percent: delayPercent,
				DelayDurationConfig: api.DurationConfig{
					Duration: fixedDelaygo,
				},
			},
			Delay: fixedDelaygo,
		},
		Abort: &v2.AbortInject{
			Percent: abortPercent,
			Status:  int(faultConfig.Abort.GetHttpStatus()),
		},
		UpstreamCluster: faultConfig.UpstreamCluster,
		Headers:         convertHeaders(faultConfig.GetHeaders()),
	}
	return makeJsonMap(streamFault)
}

func convertStreamGzipConfig(s *any.Any) (map[string]interface{}, error) {
	gzipConfig := &envoy_extensions_filters_http_gzip_v3.Gzip{}
	if err := ptypes.UnmarshalAny(s, gzipConfig); err != nil {
		return nil, err
	}

	// convert istio gzip for mosn
	var minContentLength, level uint32
	var contentType []string
	switch gzipConfig.GetCompressionLevel() {
	case envoy_extensions_filters_http_gzip_v3.Gzip_CompressionLevel_BEST:
		level = fasthttp.CompressBestCompression
	case envoy_extensions_filters_http_gzip_v3.Gzip_CompressionLevel_SPEED:
		level = fasthttp.CompressBestSpeed
	default:
		level = fasthttp.CompressDefaultCompression
	}

	if gzipConfig.GetCompressor() != nil {
		minContentLength = gzipConfig.GetCompressor().GetContentLength().GetValue()
		contentType = gzipConfig.GetCompressor().GetContentType()
	}
	streamGzip := &v2.StreamGzip{
		GzipLevel:     level,
		ContentLength: minContentLength,
		ContentType:   contentType,
	}
	return makeJsonMap(streamGzip)
}

func convertIstioPercentage(percent *envoy_type_v3.FractionalPercent) uint32 {
	if percent == nil {
		return 0
	}
	switch percent.Denominator {
	case envoy_type_v3.FractionalPercent_MILLION:
		return percent.Numerator / 10000
	case envoy_type_v3.FractionalPercent_TEN_THOUSAND:
		return percent.Numerator / 100
	case envoy_type_v3.FractionalPercent_HUNDRED:
		return percent.Numerator
	}
	return percent.Numerator
}

func makeJsonMap(v interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil

}

func convertUdpaTypedStructConfig(s *any.Any) (map[string]interface{}, error) {
	conf := udpa_type_v1.TypedStruct{}
	err := ptypes.UnmarshalAny(s, &conf)
	if err != nil {
		return nil, err
	}

	config := map[string]interface{}{}
	if conf.Value == nil || conf.Value.Fields == nil || conf.Value.Fields["configuration"] == nil {
		return config, nil
	}
	jsonpbMarshaler := jsonpb.Marshaler{}

	buf := bytes.NewBuffer(nil)
	err = jsonpbMarshaler.Marshal(buf, conf.Value.Fields["configuration"])
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf.Bytes(), &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func isTCPToBlackHole(proxy *envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy) bool {
	return proxy.GetCluster() == "BlackHoleCluster"
}

func isTCPPassthrough(proxy *envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy) bool {
	return proxy.GetCluster() == "InboundPassthroughClusterIpv4"
}

func unmarshalTypedConfigHTTP(config *any.Any) (manager *envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager, err error) {
	var receiver envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager
	if err = ptypes.UnmarshalAny(config, &receiver); err != nil {
		return
	}
	manager = &receiver
	return
}

func isHTTPPassthrough(manager *envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager) bool {
	return manager.GetRouteConfig() != nil && manager.GetRouteConfig().Name == "InboundPassthroughClusterIpv4"
}

type filterPack struct {
	connectionManager *envoy_config_listener_v3.Filter
	filter            *v2.Filter
}

func convertFilterChains(xdsListener *envoy_config_listener_v3.Listener, useOriginalDst bool, rh routeHandler) (filterChains []v2.FilterChain, streamFilters []v2.Filter, virtualListeners []*v2.Listener) {
	if xdsListener == nil {
		return
	}

	xdsFilterChains := xdsListener.GetFilterChains()
	if xdsFilterChains == nil {
		return
	}

	var listernerPort uint32
	if address := xdsListener.GetAddress(); address != nil {
		if sa := address.GetSocketAddress(); sa != nil {
			listernerPort = sa.GetPortValue()
		}
	}

	// TODO: Only one chain is supported now
	for _, xdsFilterChain := range xdsFilterChains {
		xdsFilters := xdsFilterChain.GetFilters()
		// distinguish between multiple filterChainMaths
		chainMatch, chainMatchPort := convertFilterChainMatch(xdsFilterChain.GetFilterChainMatch())
		// TODO remove port arg?
		var port uint32
		if chainMatchPort != 0 {
			port = chainMatchPort
		} else {
			port = listernerPort
		}
		var filters []v2.Filter
		var name string
		filters, streamFilters, name = convertFilters(xdsFilters, port, rh)
		if filters == nil {
			log.DefaultLogger.Debugf("[convertxds] get empty filters, skip. listenerName: %s, chainMatch: %s", xdsListener.GetName(), chainMatch)
			continue
		}
		tls := convertTLS(xdsFilterChain.TransportSocket)
		// Build virtual listener with destination port
		if useOriginalDst {
			if chainMatchPort == 0 {
				log.DefaultLogger.Debugf("[convertxds] DestinationPort is nil, skip building virtualListener. xdsListener: %s, chainmatch: %s", xdsListener.GetName(), chainMatch)
				continue
			}
			addr := &net.TCPAddr{
				IP:   net.ParseIP("127.0.0.1"), // TODO: v6?
				Port: int(chainMatchPort),
			}
			found := false

			for i, virtualListener := range virtualListeners {
				if virtualListener.Addr.String() == addr.String() {
					found = true
					tlsFound := virtualListeners[i].FilterChains[0].TLSContexts
					if !tlsFound[0].Status && tls.Status {
						tlsFound[0] = tls
						virtualListeners[i].Inspector = true
					} else if tlsFound[0].Status && !tls.Status {
						virtualListeners[i].Inspector = true
					}
					break
				}
			}
			if !found {
				virtualListeners = append(virtualListeners, &v2.Listener{
					Addr: addr,
					ListenerConfig: v2.ListenerConfig{
						Name:       name,
						BindToPort: false,
						Inspector:  false,
						Type:       convertTrafficDirection(xdsListener),
						FilterChains: []v2.FilterChain{
							{
								FilterChainConfig: v2.FilterChainConfig{
									FilterChainMatch: chainMatch,
									Filters:          filters,
								},
								TLSContexts: []v2.TLSConfig{tls},
							},
						},
						StreamFilters: streamFilters,
						AccessLogs:    convertAccessLogsFromFilterChain(xdsFilterChain),
					},
					PerConnBufferLimitBytes: xdsListener.GetPerConnectionBufferLimitBytes().GetValue(),
				})
			}
		} else {
			log.DefaultLogger.Debugf("[convertxds] useOriginalDst is false, skip building virtualListener. xdsListener: %s, chainmatch: %s", xdsListener.GetName(), chainMatch)
			filterChains = []v2.FilterChain{
				{
					FilterChainConfig: v2.FilterChainConfig{
						FilterChainMatch: chainMatch,
						Filters:          filters,
					},
					TLSContexts: []v2.TLSConfig{tls},
				},
			}
			log.DefaultLogger.Debugf("[convertxds] filterChains got, ignore other xdsFilterChains")
			break
		}
	}
	if useOriginalDst {
		// TODO support blackhole

		var cluster DefaultPassthroughCluster
		if convertTrafficDirection(xdsListener) == v2.INGRESS {
			cluster = INGRESS_CLUSTER
		} else {
			cluster = EGRESS_CLUSTER
		}
		filterChains = []v2.FilterChain{
			{
				FilterChainConfig: v2.FilterChainConfig{
					Filters: []v2.Filter{
						{
							Type: v2.TCP_PROXY,
							Config: map[string]interface{}{
								"cluster": cluster,
							},
						},
					},
				},
				TLSContexts: []v2.TLSConfig{{}},
			},
		}
		streamFilters = nil
	}

	if filterChains == nil {
		log.DefaultLogger.Errorf("[convertxds] unsupported listener with no filter, listener %s", xdsListener.Name)
	}
	return
}

func convertFilters(xdsFilters []*envoy_config_listener_v3.Filter, port uint32, rh routeHandler) (
	filters []v2.Filter, streamFilters []v2.Filter, name string) {
	proxy, tcpProxy, connectionManager, err := convertNetworkFilters(xdsFilters, port, rh)
	if err != nil {
		log.DefaultLogger.Errorf("[convertxds] convertNetworkFilters, failed, %s", err)
		return
	}
	var mainFilter *v2.Filter
	if tcpProxy != nil {
		mainFilter = &v2.Filter{
			Type:   v2.TCP_PROXY,
			Config: toMap(tcpProxy),
		}
		name = tcpProxy.Cluster
	}
	if proxy != nil {
		mainFilter = &v2.Filter{
			Type:   v2.DEFAULT_NETWORK_FILTER,
			Config: toMap(proxy),
		}
		name = proxy.RouterConfigName
	}
	if mainFilter == nil {
		log.DefaultLogger.Warnf("[convertxds] convertFilters get mainFilter empty: %+v", xdsFilters)
		return
	}
	filters = []v2.Filter{*mainFilter}
	streamFilters = convertStreamFilters(&filterPack{
		connectionManager: connectionManager,
		filter:            mainFilter,
	})
	return
}

func toMap(in interface{}) map[string]interface{} {
	var out map[string]interface{}
	data, _ := json.Marshal(in)
	json.Unmarshal(data, &out)
	return out
}

type filterConverter struct {
	filter   *envoy_config_listener_v3.Filter
	proxy    *v2.Proxy
	tcpProxy *v2.StreamProxy
}

func (fc *filterConverter) convertHTTP(
	filter *envoy_config_listener_v3.Filter, oldRC *v2.RouterConfiguration, oldRds bool) (
	routerConfig *v2.RouterConfiguration, isRds bool, err error) {
	if fc.proxy != nil {
		routerConfig = oldRC
		isRds = oldRds
		return
	}
	if config := filter.GetTypedConfig(); config != nil {
		var manager *envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager
		if manager, err = unmarshalTypedConfigHTTP(config); err != nil {
			log.DefaultLogger.Warnf(
				"[convertxds] convertFilterChainsAndGetRawFilter, unmarshal http connection manager typed config failed, %s", err)
			return
		} else if isHTTPPassthrough(manager) {
			return
		} else {
			config := GetHTTPConnectionManager(filter)
			routerConfig, isRds = ConvertRouterConf(config.GetRds().GetRouteConfigName(), config.GetRouteConfig())
			fc.proxy = &v2.Proxy{
				DownstreamProtocol: string(protocol.Auto),
				RouterConfigName:   routerConfig.RouterConfigName,
				UpstreamProtocol:   string(protocol.Auto),
			}
			fc.filter = filter
		}
	}
	return
}

func (fc *filterConverter) convertTCP(filter *envoy_config_listener_v3.Filter) error {
	if fc.proxy != nil {
		return nil
	}
	filterConfig := GetTcpProxy(filter)
	if isTCPPassthrough(filterConfig) {
		return nil
	}
	if isTCPToBlackHole(filterConfig) {
		return nil
	}
	log.DefaultLogger.Tracef("TCPProxy:filter config = %v", filterConfig)
	d, err := ptypes.Duration(filterConfig.GetIdleTimeout())
	if err != nil {
		log.DefaultLogger.Infof("[xds] [convert] Idletimeout is nil: %s", filter.Name)
	}
	fc.tcpProxy = &v2.StreamProxy{
		StatPrefix:         filterConfig.GetStatPrefix(),
		Cluster:            filterConfig.GetCluster(),
		IdleTimeout:        &d,
		MaxConnectAttempts: filterConfig.GetMaxConnectAttempts().GetValue(),
	}
	return nil
}

func convertNetworkFilters(filters []*envoy_config_listener_v3.Filter, port uint32, rh routeHandler) (proxy *v2.Proxy, tcpProxy *v2.StreamProxy,
	connectionManager *envoy_config_listener_v3.Filter, err error) {
	var routerConfig *v2.RouterConfiguration
	var isRds bool
	converter := new(filterConverter)
	for _, filter := range filters {
		switch filter.Name {
		case wellknown.HTTPConnectionManager:
			if routerConfig, isRds, err = converter.convertHTTP(filter, routerConfig, isRds); err != nil {
				return
			}
		case wellknown.TCPProxy:
			if err = converter.convertTCP(filter); err != nil {
				return
			}
		case wellknown.RoleBasedAccessControl:
			// TODO
			fallthrough
		default:
			log.DefaultLogger.Infof(
				"[xds] convertNetworkFilters, unsupported filter config, filter name: %s", filter.Name)
		}
	}
	connectionManager = converter.filter
	proxy = converter.proxy
	tcpProxy = converter.tcpProxy
	// get connection manager filter for rds
	if rh != nil {
		rh(isRds, routerConfig)
	}
	//  get proxy
	if routerConfig != nil && proxy != nil {
		routerConfigName := routerConfig.RouterConfigName
		proxy.RouterConfigName = routerConfigName
	}
	return
}

func convertCidrRange(cidr []*envoy_config_core_v3.CidrRange) []v2.CidrRange {
	if cidr == nil {
		return nil
	}
	cidrRanges := make([]v2.CidrRange, 0, len(cidr))
	for _, cidrRange := range cidr {
		cidrRanges = append(cidrRanges, v2.CidrRange{
			Address: cidrRange.GetAddressPrefix(),
			Length:  cidrRange.GetPrefixLen().GetValue(),
		})
	}
	return cidrRanges
}

func convertTLS(transport interface{}) v2.TLSConfig {
	var config v2.TLSConfig
	var isUpstream bool
	var isSdsMode bool
	var common *envoy_extensions_transport_sockets_tls_v3.CommonTlsContext

	ts, ok := transport.(*envoy_config_core_v3.TransportSocket)
	if !ok {
		return config
	}

	// check upstream tls context for client or downstream tls context for server
	upTLSContext := &envoy_extensions_transport_sockets_tls_v3.UpstreamTlsContext{}
	if err := ptypes.UnmarshalAny(ts.GetTypedConfig(), upTLSContext); err == nil {
		// is upstream tls context, parse some parameters for upstream tls context
		isUpstream = true
		config.ServerName = upTLSContext.GetSni()
		// set common for create tls config
		common = upTLSContext.GetCommonTlsContext()

	} else {
		// is not upstream tls context, try downstream tls context
		dsTLSContext := &envoy_extensions_transport_sockets_tls_v3.DownstreamTlsContext{}
		if err := ptypes.UnmarshalAny(ts.GetTypedConfig(), dsTLSContext); err != nil {
			return config
		}
		// parse some parameters for downstream tls context
		if requireClient := dsTLSContext.GetRequireClientCertificate(); requireClient != nil {
			config.RequireClientCert = requireClient.GetValue()
			config.VerifyClient = requireClient.GetValue()
		}
		// set common for create tls config
		common = dsTLSContext.GetCommonTlsContext()
	}

	// Only one of *tls_certificates*, *tls_certificate_sds_secret_configs*, and *tls_certificate_provider_instance* may be used.
	// MOSN support *tls_certificates* and *tls_certificate_sds_secret_configs*
	// MOSN support single certificate in xds. TODO: support multiple certificates
	if tlsCertConfig := common.GetTlsCertificates(); tlsCertConfig != nil {
		for _, cert := range tlsCertConfig {
			config.CertChain = parseDataSource(cert.GetCertificateChain())
			config.PrivateKey = parseDataSource(cert.GetPrivateKey())
		}
	} else if tlsCertSdsConfig := common.GetTlsCertificateSdsSecretConfigs(); len(tlsCertSdsConfig) > 0 {
		isSdsMode = true
		config.SdsConfig = &v2.SdsConfig{
			CertificateConfig: &v2.SecretConfigWrapper{
				Name:      tlsCertSdsConfig[0].GetName(),
				SdsConfig: tlsCertSdsConfig[0].GetSdsConfig(),
			},
		}
	} else {
		log.DefaultLogger.Errorf("unsupported tls certificate types")
	}

	// ValidationContextType, MOSN support *CommonTlsContext_ValidationContext* and *CommonTlsContext_CombinedValidationContext*
	// ValidationContextType can be empty, which means use system info to verify
	if validationConfig := common.GetValidationContext(); validationConfig != nil {
		if isUpstream { // usptream maybe use sni verify type
			if sans := validationConfig.GetMatchSubjectAltNames(); len(sans) == 1 {
				config.Type = sni.SniVerify
				config.ExtendVerify = map[string]interface{}{
					sni.ConfigKey: sans[0].GetExact(),
				}
			}
		}
		config.CACert = parseDataSource(validationConfig.GetTrustedCa())
	} else if combinedValidationConfig := common.GetCombinedValidationContext(); combinedValidationConfig != nil {
		// In MOSN, a validation context should be SDS when certificate is SDS too. but we still support
		// other configs from default validation context when we use SDS config.
		if isUpstream { // usptream maybe use sni verify type
			if defaultValidationConfig := combinedValidationConfig.GetDefaultValidationContext(); defaultValidationConfig != nil {
				if sans := defaultValidationConfig.GetMatchSubjectAltNames(); len(sans) == 1 {
					config.Type = sni.SniVerify
					config.ExtendVerify = map[string]interface{}{
						sni.ConfigKey: sans[0].GetExact(),
					}
				}
			}
		}
		// sds mode, validation is sds too
		if isSdsMode {
			if sdsValidationConfig := combinedValidationConfig.GetValidationContextSdsSecretConfig(); sdsValidationConfig != nil {
				config.SdsConfig.ValidationConfig = &v2.SecretConfigWrapper{
					Name:      sdsValidationConfig.GetName(),
					SdsConfig: sdsValidationConfig.GetSdsConfig(),
				}
			}
		} else {
			if defaultValidationConfig := combinedValidationConfig.GetDefaultValidationContext(); defaultValidationConfig != nil {
				config.CACert = parseDataSource(defaultValidationConfig.GetTrustedCa())
			}
		}
	}

	// TlsParameters contains TLS protocol versions, cipher suites etc.
	if common.GetAlpnProtocols() != nil {
		config.ALPN = strings.Join(common.GetAlpnProtocols(), ",")
	}
	if param := common.GetTlsParams(); param != nil {
		if param.GetCipherSuites() != nil {
			config.CipherSuites = strings.Join(param.GetCipherSuites(), ":")
		}
		if param.GetEcdhCurves() != nil {
			config.EcdhCurves = strings.Join(param.GetEcdhCurves(), ",")
		}
		config.MinVersion = envoy_extensions_transport_sockets_tls_v3.TlsParameters_TlsProtocol_name[int32(param.GetTlsMinimumProtocolVersion())]
		config.MaxVersion = envoy_extensions_transport_sockets_tls_v3.TlsParameters_TlsProtocol_name[int32(param.GetTlsMaximumProtocolVersion())]

	}

	if !isSdsMode && !isUpstream && (config.CertChain == "" || config.PrivateKey == "") {
		log.DefaultLogger.Errorf("tls_certificates are required in downstream tls_context")
		config.Status = false
		return config
	}
	config.Status = true
	return config
}

// Types that are assignable to Specifier:
//      *DataSource_Filename
//      *DataSource_InlineBytes
//      *DataSource_InlineString
//      *DataSource_EnvironmentVariable
func parseDataSource(ds *envoy_config_core_v3.DataSource) string {
	spec := ds.GetSpecifier()
	switch spec.(type) {
	case *envoy_config_core_v3.DataSource_Filename:
		return ds.GetFilename()
	case *envoy_config_core_v3.DataSource_InlineBytes:
		// try to decode base64 encoded data
		data := ds.GetInlineBytes()
		enc := base64.StdEncoding
		dbuf := make([]byte, enc.DecodedLen(len(data)))
		n, err := enc.Decode(dbuf, data)
		if err != nil {
			return string(data)
		}
		return string(dbuf[:n])
	case *envoy_config_core_v3.DataSource_InlineString:
		return ds.GetInlineString()
	case *envoy_config_core_v3.DataSource_EnvironmentVariable:
		return ds.GetEnvironmentVariable()
	default: // unsupported typed
		return ds.String()
	}
}

func convertMirrorPolicy(xdsRouteAction *envoy_config_route_v3.RouteAction) *v2.RequestMirrorPolicy {
	if len(xdsRouteAction.GetRequestMirrorPolicies()) > 0 {
		return &v2.RequestMirrorPolicy{
			Cluster: xdsRouteAction.GetRequestMirrorPolicies()[0].GetCluster(),
			Percent: convertRuntimePercentage(xdsRouteAction.GetRequestMirrorPolicies()[0].GetRuntimeFraction()),
		}
	}

	return nil
}

func convertRuntimePercentage(percent *envoy_config_core_v3.RuntimeFractionalPercent) uint32 {
	if percent == nil {
		return 0
	}

	v := percent.GetDefaultValue()
	switch v.GetDenominator() {
	case envoy_type_v3.FractionalPercent_MILLION:
		return v.Numerator / 10000
	case envoy_type_v3.FractionalPercent_TEN_THOUSAND:
		return v.Numerator / 100
	case envoy_type_v3.FractionalPercent_HUNDRED:
		return v.Numerator
	}
	return v.Numerator
}

func convertUseOriginalDst(xdsListener *envoy_config_listener_v3.Listener) bool {
	if xdsListener.GetName() == "virtualOutbound" {
		return true
	}
	for _, xl := range xdsListener.GetListenerFilters() {
		if xl.Name == wellknown.OriginalDestination {
			return true
		}
	}
	return false
}

func convertFilterChainMatch(filterChainMatch *envoy_config_listener_v3.FilterChainMatch) (chainMatch string, destinationPort uint32) {
	chainMatch = filterChainMatch.String()
	if filterChainMatch.GetDestinationPort() != nil {
		destinationPort = filterChainMatch.GetDestinationPort().Value
	}
	return
}

func convertStreamRbacConfig(s *any.Any) (map[string]interface{}, error) {
	rbacConfig := iv2.RBACConfig{}
	err := ptypes.UnmarshalAny(s, &rbacConfig.RBAC)
	if err != nil {
		return nil, err
	}
	m := jsonpb.Marshaler{
		OrigName: true,
	}
	str, err := m.MarshalToString(&rbacConfig.RBAC)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}
	err = json.Unmarshal([]byte(str), &config)

	if err != nil {
		return nil, err
	}
	config["version"] = "envoy_config_rbac_v3"
	return config, nil
}
