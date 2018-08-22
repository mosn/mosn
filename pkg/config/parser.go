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
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/server"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/json-iterator/go"
)

type ContentKey string

const (
	MinHostWeight = uint32(1)
	MaxHostWeight = uint32(128)
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var protocolsSupported = map[string]bool{
	string(protocol.SofaRPC):   true,
	string(protocol.HTTP2):     true,
	string(protocol.HTTP1):     true,
	string(protocol.Xprotocol): true,
}

// RegisterProtocolParser
// used to register parser
func RegisterProtocolParser(key string) bool {
	if _, ok := protocolsSupported[key]; ok {
		return false
	} else {
		log.StartLogger.Infof(" %s added to protocolsSupported", key)
		protocolsSupported[key] = true
		return true
	}
}

// ParsedCallback is an
// alias for closure func(data interface{}, endParsing bool) error
type ParsedCallback func(data interface{}, endParsing bool) error

// Group of ContentKey
// notes: configcontentkey equals to the key of config file
const (
	ParseCallbackKeyCluster        ContentKey = "clusters"
	ParseCallbackKeyServiceRgtInfo ContentKey = "service_registry"
)

// RegisterConfigParsedListener
// used to register ParsedCallback
func RegisterConfigParsedListener(key ContentKey, cb ParsedCallback) {
	if cbs, ok := configParsedCBMaps[key]; ok {
		cbs = append(cbs, cb)
	} else {
		log.StartLogger.Infof(" %s added to configParsedCBMaps", key)
		cpc := []ParsedCallback{cb}
		configParsedCBMaps[key] = cpc
	}
}

var (
	configParsedCBMaps = make(map[ContentKey][]ParsedCallback)

	logLevelMap = map[string]log.Level{
		"TRACE": log.TRACE,
		"DEBUG": log.DEBUG,
		"FATAL": log.FATAL,
		"ERROR": log.ERROR,
		"WARN":  log.WARN,
		"INFO":  log.INFO,
	}

	clusterTypeMap = map[string]v2.ClusterType{
		"SIMPLE":  v2.SIMPLE_CLUSTER,
		"DYNAMIC": v2.DYNAMIC_CLUSTER,
	}

	lbTypeMap = map[string]v2.LbType{
		"LB_RANDOM":     v2.LB_RANDOM,
		"LB_ROUNDROBIN": v2.LB_ROUNDROBIN,
	}
)

func parseLogLevel(level string) log.Level {
	if level != "" {
		if logLevel, ok := logLevelMap[level]; ok {
			return logLevel
		}

		log.StartLogger.Fatalln("unsupported log level: ", level)
	}
	//use INFO as default log level
	return log.INFO
}

// ParseServerConfig
func ParseServerConfig(c *ServerConfig) *server.Config {
	sc := &server.Config{
		LogPath:         c.DefaultLogPath,
		LogLevel:        parseLogLevel(c.DefaultLogLevel),
		GracefulTimeout: c.GracefulTimeout.Duration,
		Processor:       c.Processor,
	}

	return sc
}

// ParseProxyFilterJSON
func ParseProxyFilterJSON(config map[string]interface{}) *v2.Proxy {
	proxyConfig := &Proxy{}

	if data, err := json.Marshal(config); err == nil {
		json.Unmarshal(data, &proxyConfig)
	} else {
		log.StartLogger.Fatal("Parsing Proxy Network Filter Error")
	}

	if proxyConfig.DownstreamProtocol == "" || proxyConfig.UpstreamProtocol == "" {
		log.StartLogger.Fatal("Protocol in String Needed in Proxy Network Filter")
	} else if _, ok := protocolsSupported[proxyConfig.DownstreamProtocol]; !ok {
		log.StartLogger.Fatal("Invalid Downstream Protocol = ", proxyConfig.DownstreamProtocol)
	} else if _, ok := protocolsSupported[proxyConfig.UpstreamProtocol]; !ok {
		log.StartLogger.Fatal("Invalid Upstream Protocol = ", proxyConfig.UpstreamProtocol)
	}

	if !proxyConfig.SupportDynamicRoute {
		log.StartLogger.Warnf("Mesh Doesn't Support Dynamic Router")
	}

	for _, vh := range proxyConfig.VirtualHosts {
		if len(vh.Routers) == 0 {
			log.StartLogger.Fatal("No Router Founded in VirtualHosts")
		}
	}

	return &v2.Proxy{
		Name:                proxyConfig.Name,
		DownstreamProtocol:  proxyConfig.DownstreamProtocol,
		UpstreamProtocol:    proxyConfig.UpstreamProtocol,
		SupportDynamicRoute: proxyConfig.SupportDynamicRoute,
		BasicRoutes:         nil,
		VirtualHosts:        parseVirtualHost(proxyConfig.VirtualHosts),
		ValidateClusters:    proxyConfig.ValidateClusters,
	}
}

func parseVirtualHost(confighost []*VirtualHost) []*v2.VirtualHost {
	result := []*v2.VirtualHost{}

	for _, cfh := range confighost {
		result = append(result, &v2.VirtualHost{
			Name:            cfh.Name,
			Domains:         cfh.Domains,
			Routers:         parseRouters(cfh.Routers),
			RequireTLS:      cfh.RequireTLS,
			VirtualClusters: parseVirtualClusters(cfh.VirtualClusters),
		})

	}

	return result
}

// used to check weight's validity
func checkWeightedClusterValid(action RouteAction) bool {
	var totalWeighted uint32 = 0

	for _, weightedCluster := range action.WeightedClusters {
		totalWeighted = totalWeighted + weightedCluster.Cluster.Weight
	}

	return totalWeighted == action.TotalClusterWeight
}

func parseRetryPolicy(action RouteAction) *v2.RetryPolicy {
	if action.RetryPolicy == nil {
		return nil
	} else {
		return &v2.RetryPolicy{
			action.RetryPolicy.RetryOn,
			action.RetryPolicy.RetryTimeout,
			action.RetryPolicy.NumRetries,
		}
	}
}

func parseRouters(Router []Router) []v2.Router {
	result := []v2.Router{}

	for _, router := range Router {
		if len(router.Route.WeightedClusters) > 0 && !checkWeightedClusterValid(router.Route) {
			log.StartLogger.Fatalln("Sum of weights in the weighted_cluster should add up to:", router.Route.TotalClusterWeight)
		}

		routerMatch := v2.RouterMatch{
			Prefix:        router.Match.Prefix,
			Path:          router.Match.Path,
			Regex:         router.Match.Regex,
			CaseSensitive: router.Match.CaseSensitive,
			Runtime: v2.RuntimeUInt32{
				router.Match.Runtime.DefaultValue,
				router.Match.Runtime.RuntimeKey,
			},
			Headers: parseMatchHeaders(router.Match.Headers),
		}

		routeAction := v2.RouteAction{
			ClusterName:        router.Route.ClusterName,
			ClusterHeader:      router.Route.ClusterHeader,
			TotalClusterWeight: router.Route.TotalClusterWeight,
			WeightedClusters:   parseWeightClusters(router.Route.WeightedClusters),
			MetadataMatch:      parseRouterMetadata(router.Route.MetadataMatch),
			Timeout:            router.Route.Timeout,
			RetryPolicy:        parseRetryPolicy(router.Route),
		}

		result = append(result, v2.Router{
			Match: routerMatch,
			Route: routeAction,
			Redirect: v2.RedirectAction{
				HostRedirect: router.Redirect.HostRedirect,
				PathRedirect: router.Redirect.PathRedirect,
				ResponseCode: router.Redirect.ResponseCode,
			},
			Metadata:  parseRouterMetadata(router.Metadata),
			Decorator: v2.Decorator(router.Decorator),
		})
	}

	return result
}

func parseWeightClusters(weightClusters []WeightedCluster) []v2.WeightedCluster {
	result := []v2.WeightedCluster{}

	for _, wc := range weightClusters {
		result = append(result, v2.WeightedCluster{
			Cluster: v2.ClusterWeight{
				wc.Cluster.Name,
				wc.Cluster.Weight,
				parseRouterMetadata(wc.Cluster.MetadataMatch),
			},
			RuntimeKeyPrefix: wc.RuntimeKeyPrefix,
		})
	}

	return result
}

// metadata format:
// { "filter_metadata": {"mosn.lb": { "label": "gray"  } } }
func parseRouterMetadata(metadata Metadata) v2.Metadata {
	if len(metadata) == 0 {
		return nil
	}

	result := v2.Metadata{}
	if metadataInterface, ok := metadata[types.RouterMetadataKey]; ok {
		if value, ok := metadataInterface.(map[string]interface{}); ok {
			if mosnLbInterface, ok := value[types.RouterMetadataKeyLb]; ok {
				if mosnLb, ok := mosnLbInterface.(map[string]interface{}); ok {
					for k, v := range mosnLb {
						if vs, ok := v.(string); ok {
							result[k] = vs
						}
					}

					return result
				}
			}
		}
	}
	log.StartLogger.Fatal("Metadata for routing format invalid, metadata format such as: { \"filter_metadata\": {\"mosn.lb\": { \"label\": \"gray\" } } }")

	return result
}

func parseMatchHeaders(headerMatchers []HeaderMatcher) []v2.HeaderMatcher {
	result := []v2.HeaderMatcher{}

	for _, hm := range headerMatchers {
		result = append(result, v2.HeaderMatcher{
			Name:  hm.Name,
			Value: hm.Value,
			Regex: hm.Regex,
		})
	}

	return result
}

func parseVirtualClusters(VirtualClusters []VirtualCluster) []v2.VirtualCluster {
	result := []v2.VirtualCluster{}

	for _, vc := range VirtualClusters {
		result = append(result, v2.VirtualCluster{
			Pattern: vc.Pattern,
			Name:    vc.Name,
			Method:  vc.Method,
		})

	}

	return result
}

func getServiceFromHeader(router *v2.Router) *v2.BasicServiceRoute {
	if router == nil {
		return nil
	}

	var ServiceName, ClusterName string
	for _, h := range router.Match.Headers {
		if h.Name == "service" || h.Name == "Service" {
			ServiceName = h.Value
		}
	}

	ClusterName = router.Route.ClusterName
	if ServiceName == "" || ClusterName == "" {
		return nil
	}

	return &v2.BasicServiceRoute{
		Service: ServiceName,
		Cluster: ClusterName,
	}
}

func parseBasicFilter(proxy *v2.Proxy) []*v2.BasicServiceRoute {
	var BSR []*v2.BasicServiceRoute

	for _, p := range proxy.VirtualHosts {

		for _, r := range p.Routers {
			BSR = append(BSR, getServiceFromHeader(&r))
		}
	}

	return BSR
}

// no used currently
func parseProxyFilter(c *v2.Filter) *v2.Proxy {
	proxyConfig := &v2.Proxy{}

	//downstream protocol
	//TODO config(json object) extract and type convert util
	if downstreamProtocol, ok := c.Config["downstream_protocol"]; ok {
		if downstreamProtocol, ok := downstreamProtocol.(string); ok {
			proxyConfig.DownstreamProtocol = downstreamProtocol
		} else {
			log.StartLogger.Fatalln("[downstream_protocol] in proxy filter config is not string")
		}
	} else {
		log.StartLogger.Fatalln("[downstream_protocol] is required in proxy filter config")
	}

	//upstream protocol
	if upstreamProtocol, ok := c.Config["upstream_protocol"]; ok {
		if upstreamProtocol, ok := upstreamProtocol.(string); ok {
			proxyConfig.UpstreamProtocol = upstreamProtocol
		} else {
			log.StartLogger.Fatalln("[upstream_protocol] in proxy filter config is not string")
		}
	} else {
		log.StartLogger.Fatalln("[upstream_protocol] is required in proxy filter config")
	}

	//todo support dynamic route or not, save
	if dynamicBool, ok := c.Config["support_dynamic_route"]; ok {
		if dynamicBool, ok := dynamicBool.(bool); ok {
			proxyConfig.SupportDynamicRoute = dynamicBool
		} else {
			log.StartLogger.Fatalln("support_dynamic_route in proxy filter support_dynamic_route is not bool")
		}
	} else {
		log.StartLogger.Debugf("support_dynamic_route doesn't set in proxy filter config")
	}

	//routes
	if routes, ok := c.Config["routes"]; ok {
		if routes, ok := routes.([]interface{}); ok {
			for _, route := range routes {
				proxyConfig.BasicRoutes = append(proxyConfig.BasicRoutes, parseRouteConfig(route.(map[string]interface{})))
			}
		} else {
			log.StartLogger.Fatalln("[routes] in proxy filter config is not list of routemap")
		}
	} else {
		log.StartLogger.Fatalln("[routes] is required in proxy filter config")
	}

	return proxyConfig
}

func parseAccessConfig(c []AccessLogConfig) []v2.AccessLog {
	var logs []v2.AccessLog

	for _, logConfig := range c {
		logs = append(logs, v2.AccessLog{
			Path:   logConfig.LogPath,
			Format: logConfig.LogFormat,
		})
	}

	return logs
}

func parseFilterChains(c []FilterChain) []v2.FilterChain {
	var filterchains []v2.FilterChain

	for _, fc := range c {
		filters := []v2.Filter{}
		for _, f := range fc.Filters {
			filters = append(filters, v2.Filter{
				Name:   f.Type,
				Config: f.Config,
			})
		}

		filterchains = append(filterchains, v2.FilterChain{
			FilterChainMatch: fc.FilterChainMatch,
			TLS:              parseTLSConfig(&fc.TLS),
			Filters:          filters,
		})
	}

	return filterchains
}

func parseTLSConfig(tlsconfig *TLSConfig) v2.TLSConfig {
	if tlsconfig.Status == false {
		return v2.TLSConfig{
			Status: false,
		}
	}

	return v2.TLSConfig{
		Status:       tlsconfig.Status,
		Type:         tlsconfig.Type,
		ServerName:   tlsconfig.ServerName,
		CACert:       tlsconfig.CACert,
		CertChain:    tlsconfig.CertChain,
		PrivateKey:   tlsconfig.PrivateKey,
		VerifyClient: tlsconfig.VerifyClient,
		InsecureSkip: tlsconfig.InsecureSkip,
		CipherSuites: tlsconfig.CipherSuites,
		EcdhCurves:   tlsconfig.EcdhCurves,
		MinVersion:   tlsconfig.MinVersion,
		MaxVersion:   tlsconfig.MaxVersion,
		ALPN:         tlsconfig.ALPN,
		Ticket:       tlsconfig.Ticket,
		ExtendVerify: tlsconfig.ExtendVerify,
	}
}

func parseRouteConfig(config map[string]interface{}) *v2.BasicServiceRoute {
	route := &v2.BasicServiceRoute{}

	//name
	if name, ok := config["name"]; ok {
		if name, ok := name.(string); ok {
			route.Name = name
		} else {
			log.StartLogger.Fatalln("[name] in proxy filter route config is not string")
		}
	} else {
		log.StartLogger.Fatalln("[name] is required in proxy filter route config")
	}

	//service
	if service, ok := config["service"]; ok {
		if service, ok := service.(string); ok {
			route.Service = service
		} else {
			log.StartLogger.Fatalln("[service] in proxy filter route config is not string")
		}
	} else {
		log.StartLogger.Fatalln("[service] is required in proxy filter route config")
	}

	//cluster
	if cluster, ok := config["cluster"]; ok {
		if cluster, ok := cluster.(string); ok {
			route.Cluster = cluster
		} else {
			log.StartLogger.Fatalln("[cluster] in proxy filter route config is not string")
		}
	} else {
		log.StartLogger.Fatalln("[cluster] is required in proxy filter route config")
	}

	return route
}

// ParseFaultInjectFilter
func ParseFaultInjectFilter(config map[string]interface{}) *v2.FaultInject {

	faultInject := &v2.FaultInject{}

	//percent
	if percent, ok := config["delay_percent"]; ok {
		if percent, ok := percent.(float64); ok {
			faultInject.DelayPercent = uint32(percent)
		} else {
			log.StartLogger.Fatalln("[delay_percent] in fault inject filter config is not integer")
		}
	} else {
		log.StartLogger.Fatalln("[delay_percent] is required in fault inject filter config")
	}

	//duration
	if duration, ok := config["delay_duration"]; ok {
		if duration, ok := duration.(string); ok {
			if duration, error := time.ParseDuration(strings.Trim(duration, `"`)); error == nil {
				faultInject.DelayDuration = uint64(duration)
			} else {
				log.StartLogger.Fatalln("[delay_duration] in fault inject filter config is not valid ,", error)
			}
		} else {
			log.StartLogger.Fatalln("[delay_duration] in fault inject filter config is not a numeric string, like '30s'")
		}
	} else {
		log.StartLogger.Fatalln("[delay_duration] is required in fault inject filter config")
	}

	return faultInject
}

// ParseHealthcheckFilter
func ParseHealthcheckFilter(config map[string]interface{}) *v2.HealthCheckFilter {
	healthcheck := &v2.HealthCheckFilter{}

	//passthrough
	if passthrough, ok := config["passthrough"]; ok {
		if passthrough, ok := passthrough.(bool); ok {
			healthcheck.PassThrough = passthrough
		} else {
			log.StartLogger.Fatalln("[passthrough] in health check filter config is not bool")
		}
	} else {
		log.StartLogger.Fatalln("[passthrough] is required in healthcheck filter config")
	}

	//cache time
	if cacheTime, ok := config["cache_time"]; ok {
		if cacheTime, ok := cacheTime.(string); ok {
			if duration, error := time.ParseDuration(strings.Trim(cacheTime, `"`)); error == nil {
				healthcheck.CacheTime = duration
			} else {
				log.StartLogger.Fatalln("[cache_time] in health check filter is not valid ,", error)
			}
		} else {
			log.StartLogger.Fatalln("[cache_time] in health check filter config is not a numeric string")
		}
	} else {
		log.StartLogger.Fatalln("[cache_time] is required in healthcheck filter config")
	}

	//cluster_min_healthy_percentagesp
	if clusterMinHealthyPercentage, ok := config["cluster_min_healthy_percentages"]; ok {
		if clusterMinHealthyPercentage, ok := clusterMinHealthyPercentage.(map[string]interface{}); ok {
			healthcheck.ClusterMinHealthyPercentage = make(map[string]float32)
			for cluster, percent := range clusterMinHealthyPercentage {
				healthcheck.ClusterMinHealthyPercentage[cluster] = float32(percent.(float64))
			}
		} else {
			log.StartLogger.Fatalln("[passthrough] in health check filter config is not bool")
		}
	} else {
		log.StartLogger.Fatalln("[passthrough] is required in healthcheck filter config")
	}

	return healthcheck
}

// ParseListenerConfig
func ParseListenerConfig(c *ListenerConfig, inheritListeners []*v2.ListenerConfig) *v2.ListenerConfig {
	if c.Name == "" {
		log.StartLogger.Fatalln("[name] is required in listener config")
	}

	if c.Address == "" {
		log.StartLogger.Fatalln("[Address] is required in listener config")
	}
	addr, err := net.ResolveTCPAddr("tcp", c.Address)

	if err != nil {
		log.StartLogger.Fatalln("[Address] not valid:", c.Address)
	}

	//try inherit legacy listener
	currentIP := net.ParseIP(addr.String())
	var old *net.TCPListener

	for _, il := range inheritListeners {
		inheritIP := net.ParseIP(il.Addr.String())

		// use ip.Equal to solve ipv4 and ipv6 case
		if inheritIP.Equal(currentIP) {
			log.StartLogger.Infof("inherit listener addr: %s", c.Address)
			old = il.InheritListener
			il.Remain = true
			break
		}
	}

	return &v2.ListenerConfig{
		Name:                                  c.Name,
		Addr:                                  addr,
		BindToPort:                            c.BindToPort,
		Inspector:                             c.Inspector,
		InheritListener:                       old,
		PerConnBufferLimitBytes:               1 << 15,
		LogPath:                               c.LogPath,
		LogLevel:                              uint8(parseLogLevel(c.LogLevel)),
		AccessLogs:                            parseAccessConfig(c.AccessLogs),
		HandOffRestoredDestinationConnections: c.HandOffRestoredDestinationConnections,
		FilterChains:                          parseFilterChains(c.FilterChains),
	}
}

// ParseClusterConfig
func ParseClusterConfig(clusters []ClusterConfig) ([]v2.Cluster, map[string][]v2.Host) {
	if len(clusters) == 0 {
		log.StartLogger.Warnf("No Cluster provided in cluster config")
	}

	var clustersV2 []v2.Cluster
	clusterV2Map := make(map[string][]v2.Host)

	for _, c := range clusters {
		// cluster name
		if c.Name == "" {
			log.StartLogger.Fatalln("[name] is required in cluster config")
		}

		var clusterType v2.ClusterType

		//cluster type
		if c.Type == "" {
			log.StartLogger.Fatalln("[type] is required in cluster config")
		} else {
			if ct, ok := clusterTypeMap[c.Type]; ok {
				clusterType = ct
			} else {
				log.StartLogger.Fatalln("unknown cluster type:", c.Type)
			}
		}

		var lbType v2.LbType

		if c.LbType == "" {
			log.StartLogger.Fatalln("[lb_type] is required in cluster config")
		} else {
			if lt, ok := lbTypeMap[c.LbType]; ok {
				lbType = lt
			} else {
				log.StartLogger.Fatalln("unknown lb type:", c.LbType)
			}
		}

		if c.MaxRequestPerConn == 0 {
			c.MaxRequestPerConn = 1024
			log.StartLogger.Infof("[max_request_per_conn] is not specified, use default value %d", 1024)
		}

		if c.ConnBufferLimitBytes == 0 {
			c.ConnBufferLimitBytes = 16 * 1026
			log.StartLogger.Infof("[conn_buffer_limit_bytes] is not specified, use default value %d", 1024*16)
		}

		//clusterSpec := c.ClusterSpecConfig.(ClusterSpecConfig)
		clusterSpec := c.ClusterSpecConfig

		// checkout LBSubsetConfig
		if c.LBSubsetConfig.FallBackPolicy > 2 {
			log.StartLogger.Panic("lb subset config 's fall back policy set error. " +
				"For 0, represent NO_FALLBACK" +
				"For 1, represent ANY_ENDPOINT" +
				"For 2, represent DEFAULT_SUBSET")
		}

		//v2.Cluster
		clusterV2 := v2.Cluster{
			Name:                 c.Name,
			ClusterType:          clusterType,
			LbType:               lbType,
			MaxRequestPerConn:    c.MaxRequestPerConn,
			ConnBufferLimitBytes: c.ConnBufferLimitBytes,

			HealthCheck:      parseClusterHealthCheckConf(&c.HealthCheck),
			CirBreThresholds: parseCircuitBreakers(c.CircuitBreakers),

			Spec: parseConfigSpecConfig(&clusterSpec),
			LBSubSetConfig: v2.LBSubsetConfig{
				c.LBSubsetConfig.FallBackPolicy,
				c.LBSubsetConfig.DefaultSubset,
				c.LBSubsetConfig.SubsetSelectors,
			},

			TLS: parseTLSConfig(&c.TLS),
		}

		clustersV2 = append(clustersV2, clusterV2)
		hostV2 := parseHostConfig(&c)
		clusterV2Map[c.Name] = hostV2
	}

	// trigger all callbacks
	if cbs, ok := configParsedCBMaps[ParseCallbackKeyCluster]; ok {
		for _, cb := range cbs {
			cb(clustersV2, false)
		}
	}

	return clustersV2, clusterV2Map
}

func parseClusterHealthCheckConf(c *ClusterHealthCheckConfig) v2.HealthCheck {

	var healthcheckInstance v2.HealthCheck

	if c.Protocol == "" {
		log.StartLogger.Warnf("healthcheck for cluster is disabled")

	} else if _, ok := protocolsSupported[c.Protocol]; ok {
		healthcheckInstance = v2.HealthCheck{
			Protocol:           c.Protocol,
			Timeout:            c.Timeout.Duration,
			Interval:           c.Interval.Duration,
			IntervalJitter:     c.IntervalJitter.Duration,
			HealthyThreshold:   c.HealthyThreshold,
			UnhealthyThreshold: c.UnhealthyThreshold,
			CheckPath:          c.CheckPath,
			ServiceName:        c.ServiceName,
		}
	} else {
		log.StartLogger.Fatal("unsupported health check protocol:", c.Protocol)
	}

	return healthcheckInstance
}

func parseCircuitBreakers(cbcs []*CircuitBreakerConfig) v2.CircuitBreakers {
	var cb v2.CircuitBreakers
	var rp v2.RoutingPriority

	for _, cbc := range cbcs {
		if strings.ToLower(cbc.Priority) == "default" {
			rp = v2.DEFAULT
		} else {
			rp = v2.HIGH
		}

		if 0 == cbc.MaxConnections || 0 == cbc.MaxPendingRequests ||
			0 == cbc.MaxRequests || 0 == cbc.MaxRetries {
			log.StartLogger.Warnf("zero is set in circuitBreakers' config")
		}

		threshold := v2.Thresholds{
			Priority:           rp,
			MaxConnections:     cbc.MaxConnections,
			MaxPendingRequests: cbc.MaxPendingRequests,
			MaxRequests:        cbc.MaxRequests,
			MaxRetries:         cbc.MaxRetries,
		}

		cb.Thresholds = append(cb.Thresholds, threshold)
	}

	return cb
}

func parseConfigSpecConfig(c *ClusterSpecConfig) v2.ClusterSpecInfo {
	var specs []v2.SubscribeSpec

	for _, sub := range c.Subscribes {
		specs = append(specs, v2.SubscribeSpec{
			ServiceName: sub.ServiceName,
		})
	}

	return v2.ClusterSpecInfo{
		Subscribes: specs,
	}
}

func parseHostConfig(c *ClusterConfig) []v2.Host {
	// host maybe nil when rewriting config
	//if c.Hosts == nil || len(c.Hosts) == 0 {
	//	log.StartLogger.Debugf("[hosts] is required in cluster config")
	//}
	var hosts []v2.Host
	for _, host := range c.Hosts {

		if host.Address == "" {
			log.StartLogger.Fatalln("[host.address] is required in host config")
		}

		hosts = append(hosts, v2.Host{
			host.Address,
			host.Hostname,
			getHostWeight(host.Weight),
			parseRouterMetadata(host.MetaData),
		})
	}

	return hosts
}

func getHostWeight(weight uint32) uint32 {
	if weight > MaxHostWeight {
		weight = MaxHostWeight
	}

	if weight < MinHostWeight {
		weight = MinHostWeight
	}

	return weight
}

// ParseServiceRegistry
func ParseServiceRegistry(src ServiceRegistryConfig) {
	var SrvRegInfo v2.ServiceRegistryInfo

	if src.ServiceAppInfo.AppName == "" {
		//log.StartLogger.Debugf("[ParseServiceRegistry] appname is nil")
	}

	srvappinfo := v2.ApplicationInfo{
		AntShareCloud: src.ServiceAppInfo.AntShareCloud,
		DataCenter:    src.ServiceAppInfo.DataCenter,
		AppName:       src.ServiceAppInfo.AppName,
	}

	var SrvPubInfoArray []v2.PublishInfo

	for _, pubs := range src.ServicePubInfo {
		SrvPubInfoArray = append(SrvPubInfoArray, v2.PublishInfo{
			Pub: v2.PublishContent{
				ServiceName: pubs.ServiceName,
				PubData:     pubs.PubData,
			},
		})
	}

	SrvRegInfo = v2.ServiceRegistryInfo{
		srvappinfo,
		SrvPubInfoArray,
	}

	//trigger all callbacks
	if cbs, ok := configParsedCBMaps[ParseCallbackKeyServiceRgtInfo]; ok {
		for _, cb := range cbs {
			cb(SrvRegInfo, true)
		}
	}
}

// ParseTCPProxy transfer map to TCPProxy
func ParseTCPProxy(config map[string]interface{}) (*v2.TCPProxy, error) {
	data, _ := json.Marshal(config)
	cfg := &TCPProxyConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config is not a tcp proxy config: %v", err)
	}
	proxy := &v2.TCPProxy{
		Routes: []*v2.TCPRoute{},
	}
	for _, route := range cfg.Routes {
		tcpRoute := &v2.TCPRoute{
			Cluster: route.Cluster,
		}
		for _, addr := range route.SourceAddrs {
			src, err := net.ResolveTCPAddr("tcp", addr)
			if err != nil {
				return nil, fmt.Errorf("source addrs is not a valid addr: %v", err)
			}
			tcpRoute.SourceAddrs = append(tcpRoute.SourceAddrs, src)
		}
		for _, addr := range route.DestinationAddrs {
			dst, err := net.ResolveTCPAddr("tcp", addr)
			if err != nil {
				return nil, fmt.Errorf("destination addrs is not a valid addr: %v", err)
			}
			tcpRoute.DestinationAddrs = append(tcpRoute.DestinationAddrs, dst)
		}
		proxy.Routes = append(proxy.Routes, tcpRoute)
	}
	return proxy, nil
}
