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

package types

import (
	"container/list"
	"crypto/md5"
	"regexp"
	"time"
)

// Priority type
type Priority int

// Priority types
const (
	PriorityDefault Priority = 0
	PriorityHigh    Priority = 1
)

// Default parameters for route
const (
	GlobalTimeout       = 60 * time.Second
	DefaultRouteTimeout = 15 * time.Second
	SofaRouteMatchKey   = "service"
	RouterMetadataKey   = "filter_metadata"
	RouterMetadataKeyLb = "mosn.lb"
)

// Routers defines and manages all router
type Routers interface {
	// Route is used to route with headers
	Route(headers map[string]string, randomValue uint64) Route
	// AddRouter adds router to Routers
	AddRouter(routerName string)
	// DelRouter deletes router from Routers
	DelRouter(routerName string)
}

// RouterConfigManager is a manager for all routers' config
type RouterConfigManager interface {
	// AddRoutersSet adds router config when generated
	AddRoutersSet(routerConfig Routers)
	// RemoveRouterInRouters removes routers
	RemoveRouterInRouters(routerNames []string)
	// AddRouterInRouters adds routers
	AddRouterInRouters(routerNames []string)
}

// Route is a route instance
type Route interface {
	// RedirectRule returns the redirect rule
	RedirectRule() RedirectRule

	// RouteRule returns the route rule
	RouteRule() RouteRule

	// TraceDecorator returns the trace decorator
	TraceDecorator() TraceDecorator
}

// RouteRule defines parameters for a route
type RouteRule interface {
	// ClusterName returns the route's cluster name
	ClusterName() string

	// GlobalTimeout returns the global timeout
	GlobalTimeout() time.Duration

	// Priority returns the route's priority
	Priority() Priority

	// VirtualHost returns the route's virtual host
	VirtualHost() VirtualHost

	// VirtualCluster returns the route's virtual cluster
	VirtualCluster(headers map[string]string) VirtualCluster

	// Policy returns the route's route policy
	Policy() Policy

	//MetadataMatcher() MetadataMatcher

	// Metadata returns the route's route meta data
	Metadata() RouteMetaData

	// MetadataMatchCriteria returns the metadata that a subset load balancer should match when selecting an upstream host
	// as we may use weighted cluster's metadata, so need to input cluster's name
	MetadataMatchCriteria(clusterName string) MetadataMatchCriteria
}

// Policy defines a group of route policy
type Policy interface {
	RetryPolicy() RetryPolicy

	ShadowPolicy() ShadowPolicy

	CorsPolicy() CorsPolicy

	LoadBalancerPolicy() LoadBalancerPolicy
}

// CorsPolicy is a type of Policy
type CorsPolicy interface {
	AllowOrigins() []string

	AllowMethods() string

	AllowHeaders() string

	ExposeHeaders() string

	MaxAga() string

	AllowCredentials() bool

	Enabled() bool
}

// LoadBalancerPolicy is a type of Policy
type LoadBalancerPolicy interface {
	HashPolicy() HashPolicy
}

type AddCookieCallback func(key string, ttl int)

// HashPolicy is a type of Policy
type HashPolicy interface {
	GenerateHash(downstreamAddress string, headers map[string]string, addCookieCb AddCookieCallback)
}

// RateLimitPolicy is a type of Policy
type RateLimitPolicy interface {
	Enabled() bool

	GetApplicableRateLimit(stage string) []RateLimitPolicyEntry
}

type RateLimitPolicyEntry interface {
	Stage() uint64

	DisableKey() string

	PopulateDescriptors(route RouteRule, descriptors []Descriptor, localSrvCluster string, headers map[string]string, remoteAddr string)
}

// LimitStatus type
type LimitStatus string

// LimitStatus types
const (
	OK        LimitStatus = "OK"
	Error     LimitStatus = "Error"
	OverLimit LimitStatus = "OverLimit"
)

// DescriptorEntry is a key-value pair
type DescriptorEntry struct {
	Key   string
	Value string
}

