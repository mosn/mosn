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
	"context"
	"regexp"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
)

// Default parameters for route

type RouterType string

const (
	GlobalTimeout                  = 60 * time.Second
	DefaultRouteTimeout            = 15 * time.Second
	SofaRouteMatchKey              = "service"
	RouterMetadataKey              = "filter_metadata"
	RouterMetadataKeyLb            = "mosn.lb"
	SofaRouterType      RouterType = "sofa"
)

// Routers defines and manages all router
type Routers interface {
	// MatchRoute return first route with headers
	MatchRoute(headers HeaderMap, randomValue uint64) Route
	// MatchAllRoutes returns all routes with headers
	MatchAllRoutes(headers HeaderMap, randomValue uint64) []Route
	// MatchRouteFromHeaderKV is used to quickly locate and obtain routes in certain scenarios
	// header is used to find virtual host
	MatchRouteFromHeaderKV(headers HeaderMap, key, value string) Route
	// AddRoute adds a route into virtual host, find virtual host by domain
	// returns the virtualhost index, -1 means no virtual host found
	AddRoute(domain string, route *v2.Router) int
	// RemoveAllRoutes will clear all the routes in the virtual host, find virtual host by domain
	RemoveAllRoutes(domain string) int
}

// RouterManager is a manager for all routers' config
type RouterManager interface {
	// AddRoutersSet adds router config when generated
	AddOrUpdateRouters(routerConfig *v2.RouterConfiguration) error

	GetRouterWrapperByName(routerConfigName string) RouterWrapper

	AddRoute(routerConfigName, domain string, route *v2.Router) error

	RemoveAllRoutes(routerConfigName, domain string) error
}

// HandlerStatus returns the Handler's available status
type HandlerStatus int

// HandlerStatus enum
const (
	HandlerAvailable HandlerStatus = iota
	HandlerNotAvailable
	HandlerStop
)

// RouteHandler is an external check handler for a route
type RouteHandler interface {
	// IsAvailable returns HandlerStatus represents the handler will be used/not used/stop next handler check
	IsAvailable(context.Context, ClusterManager) (ClusterSnapshot, HandlerStatus)
	// Route returns handler's route
	Route() Route
}
type RouterWrapper interface {
	// GetRouters returns the routers in the wrapper
	GetRouters() Routers
	// GetRoutersConfig returns the routers config in the wrapper
	GetRoutersConfig() v2.RouterConfiguration
}

// Route is a route instance
type Route interface {
	// RouteRule returns the route rule
	RouteRule() RouteRule

	// DirectResponseRule returns direct response rile
	DirectResponseRule() DirectResponseRule
}

// RouteRule defines parameters for a route
type RouteRule interface {
	// ClusterName returns the route's cluster name
	ClusterName() string

	// UpstreamProtocol returns the protocol that route's cluster supported
	// If it is configured, the protocol will replace the proxy config's upstream protocol
	UpstreamProtocol() string

	// GlobalTimeout returns the global timeout
	GlobalTimeout() time.Duration

	// VirtualHost returns the route's virtual host
	VirtualHost() VirtualHost

	// Policy returns the route's route policy
	Policy() Policy

	// MetadataMatchCriteria returns the metadata that a subset load balancer should match when selecting an upstream host
	// as we may use weighted cluster's metadata, so need to input cluster's name
	MetadataMatchCriteria(clusterName string) MetadataMatchCriteria

	// PerFilterConfig returns per filter config from xds
	PerFilterConfig() map[string]interface{}

	// FinalizeRequestHeaders do potentially destructive header transforms on request headers prior to forwarding
	FinalizeRequestHeaders(headers HeaderMap, requestInfo RequestInfo)

	// FinalizeResponseHeaders do potentially destructive header transforms on response headers prior to forwarding
	FinalizeResponseHeaders(headers HeaderMap, requestInfo RequestInfo)

	// PathMatchCriterion returns the route's PathMatchCriterion
	PathMatchCriterion() PathMatchCriterion
}

// Policy defines a group of route policy
type Policy interface {
	RetryPolicy() RetryPolicy

	ShadowPolicy() ShadowPolicy
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

type VirtualHost interface {
	Name() string

	// GetRouteFromEntries returns a Route matched the condition
	GetRouteFromEntries(headers HeaderMap, randomValue uint64) Route
	// GetAllRoutesFromEntries returns all Route matched the condition
	GetAllRoutesFromEntries(headers HeaderMap, randomValue uint64) []Route
	// GetRouteFromHeaderKV is used to quickly locate and obtain routes in certain scenarios
	GetRouteFromHeaderKV(key, value string) Route
	// AddRoute adds a new route into virtual host
	AddRoute(route *v2.Router) error
	// RemoveAllRoutes clear all the routes in the virtual host
	RemoveAllRoutes()
}

// DirectResponseRule contains direct response info
type DirectResponseRule interface {

	// StatusCode returns the repsonse status code
	StatusCode() int
	// Body returns the response body string
	Body() string
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
	RegexPattern *regexp.Regexp
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

//EqualHashValue comapres two HashedValues are equaled or not
func EqualHashValue(h1 HashedValue, h2 HashedValue) bool {
	return h1 == h2
}
