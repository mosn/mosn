package v1

import (
	"fmt"
	"strings"
	"strconv"
	"errors"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	google_protobuf "github.com/gogo/protobuf/types"
	envoy_type "github.com/envoyproxy/go-control-plane/envoy/type"
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoy_api_v2_cluster "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"

	"time"
)

const (
	TCP_SCHEME string = "tcp://"
	UNIX_SCHEME string = "unix://"
)

func translateAddress(addressV1 string, url, resolved bool, addressV2 *core.Address) (err error) {
	var host, port, path string = "", "", ""

	if resolved {
		if url {
			if strings.HasPrefix(addressV1, TCP_SCHEME) {
				host, port, err = parseInternetAddressAndPort(addressV1)
				if err != nil {
					return err
				}
			}else if strings.HasPrefix(addressV1, UNIX_SCHEME) {
                path, err = parsePipePath(addressV1)
                if err != nil {
                	return err
				}
			}

		}else{
			host, port, err = parseInternetAddressAndPort(addressV1)
			if err != nil {
				return err
			}
		}
	}

	host, port, err = getHostAndPortFromUrl(addressV1)
	if err != nil {
		return err
	}

	if host != "" && port != ""{

		portValue := &core.SocketAddress_PortValue{}
		p, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		portValue.PortValue = uint32(p)

		socketAddress := &core.SocketAddress{}
		socketAddress.PortSpecifier = portValue
		socketAddress.Address = host

		addressV2.Address = &core.Address_SocketAddress{socketAddress}

	}else if path != "" {
		pipe := &core.Pipe{path}
		addressV2.Address = &core.Address_Pipe{pipe}
	}
	return nil
}

func parseInternetAddressAndPort(address string) (host, port string, err error) {

	addressTmp := address[len(TCP_SCHEME):]
	if strings.HasPrefix(address, "[") {
		// ipv6
		idx := strings.LastIndex(addressTmp, "]:")
		if idx == -1 {
			err = errors.New(fmt.Sprintf("invaild address %s\n", address))
			return host, port , err
		}
		host = addressTmp[1:idx]
		port = addressTmp[idx+2:]
	}else{
		idx := strings.LastIndex(addressTmp, ":")
		if idx == -1 {
			err = errors.New(fmt.Sprintf("invaild address %s\n", address))
			return host, port , err
		}
		host = addressTmp[:idx]
		port = addressTmp[idx+1:]
	}
	return host, port, nil
}

func parsePipePath(address string) (path string, err error) {
	path = address[len(UNIX_SCHEME):]
	return path, nil
}

func getHostAndPortFromUrl(address string) (host, port string, err error) {
	if !strings.HasPrefix(address, TCP_SCHEME) {
		err = errors.New(fmt.Sprintf("invaild address %s\n", address))
		return host, port , err
	}
	addressTmp := address[len(TCP_SCHEME):]
	idx := strings.LastIndex(addressTmp, ":")
	if idx == -1 {
		err = errors.New(fmt.Sprintf("invaild address %s\n", address))
		return host, port , err
	}
	host = addressTmp[:idx]
	port = addressTmp[idx+1:]
	return host, port, nil
}

func translateVirtualHost(virtualHostV1 *VirtualHost, virtualHostV2 *envoy_api_v2_route.VirtualHost) (err error) {
	virtualHostV2.Name = virtualHostV1.Name
	for _, domain := range virtualHostV1.Domains {
		virtualHostV2.Domains = append(virtualHostV2.Domains, domain)
	}

	for _, routeV1 := range virtualHostV1.Routes {
		routeV2 := envoy_api_v2_route.Route{}
		err = translateRoute(routeV1, &routeV2)
		if err != nil {
			return err
		}
		virtualHostV2.Routes = append(virtualHostV2.Routes, routeV2)
	}

	return nil
}

