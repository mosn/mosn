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

package api

import (
	"context"
	"time"
)

// Route is a route instance
type Route interface {
	// RouteRule returns the route rule
	RouteRule() RouteRule

	// DirectResponseRule returns direct response rule
	DirectResponseRule() DirectResponseRule

	// RedirectRule returns redirect rule
	RedirectRule() RedirectRule
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

	HashPolicy() HashPolicy

	MirrorPolicy() MirrorPolicy
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

// DirectResponseRule contains direct response info
type DirectResponseRule interface {

	// StatusCode returns the repsonse status code
	StatusCode() int
	// Body returns the response body string
	Body() string
}

// RedirectRule contains redirect info
type RedirectRule interface {
	// RedirectCode returns the redirect repsonse status code
	RedirectCode() int
	// RedirectPath returns the path that will overwrite the current path
	RedirectPath() string
	// RedirectHost returns the host that will overwrite the current host
	RedirectHost() string
	// RedirectScheme returns the scheme that will overwrite the current scheme
	RedirectScheme() string
}

type MetadataMatchCriterion interface {
	// the name of the metadata key
	MetadataKeyName() string

	// the value for the metadata key
	MetadataValue() string
}

type MetadataMatchCriteria interface {
	// @return: a set of MetadataMatchCriterion(metadata) sorted lexically by name
	// to be matched against upstream endpoints when load balancing
	MetadataMatchCriteria() []MetadataMatchCriterion

	MergeMatchCriteria(metadataMatches map[string]interface{}) MetadataMatchCriteria
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
	Variable
)

type PathMatchCriterion interface {
	MatchType() PathMatchType
	Matcher() string
}

type HashPolicy interface {
	GenerateHash(context context.Context) uint64
}

type MirrorPolicy interface {
	ClusterName() string
	IsMirror() bool
}