// Descriptor contains a list pf DescriptorEntry
type Descriptor struct {
	entries []DescriptorEntry
}

// RetryCheckStatus type
type RetryCheckStatus int

// RetryCheckStatus types
const (
	ShouldRetry   RetryCheckStatus = 0
	NoRetry       RetryCheckStatus = -1
	RetryOverflow RetryCheckStatus = -2
)

// RetryPolicy is a type of Policy
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

// ShadowPolicy is a type of Policy
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

	// GetRouteFromEntries returns a Route matched the condition
	GetRouteFromEntries(headers map[string]string, randomValue uint64) Route
}

type MetadataMatcher interface {
	Metadata() RouteMetaData

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

type MetadataMatchCriterion interface {
	// the name of the metadata key
	MetadataKeyName() string

	// the value for the metadata key
	MetadataValue() HashedValue
}

type MetadataMatchCriteria interface {
	// @return: a set of MetadataMatchCriterion(metadata) sorted lexically by name
	// to be matched against upstream endpoints when load balancing
	MetadataMatchCriteria() []MetadataMatchCriterion

	MergeMatchCriteria(metadataMatches map[string]interface{}) MetadataMatchCriteria
}

type Decorator interface {
	apply(span Span)
	getOperation() string
}

// HashedValue is a value as md5's result
// TODO: change hashed value to [16]string
// currently use string for easily debug
type HashedValue string

type HeaderFormat interface {
	Format(info RequestInfo) string
	Append() bool
}

// PathMatchType defines the match pattern
type PathMatchType uint32

// Path match patterns
const (
	None PathMatchType = iota
	Prefix
	Exact
	Regex
	SofaHeader
)

// SslRequirements type
type SslRequirements uint32

// SslRequirements types
const (
	NONE SslRequirements = iota
	EXTERNALONLY
	ALL
)

// Config defines the router configuration
type Config interface {
	Route(headers map[string]string, randomValue uint64) (Route, string)
	InternalOnlyHeaders() *list.List
	Name() string
}

// QueryParams is a string-string map
type QueryParams map[string]string

// QueryParameterMatcher match request's query parameter
type QueryParameterMatcher interface {
	// Matches returns true if a match for this QueryParameterMatcher exists in request_query_params.
	Matches(requestQueryParams QueryParams) bool
}

// HeaderData defines headers data.
// An empty header value allows for matching to be only based on header presence.
// Regex is an opt-in. Unless explicitly mentioned, the header values will be used for
// exact string matching.
type HeaderData struct {
	Name         LowerCaseString
	Value        string
	IsRegex      bool
	RegexPattern regexp.Regexp
}

// ConfigUtility is utility routines for loading route configuration and matching runtime request headers.
type ConfigUtility interface {
	// MatchHeaders check whether the headers specified in the config are present in a request.
	// If all the headers (and values) in the config_headers are found in the request_headers, return true.
	MatchHeaders(requestHeaders map[string]string, configHeaders []*HeaderData) bool

	// MatchQueryParams check whether the query parameters specified in the config are present in a request.
	// If all the query params (and values) in the config_params are found in the query_params, return true.
	MatchQueryParams(queryParams QueryParams, configQueryParams []QueryParameterMatcher) bool
}

// LowerCaseString is a string wrapper
type LowerCaseString interface {
	Lower()
	Equal(rhs LowerCaseString) bool
	Get() string
}

type PathMatchCriterion interface {
	MatchType() PathMatchType
	Matcher() string
}

type Loader struct{}

type RouteMetaData map[string]HashedValue

// GenerateHashedValue generates generates hashed valued with md5
func GenerateHashedValue(input string) HashedValue {
	data := []byte(input)
	h := md5.Sum(data)
	_ = h
	// return h
	// todo use hashed value as md5
	return HashedValue(input)
}

//EqualHashValue comapres two HashedValues are equaled or not
func EqualHashValue(h1 HashedValue, h2 HashedValue) bool {
	return h1 == h2
}
