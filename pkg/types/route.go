package types

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/flowcontrol/ratelimit"
	"container/list"
	"regexp"
	"time"
)

type Priority int

const (
	PriorityDefault     Priority      = 0
	PriorityHigh        Priority      = 1
	GlobalTimeout       time.Duration = 60 * time.Second
	DefaultRouteTimeout               = 15 * time.Second
	SofaRouteMatchKey                 = "service"
)

// change RouterConfig -> Routers to manage all routers
type Routers interface {
	// routing with headers
	Route(headers map[string]string, randomValue uint64) Route
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
	ClusterName() string

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
	VirtualClusterName() string
}

type VirtualHost interface {
	Name() string

	CorsPolicy() CorsPolicy

	RateLimitPolicy() RateLimitPolicy

	// Get Matched Route
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

// the name of the metadata key
type MetadataMatchCriterion interface {
	MetadataKeyName() string
	Value() HashedValue
}

type MetadataMatchCriteria interface {
	MetadataMatchCriteria() []*MetadataMatchCriterion
}

type Decorator interface {
	apply(span Span)
	getOperation() string
}

// todo add HashedValue
type HashedValue struct{}

type HeaderFormat interface {
	Format(info RequestInfo) string
	Append() bool
}

type PathMatchType uint32

const (
	None PathMatchType = iota
	Prefix
	Exact
	Regex
	SofaHeader
)

type SslRequirements uint32

const (
	NONE SslRequirements = iota
	EXTERNALONLY
	ALL
)

/**
 * The router configuration.
 */
type Config interface{
	Route(headers map[string]string, randomValue uint64) (Route,string)
	InternalOnlyHeaders() *list.List
	Name() string
}

type QueryParams map[string]string


// match request's query parameter
type QueryParameterMatcher interface {
	// bool true if a match for this QueryParameterMatcher exists in request_query_params.
	Matches(requestQueryParams QueryParams) bool
}

// An empty header value allows for matching to be only based on header presence.
// Regex is an opt-in. Unless explicitly mentioned, the header values will be used for
// exact string matching.

type HeaderData struct {
	Name         LowerCaseString
	Value        string
	IsRegex      bool
	RegexPattern regexp.Regexp
}

// Utility routines for loading route configuration and matching runtime request headers.
type ConfigUtility interface {
	// See if the headers specified in the config are present in a request.
	// bool true if all the headers (and values) in the config_headers are found in the request_headers
	MatchHeaders (requestHeaders map[string]string, configHeaders[]*HeaderData) bool
	
	// See if the query parameters specified in the config are present in a request.
	// bool true if all the query params (and values) in the config_params are found in the query_params
	MatchQueryParams (queryParams *QueryParams, configQueryParams []QueryParameterMatcher) bool
	
}

type LowerCaseString interface {
	Lower()
	Equal(rhs LowerCaseString) bool
	Get() string
}

type PathMatchCriterion interface {
	MatchType() PathMatchType
	Matcher()string
}
