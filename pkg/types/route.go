package types

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/flowcontrol/ratelimit"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"time"
)

type Priority int

const (
	PriorityDefault Priority = 0
	PriorityHigh Priority = 1
)

type RouterConfig interface {
	Route(headers map[string]string) (Route,string)
	//GetRouteNameByCluster(clusterName string)string
}

type Route interface {
	RedirectRule() RedirectRule

	RouteRule() RouteRule

	TraceDecorator() TraceDecorator

}

type RouteRule interface {
	ClusterName(serviceName string) string

	GlobalTimeout() time.Duration

	Priority() Priority

	VirtualCluster(headers map[string]string) VirtualCluster

	Policy() Policy

	MetadataMatcher() MetadataMatcher

	//UpdateServiceName(serviceName string)
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

	NumRetries() int
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
