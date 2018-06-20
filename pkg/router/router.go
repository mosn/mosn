package router

import (
	"strings"
	"time"
	
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/http"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func init_test() {
	RegisteRouterConfigFactory(protocol.SofaRpc, NewRouterMatcher)
	RegisteRouterConfigFactory(protocol.Http2, NewRouterMatcher)
}


func NewRouterMatcher(config interface{}) (types.Routers,error) {
	
	routerMatcher := &RouteMatcher{
		virtualHosts: make(map[string]types.VirtualHost),
	}
	
	if config, ok := config.(*v2.Proxy); ok {
		
		for _, virtualHost := range config.VirtualHosts {
			
			vh := &VirtualHostImpl{
				VirtualHostName: virtualHost.Name,
			}
			routerMatcher.virtualHosts[vh.VirtualHostName] = vh
		}
		
	}
	return routerMatcher,nil
}

type RouteMatcher struct {
	virtualHosts       map[string]types.VirtualHost
	defaultVirtualHost types.VirtualHost
}

// Routing with Virtual Host
func (rm *RouteMatcher) Route(headers map[string]string, randomValue uint64) (types.Route, string) {
	
	// First Step: Select VirtualHost with "host" in Headers form VirtualHost Array
	virtualHost := rm.FindVirtualHost(headers)
	
	// Second Step: Select Route from Routes in a Virtual Host
	if virtualHost == nil {
		log.DefaultLogger.Warnf("No VirtualHost Found when Routing %+v",headers)
		return nil,""
	}
	
	return virtualHost.GetRouteFromEntries(headers,randomValue),""
	
}

func (rm *RouteMatcher) AddRouter(routerName string) {}

func (rm *RouteMatcher) DelRouter(routerName string) {}


func (rm *RouteMatcher) FindVirtualHost(headers map[string]string) types.VirtualHost {
	if len(rm.virtualHosts) == 0 && rm.defaultVirtualHost != nil {
		return rm.defaultVirtualHost
	}
	
	host := strings.ToLower(headers["HostKey"])
	
	// for service, header["host"] == header["service"] == servicename
	// or use only a unique key for sofa's virtual host
	if virtualHost, ok := rm.virtualHosts[host]; ok {
		return virtualHost
	}
	
	// todo support WildcardVirtualHost
	
	return rm.defaultVirtualHost
}

type VirtualHostImpl struct {
	VirtualHostName string
	Routes          []RouteBase    //route impl
}

func (vh *VirtualHostImpl) Name() string {
	return vh.VirtualHostName
}

func (vh *VirtualHostImpl) CorsPolicy() types.CorsPolicy {
	return nil
	
}

func (vh *VirtualHostImpl) RateLimitPolicy() types.RateLimitPolicy {
	return nil
}

func (vh *VirtualHostImpl) GetRouteFromEntries(headers map[string]string, randomValue uint64) types.Route {
	
	// todo check tls
	// Check for a route that matches the request.
	
	//for _,route := range vh.Routes {
	//
	//	//if route.Match(headers,randomValue)
	//}
	
	return nil
	
}


// Base implementation for all route entries.
type RouteRuleImplBase struct {
	caseensitive bool
	prefixRewrite string
	hostRewrite string
	includeVirtualHostRateLimit bool
	corsPolicy types.CorsPolicy
	vHost *VirtualHostImpl
	autoHostRewrite bool
	useWebsocket bool
	clusterName string
	
	clusterNotFoundResponseCode http.HttpCode
	timeout time.Duration
	
	hostRedirect string
	pathRedirect string
	httpsRedirect bool
	retryPolicy *RetryPolicyImpl
	
	rateLimitPolicy *RateLimitPolicyImpl
	
	RouterAction v2.RouteAction
	
	shadowPolicy *ShadowPolicyImpl
	priority types.ResourcePriority
	
	configHeaders []LowerCaseString
	configQueryParametes []QueryParameterMatcher
	
	
	
}




// types.RouterInfo
func (rri *RouteRuleImplBase) GetRouterName()string {

	return ""
}

// types.Matchable
func (rri *RouteRuleImplBase) Match(headers map[string]string,randomValue uint64) (types.Route){
	
	return rri
	
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
func (rri *RouteRuleImplBase) ClusterName(clusterKey string) string {
	return rri.RouterAction.ClusterName
}

func (rri *RouteRuleImplBase) GlobalTimeout() time.Duration {
	return rri.RouterAction.Timeout
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

type PathRouteRuleImpl struct {
	RouteRuleImplBase
	path string
}

type PrefixRouteEntryImpl struct {
	RouteRuleImplBase
}

type RegexRouteEntryImpl struct {
	RouteRuleImplBase
}

// todo implement CorsPolicy

type  RuntimeData  struct {
	key string
	defaultvalue uint64
}


type RetryPolicyImpl struct {
	retryOn      bool
	retryTimeout time.Duration
	numRetries   int
}

func (p *RetryPolicyImpl) RetryOn() bool {
	return p.retryOn
}

func (p *RetryPolicyImpl) TryTimeout() time.Duration {
	return p.retryTimeout
}

func (p *RetryPolicyImpl) NumRetries() int {
	return p.numRetries
}

type RateLimitPolicyImpl struct {
	
	rateLimitEntries []types.RateLimitPolicyEntry
	maxStageNumber uint64
}

func (rp *RateLimitPolicyImpl) Enabled() bool {
	
	return true
}

func (rp *RateLimitPolicyImpl) GetApplicableRateLimit(stage string) []types.RateLimitPolicyEntry {
	
	return rp.rateLimitEntries
}