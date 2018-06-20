package types

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/flowcontrol/ratelimit"
	"time"
)

type Priority int

const (
	PriorityDefault Priority      = 0
	PriorityHigh    Priority      = 1
	GlobalTimeout   time.Duration = 60 * time.Second
	DefaultRouteTimeout          = 15 * time.Second
)

// change RouterConfig -> Routers to manage all routers
type Routers interface {
	// routing with headers
	Route(headers map[string]string, randomValue uint64) (Route, string)
	// add router to Routers
	AddRouter(routerName string)
	
	// del router from Routers
	DelRouter(routerName string)
}

// used to manage all routerConfigs
type RouterConfigManager interface {
	// add routerConfig when generated
	AddRoutersSet(routerConfig Routers)
	// remove Routers in routerConfig
	RemoveRouterInRouters(routerNames []string)
	// addRouters in routerConfig
	AddRouterInRouters(routerNames []string)
}

type Route interface {
	RedirectRule() RedirectRule
	
	RouteRule() RouteRule
	
	TraceDecorator() TraceDecorator
}

type RouteRule interface {
	ClusterName(clusterKey string) string
	
	GlobalTimeout() time.Duration
	
	Priority() Priority
	
	VirtualHost() VirtualHost
	
	VirtualCluster(headers map[string]string) VirtualCluster
	
	Policy() Policy
	
	MetadataMatcher() MetadataMatcher
}

type Policy interface {
	RetryPolicy() RetryPolicy
	
	ShadowPolicy() ShadowPolicy
	
	CorsPolicy() CorsPolicy
	
	LoadBalancerPolicy() LoadBalancerPolicy
}

type TargetCluster interface {
	Name() string
	
	NotFoundResponse() interface{}
}

type CorsPolicy interface {
	AllowOrigins() []string
	
	AllowMethods() string
	
	AllowHeaders() string
	
	ExposeHeaders() string
	
	MaxAga() string
	
	AllowCredentials() bool
	
	Enabled() bool
}

type LoadBalancerPolicy interface {
	HashPolicy() HashPolicy
}

type AddCookieCallback func(key string, ttl int)

type HashPolicy interface {
	GenerateHash(downstreamAddress string, headers map[string]string, addCookieCb AddCookieCallback)
}

type RateLimitPolicy interface {
	Enabled() bool
	
	GetApplicableRateLimit(stage string) []RateLimitPolicyEntry
}

type RateLimitPolicyEntry interface {
	Stage() uint64
	
	DisableKey() string
	
	PopulateDescriptors(route RouteRule, descriptors []ratelimit.Descriptor, localSrvCluster string, headers map[string]string, remoteAddr string)
}

type RetryPolicy interface {
	RetryOn() bool
	
	TryTimeout() time.Duration
	
	NumRetries() uint32
}

type DoRetryCallback func()

type RetryState interface {
	Enabled() bool
	
	ShouldRetry(respHeaders map[string]string, resetReson string, doRetryCb DoRetryCallback) bool
}

type ShadowPolicy interface {
	ClusterName() string
	
	RuntimeKey() string
}

type VirtualServer interface {
	VirtualCluster() VirtualCluster
	
	VirtualHost() VirtualHost
}

type VirtualCluster interface {
	Name() string
}

type VirtualHost interface {
	Name() string
	
	CorsPolicy() CorsPolicy
	
	RateLimitPolicy() RateLimitPolicy
	
	GetRouteFromEntries(headers map[string]string, randomValue uint64) Route
}

type MetadataMatcher interface {
	Metadata() v2.Metadata
	
	MetadataMatchEntrySet() MetadataMatchEntrySet
}

type MetadataMatchEntrySet []MetadataMatchEntry

type MetadataMatchEntry interface {
	Key() string
	
	Value() string
}

type RedirectRule interface {
	NewPath(headers map[string]string) string
	
	ResponseCode() interface{}
	
	ResponseBody() string
}

type TraceDecorator interface {
	operate(span Span)
	
	getOperation() string
}