func translateRoute(routeV1 *HTTPRoute, routeV2 *envoy_api_v2_route.Route) (err error) {
	routeV2.Match = envoy_api_v2_route.RouteMatch{}
	if routeV1.Prefix != "" {
		routeV2.Match.PathSpecifier = &envoy_api_v2_route.RouteMatch_Prefix{routeV1.Prefix}
	} else if routeV1.Path != "" {
		routeV2.Match.PathSpecifier = &envoy_api_v2_route.RouteMatch_Path{routeV1.Path}
	} else if routeV1.Regex != "" {
		routeV2.Match.PathSpecifier = &envoy_api_v2_route.RouteMatch_Regex{routeV1.Regex}
	} else {
		err = errors.New("routes must specify one of prefix/path/regex")
		return err
	}
	routeV2.Match.CaseSensitive = &google_protobuf.BoolValue{routeV1.CaseSensitive}
	routeV2.Match.Runtime = &core.RuntimeUInt32{}
	if routeV1.Runtime != nil {
		routeV2.Match.Runtime.RuntimeKey = routeV1.Runtime.Key
		routeV2.Match.Runtime.DefaultValue = uint32(routeV1.Runtime.Default)
	}

	routeV2.Match.Headers = make([]*envoy_api_v2_route.HeaderMatcher, 0, len(routeV1.Headers))
	for _, headerV1 := range routeV1.Headers {
		headerMatch := &envoy_api_v2_route.HeaderMatcher{}
		headerMatch.Name = headerV1.Name
		if headerV1.Regex {
			headerMatch.HeaderMatchSpecifier = &envoy_api_v2_route.HeaderMatcher_RegexMatch{headerV1.Value}
		} else if headerV1.Value != "" {
			headerMatch.HeaderMatchSpecifier = &envoy_api_v2_route.HeaderMatcher_ExactMatch{headerV1.Value}
		} else if headerV1.RangeMatch != nil {
			start := int64(headerV1.RangeMatch.start)
			end := int64(headerV1.RangeMatch.end)
			headerMatch.HeaderMatchSpecifier = &envoy_api_v2_route.HeaderMatcher_RangeMatch{&envoy_type.Int64Range{start, end}}
		}
		routeV2.Match.Headers = append(routeV2.Match.Headers, headerMatch)
	}

	routeAction := &envoy_api_v2_route.RouteAction{}
	if routeV1.Cluster != "" {
		routeAction.ClusterSpecifier = &envoy_api_v2_route.RouteAction_Cluster{routeV1.Cluster}
	}else if routeV1.ClusterHeader != "" {
		routeAction.ClusterSpecifier = &envoy_api_v2_route.RouteAction_ClusterHeader{routeV1.ClusterHeader}
	}else if routeV1.WeightedClusters != nil {
		weightedClusters := &envoy_api_v2_route.WeightedCluster{}
		weightedClusters.Clusters = make([]*envoy_api_v2_route.WeightedCluster_ClusterWeight, 0, len(routeV1.WeightedClusters.ClustersV1))
		for _, cluster := range routeV1.WeightedClusters.ClustersV1 {
			weightedClusters.RuntimeKeyPrefix = routeV1.WeightedClusters.RuntimeKeyPrefix
			clusterV2 := &envoy_api_v2_route.WeightedCluster_ClusterWeight{}
			clusterV2.Name = cluster.Name
			clusterV2.Weight = &google_protobuf.UInt32Value{uint32(cluster.Weight)}
			weightedClusters.Clusters = append(weightedClusters.Clusters, clusterV2)
		}
		routeAction.ClusterSpecifier = &envoy_api_v2_route.RouteAction_WeightedClusters{weightedClusters}
	}
	routeV2.Action = &envoy_api_v2_route.Route_Route{routeAction}

	return nil
}

func setDnsHost(clusterV1 *Cluster, url, resolved bool, clusterV2 *xdsapi.Cluster) (err error) {
	clusterV2.Hosts = make([]*core.Address, 0, len(clusterV1.Hosts))
	for _, hostV1 := range clusterV1.Hosts {
		hostV2 := &core.Address{}
		err = translateAddress(hostV1.URL, true, true, hostV2)
		if err != nil {
			return err
		}
		clusterV2.Hosts = append(clusterV2.Hosts, hostV2)
	}
	return nil
}

