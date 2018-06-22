package router

import (
	"regexp"
	"strings"
	"time"

	multimap "github.com/jwangsadinata/go-multimap/slicemultimap"
	"github.com/markphelps/optional"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	httpmosn "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/http"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func init_test() {
	RegisteRouterConfigFactory(protocol.SofaRpc, NewRouterMatcher)
	RegisteRouterConfigFactory(protocol.Http2, NewRouterMatcher)
}

func NewRouterMatcher(config interface{}) (types.Routers, error) {
	routerMatcher := &RouteMatcher{
		virtualHosts: make(map[string]types.VirtualHost),
	}

	if config, ok := config.(*v2.Proxy); ok {

		for _, virtualHost := range config.VirtualHosts {

			//todo 补充virtual host 其他成员
			vh := NewVirtualHostImpl(virtualHost, config.ValidateClusters)

			for _, domain := range virtualHost.Domains {
				if domain == "*" {
					if routerMatcher.defaultVirtualHost != nil {
						log.DefaultLogger.Errorf("Only a single wildcard domain permitted")
					}
					
					routerMatcher.defaultVirtualHost = vh
				} else if  _,ok := routerMatcher.virtualHosts[domain]; ok {
					log.DefaultLogger.Errorf("Only unique values for domains are permitted, get duplicate domain = %s",
						domain)
				} else {
					routerMatcher.virtualHosts[domain] = vh
				}
			}
		}

	}

	return routerMatcher, nil
}

// A router wrapper used to matches an incoming request headers to a backend cluster
type RouteMatcher struct {
	virtualHosts                map[string]types.VirtualHost // key: host
	defaultVirtualHost          types.VirtualHost
	wildcardVirtualHostSuffixes map[int64]map[string]types.VirtualHost
}

// Routing with Virtual Host
func (rm *RouteMatcher) Route(headers map[string]string, randomValue uint64) types.Route {
	// First Step: Select VirtualHost with "host" in Headers form VirtualHost Array
	virtualHost := rm.findVirtualHost(headers)

	if virtualHost == nil {
		log.DefaultLogger.Warnf("No VirtualHost Found when Routing %+v", headers)

		return nil
	}

	// Second Step: Match Route from Routes in a Virtual Host
	return virtualHost.GetRouteFromEntries(headers, randomValue)
}

func (rm *RouteMatcher) findVirtualHost(headers map[string]string) types.VirtualHost {
	if len(rm.virtualHosts) == 0 && rm.defaultVirtualHost != nil {

		return rm.defaultVirtualHost
	}

	host := strings.ToLower(headers[protocol.MosnHeadersHostKey])

	// for service, header["host"] == header["service"] == servicename
	// or use only a unique key for sofa's virtual host
	if virtualHost, ok := rm.virtualHosts[host]; ok {
		return virtualHost
	}

	// todo support WildcardVirtualHost

	return rm.defaultVirtualHost
}

// todo match wildcard
func (rm *RouteMatcher) findWildcardVirtualHost(host string) types.VirtualHost {

	return nil
}

func (rm *RouteMatcher) AddRouter(routerName string) {}

func (rm *RouteMatcher) DelRouter(routerName string) {}

func NewVirtualHostImpl(virtualHost *v2.VirtualHost, validateClusters bool) *VirtualHostImpl {

	var virtualHostImpl = &VirtualHostImpl{virtualHostName: virtualHost.Name}

	switch virtualHost.RequireTls {
	case "EXTERNALONLY":
		virtualHostImpl.sslRequirements = types.EXTERNALONLY
	case "ALL":
		virtualHostImpl.sslRequirements = types.ALL
	default:
		virtualHostImpl.sslRequirements = types.NONE
	}

	for _, route := range virtualHost.Routers {

		if route.Match.Prefix != "" {

			virtualHostImpl.routes = append(virtualHostImpl.routes, &PrefixRouteEntryImpl{
				NewRouteRuleImplBase(virtualHostImpl, &route),
				route.Match.Prefix,
			})

		} else if route.Match.Path != "" {
			virtualHostImpl.routes = append(virtualHostImpl.routes, &PathRouteRuleImpl{
				NewRouteRuleImplBase(virtualHostImpl, &route),
				route.Match.Path,
			})

		} else if route.Match.Regex != "" {

			if regPattern, err := regexp.Compile(route.Match.Prefix); err == nil {
				virtualHostImpl.routes = append(virtualHostImpl.routes, &RegexRouteEntryImpl{
					NewRouteRuleImplBase(virtualHostImpl, &route),
					route.Match.Prefix,
					*regPattern,
				})
			} else {
				log.DefaultLogger.Errorf("Compile Regex Error")
			}
		}
	}

	// todo check cluster's validity
	if validateClusters {
	}

	// Add Virtual Cluster
	for _, vc := range virtualHost.VirtualClusters {

		if regxPattern, err := regexp.Compile(vc.Pattern); err == nil {
			virtualHostImpl.virtualClusters = append(virtualHostImpl.virtualClusters,
				VirtualClusterEntry{
					name:    vc.Name,
					method:  optional.NewString(vc.Method),
					pattern: *regxPattern,
				})
		} else {
			log.DefaultLogger.Errorf("Compile Error")
		}
	}

	return virtualHostImpl
}

type VirtualHostImpl struct {
	virtualHostName       string
	routes                []RouteBase //route impl
	virtualClusters       []VirtualClusterEntry
	sslRequirements       types.SslRequirements
	corsPolicy            types.CorsPolicy
	globalRouteConfig     *ConfigImpl
	requestHeadersParser  *HeaderParser
	responseHeadersParser *HeaderParser
}

type VirtualClusterEntry struct {
	pattern regexp.Regexp
	method  optional.String
	name    string
}

func (vce *VirtualClusterEntry) VirtualClusterName() string {

	return vce.name
}

func (vh *VirtualHostImpl) Name() string {

	return vh.virtualHostName
}

func (vh *VirtualHostImpl) CorsPolicy() types.CorsPolicy {

	return nil
}

func (vh *VirtualHostImpl) RateLimitPolicy() types.RateLimitPolicy {

	return nil
}

func (vh *VirtualHostImpl) GetRouteFromEntries(headers map[string]string, randomValue uint64) types.Route {
	// todo check tls
	for _, route := range vh.routes {

		if routeEntry := route.Match(headers, randomValue); routeEntry != nil {
			return routeEntry
		}
	}

	return nil
}

func NewRouteRuleImplBase(vHost *VirtualHostImpl, route *v2.Router) RouteRuleImplBase {
	return RouteRuleImplBase{
		vHost:        vHost,
		routerMatch:  route.Match,
		routerAction: route.Route,
		metaData:     route.Metadata,
	}
}

// Base implementation for all route entries.
type RouteRuleImplBase struct {
	caseSensitive               bool
	prefixRewrite               string
	hostRewrite                 string
	includeVirtualHostRateLimit bool
	corsPolicy                  types.CorsPolicy //todo
	vHost                       *VirtualHostImpl
	autoHostRewrite             bool
	useWebSocket                bool
	clusterName                 string //
	clusterHeaderName           LowerCaseString
	clusterNotFoundResponseCode httpmosn.HttpCode
	timeout                     time.Duration
	runtime                     v2.RuntimeUInt32
	hostRedirect                string
	pathRedirect                string
	httpsRedirect               bool
	retryPolicy                 *RetryPolicyImpl
	rateLimitPolicy             *RateLimitPolicyImpl
	routerAction                v2.RouteAction
	routerMatch                 v2.RouterMatch
	shadowPolicy                *ShadowPolicyImpl
	priority                    types.ResourcePriority
	configHeaders               []*types.HeaderData //
	configQueryParameters       []types.QueryParameterMatcher
	weightedClusters            []*WeightedClusterEntry
	totalClusterWeight          uint64
	hashPolicy                  HashPolicyImpl
	metadataMatchCriteria       *MetadataMatchCriteriaImpl
	requestHeadersParser        *HeaderParser
	responseHeadersParser       *HeaderParser
	metaData                    v2.Metadata
	opaqueConfig                multimap.MultiMap
	decorator                   *types.Decorator
	directResponseCode          httpmosn.HttpCode
	directResponseBody          string
}

// types.RouterInfo
func (rri *RouteRuleImplBase) GetRouterName() string {

	return ""
}

// types.Route
func (rri *RouteRuleImplBase) RedirectRule() types.RedirectRule {

	return rri.RedirectRule()
}

func (rri *RouteRuleImplBase) RouteRule() types.RouteRule {

	return rri
}

func (rri *RouteRuleImplBase) TraceDecorator() types.TraceDecorator {

	return rri.TraceDecorator()
}

// types.RouteRule
// Select Cluster for Routing
// todo support weighted cluster
func (rri *RouteRuleImplBase) ClusterName() string {

	return rri.routerAction.ClusterName
}

func (rri *RouteRuleImplBase) GlobalTimeout() time.Duration {

	return rri.routerAction.Timeout
}

func (rri *RouteRuleImplBase) Priority() types.Priority {

	return 0
}

func (rri *RouteRuleImplBase) VirtualHost() types.VirtualHost {

	return rri.VirtualHost()
}

func (rri *RouteRuleImplBase) VirtualCluster(headers map[string]string) types.VirtualCluster {

	return rri.VirtualCluster(headers)
}

func (rri *RouteRuleImplBase) Policy() types.Policy {

	return rri.Policy()
}

func (rri *RouteRuleImplBase) MetadataMatcher() types.MetadataMatcher {

	return rri.MetadataMatcher()
}

// todo
func (rri *RouteRuleImplBase) finalizePathHeader(headers map[string]string, matchedPath string) {

}

func (rri *RouteRuleImplBase) matchRoute(headers map[string]string, randomValue uint64) bool {

	// todo check runtime
	// 1. match headers' KV
	if !ConfigUtilityInst.MatchHeaders(headers, rri.configHeaders) {

		return false
	}

	// 2. match query parameters
	var queryParams types.QueryParams

	if QueryString, ok := headers[protocol.MosnHeaderQueryStringKey]; ok {
		queryParams = httpmosn.ParseQueryString(QueryString)
	}

	if len(queryParams) == 0 {

		return true
	} else {

		return ConfigUtilityInst.MatchQueryParams(&queryParams, rri.configQueryParameters)
	}

	return true
}

type PathRouteRuleImpl struct {
	RouteRuleImplBase
	path string
}

func (prri *PathRouteRuleImpl) Matcher() string {

	return prri.path
}

func (prri *PathRouteRuleImpl) MatchType() types.PathMatchType {

	return types.Exact
}

// Exact Path Comparing
func (prri *PathRouteRuleImpl) Match(headers map[string]string, randomValue uint64) types.Route {
	// match base rule first
	if prri.matchRoute(headers, randomValue) {

		if headerPathValue, ok := headers[protocol.MosnHeaderPathKey]; ok {

			if prri.caseSensitive {
				if headerPathValue == prri.path {

					return prri
				}
			} else if strings.EqualFold(headerPathValue, prri.path) {

				return prri
			}
		}
	}

	return nil
}

// todo
func (prri *PathRouteRuleImpl) FinalizeRequestHeaders(headers map[string]string, requestInfo types.RequestInfo) {
	prri.finalizePathHeader(headers, prri.path)
}

type PrefixRouteEntryImpl struct {
	RouteRuleImplBase
	prefix string
}

func (prei *PrefixRouteEntryImpl) Matcher() string {

	return prei.prefix
}

func (prei *PrefixRouteEntryImpl) MatchType() types.PathMatchType {

	return types.Prefix
}

// Compare Path's Prefix
func (prei *PrefixRouteEntryImpl) Match(headers map[string]string, randomValue uint64) types.Route {

	if prei.matchRoute(headers, randomValue) {

		if headerPathValue, ok := headers[protocol.MosnHeaderPathKey]; ok {

			if strings.HasPrefix(headerPathValue, prei.prefix) {

				return prei
			}
		}
	}

	return nil
}

func (prei *PrefixRouteEntryImpl) FinalizeRequestHeaders(headers map[string]string, requestInfo types.RequestInfo) {
	prei.finalizePathHeader(headers, prei.prefix)
}

//
type RegexRouteEntryImpl struct {
	RouteRuleImplBase
	regexStr     string
	regexPattern regexp.Regexp
}

func (rrei *RegexRouteEntryImpl) Matcher() string {

	return rrei.regexStr
}

func (rrei *RegexRouteEntryImpl) MatchType() types.PathMatchType {

	return types.Regex
}

func (rrei *RegexRouteEntryImpl) Match(headers map[string]string, randomValue uint64) types.Route {

	if headerPathValue, ok := headers[protocol.MosnHeaderPathKey]; ok {
		if rrei.regexPattern.MatchString(headerPathValue) {

			return rrei
		}
	}

	return nil
}

func (rrei *RegexRouteEntryImpl) FinalizeRequestHeaders(headers map[string]string, requestInfo types.RequestInfo) {
	rrei.finalizePathHeader(headers, rrei.regexStr)
}
