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
		policy: &routerPolicy{
			retryOn:      false,
			retryTimeout: 0,
			numRetries:   0,
		},
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
	policy                      *routerPolicy
	virtualClusters             *VirtualClusterEntry
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

	return nil
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

	return rri.vHost
}

func (rri *RouteRuleImplBase) VirtualCluster(headers map[string]string) types.VirtualCluster {

	return rri.virtualClusters
}

func (rri *RouteRuleImplBase) Policy() types.Policy {

	return rri.policy
}

func (rri *RouteRuleImplBase) MetadataMatcher() types.MetadataMatcher {

	return nil
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

type SofaRouteRuleImpl struct {
	RouteRuleImplBase
	matchValue string
}

func (srri *SofaRouteRuleImpl) Matcher() string {

	return srri.matchValue
}

func (srri *SofaRouteRuleImpl) MatchType() types.PathMatchType {

	return types.SofaHeader
}

func (srri *SofaRouteRuleImpl) Match(headers map[string]string, randomValue uint64) types.Route {
	if value, ok := headers[types.SofaRouteMatchKey]; ok {
		if value == srri.matchValue {
			return srri
		}

		log.DefaultLogger.Debugf("Sofa Router Matched")

	}

	return nil
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

type routerPolicy struct {
	retryOn      bool
	retryTimeout time.Duration
	numRetries   uint32
}

func (p *routerPolicy) RetryOn() bool {
	return p.retryOn
}

func (p *routerPolicy) TryTimeout() time.Duration {
	return p.retryTimeout
}

func (p *routerPolicy) NumRetries() uint32 {
	return p.numRetries
}

func (p *routerPolicy) RetryPolicy() types.RetryPolicy {
	return p
}

func (p *routerPolicy) ShadowPolicy() types.ShadowPolicy {
	return nil
}

func (p *routerPolicy) CorsPolicy() types.CorsPolicy {
	return nil
}

func (p *routerPolicy) LoadBalancerPolicy() types.LoadBalancerPolicy {
	return nil
}