func translateCircuitBreaker(circuitBreakerV1 *CircuitBreaker, circuitBreakerV2 *envoy_api_v2_cluster.CircuitBreakers) (err error){
	circuitBreakerV2.Thresholds = make([]*envoy_api_v2_cluster.CircuitBreakers_Thresholds, 0, 2)

	defaultThreshold := envoy_api_v2_cluster.CircuitBreakers_Thresholds{}
	defaultThreshold.Priority = core.RoutingPriority_DEFAULT
	defaultThreshold.MaxConnections = &google_protobuf.UInt32Value{uint32(circuitBreakerV1.Default.MaxConnections)}
	defaultThreshold.MaxPendingRequests = &google_protobuf.UInt32Value{uint32(circuitBreakerV1.Default.MaxPendingRequests)}
	defaultThreshold.MaxRequests = &google_protobuf.UInt32Value{uint32(circuitBreakerV1.Default.MaxRequests)}
	defaultThreshold.MaxRetries = &google_protobuf.UInt32Value{uint32(circuitBreakerV1.Default.MaxRetries)}
	circuitBreakerV2.Thresholds = append(circuitBreakerV2.Thresholds, &defaultThreshold)

	highThreshold := envoy_api_v2_cluster.CircuitBreakers_Thresholds{}
	highThreshold.Priority = core.RoutingPriority_HIGH
	highThreshold.MaxConnections = &google_protobuf.UInt32Value{uint32(circuitBreakerV1.High.MaxConnections)}
	highThreshold.MaxPendingRequests = &google_protobuf.UInt32Value{uint32(circuitBreakerV1.High.MaxPendingRequests)}
	highThreshold.MaxRequests = &google_protobuf.UInt32Value{uint32(circuitBreakerV1.High.MaxRequests)}
	highThreshold.MaxRetries = &google_protobuf.UInt32Value{uint32(circuitBreakerV1.High.MaxRetries)}
	circuitBreakerV2.Thresholds = append(circuitBreakerV2.Thresholds, &highThreshold)

	return nil
}

func translateOutlierDetection(outlierDetectionV1 *OutlierDetection, outlierDetectionV2 *envoy_api_v2_cluster.OutlierDetection) (err error){

	outlierDetectionV2.Interval = translatePBDuration(outlierDetectionV1.IntervalMS * int64(time.Millisecond))
	outlierDetectionV2.BaseEjectionTime = translatePBDuration(outlierDetectionV1.BaseEjectionTimeMS * int64(time.Millisecond))
	outlierDetectionV2.Consecutive_5Xx = &google_protobuf.UInt32Value{uint32(outlierDetectionV1.ConsecutiveErrors)}
	outlierDetectionV2.MaxEjectionPercent = &google_protobuf.UInt32Value{uint32(outlierDetectionV1.MaxEjectionPercent)}
	outlierDetectionV2.ConsecutiveGatewayFailure = &google_protobuf.UInt32Value{uint32(outlierDetectionV1.ConsecutiveGatewayFailure)}
	outlierDetectionV2.EnforcingConsecutive_5Xx = &google_protobuf.UInt32Value{uint32(outlierDetectionV1.EnforcingConsecutive5xx)}
	outlierDetectionV2.EnforcingConsecutiveGatewayFailure =  &google_protobuf.UInt32Value{uint32(outlierDetectionV1.EnforcingConsecutiveGatewayFailure)}
	outlierDetectionV2.EnforcingSuccessRate = &google_protobuf.UInt32Value{uint32(outlierDetectionV1.EnforcingSuccessRate)}
	outlierDetectionV2.SuccessRateMinimumHosts = &google_protobuf.UInt32Value{uint32(outlierDetectionV1.SuccessRateMinimumHosts)}
	outlierDetectionV2.SuccessRateRequestVolume = &google_protobuf.UInt32Value{uint32(outlierDetectionV1.SuccessRateRequestVolume)}
	outlierDetectionV2.SuccessRateStdevFactor = &google_protobuf.UInt32Value{uint32(outlierDetectionV1.SuccessRateStdevFactor)}
	return nil
}

func translatePBDuration(nanoseconds int64) *google_protobuf.Duration{
	seconds := nanoseconds / int64(time.Second)
	var nanos int32 = int32(nanoseconds % int64(time.Second))
	return &google_protobuf.Duration{seconds, nanos}
}