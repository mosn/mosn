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
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

// Default parameters for route

type RouterType string

const (
	GlobalTimeout       = 60 * time.Second
	DefaultRouteTimeout = 15 * time.Second
	RPCRouteMatchKey    = "service"
	RouterMetadataKey   = "filter_metadata"
	RouterMetadataKeyLb = "mosn.lb"
)

// Routers defines and manages all router
type Routers interface {
	// MatchRoute return first route with headers
	MatchRoute(ctx context.Context, headers api.HeaderMap) api.Route
	// MatchAllRoutes returns all routes with headers
	MatchAllRoutes(ctx context.Context, headers api.HeaderMap) []api.Route
	// MatchRouteFromHeaderKV is used to quickly locate and obtain routes in certain scenarios
	// header is used to find virtual host
	MatchRouteFromHeaderKV(ctx context.Context, headers api.HeaderMap, key, value string) api.Route
	// AddRoute adds a route into virtual host, find virtual host by domain
	// returns the virtualhost index, -1 means no virtual host found
	AddRoute(domain string, route *v2.Router) int
	// RemoveAllRoutes will clear all the routes in the virtual host, find virtual host by domain
	RemoveAllRoutes(domain string) int
}

// RouterManager is a manager for all routers' config
type RouterManager interface {
	// AddOrUpdateRouters used to add or update router
	AddOrUpdateRouters(routerConfig *v2.RouterConfiguration) error
	// GetRouterWrapperByName returns a router wrapper from manager
	GetRouterWrapperByName(routerConfigName string) RouterWrapper
	// AddRoute adds a single router rule into specified virtualhost(by domain)
	AddRoute(routerConfigName, domain string, route *v2.Router) error
	// RemoveAllRoutes clear all the specified virtualhost's routes
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

const DefaultRouteHandler = "default"

// RouteHandler is an external check handler for a route
type RouteHandler interface {
	// IsAvailable returns HandlerStatus represents the handler will be used/not used/stop next handler check
	IsAvailable(context.Context, ClusterManager) (ClusterSnapshot, HandlerStatus)
	// Route returns handler's route
	Route() api.Route
}
type RouterWrapper interface {
	// GetRouters returns the routers in the wrapper
	GetRouters() Routers
	// GetRoutersConfig returns the routers config in the wrapper
	GetRoutersConfig() v2.RouterConfiguration
}

type HeaderFormat interface {
	Format(info api.RequestInfo) string
	Append() bool
}

// QueryParams is a string-string map
type QueryParams map[string]string

// QueryParameterMatcher match request's query parameter
type QueryParameterMatcher interface {
	// Matches check whether the query parameters specified in the config are present in a request.
	// If all the query params (and values) in the query parameter matcher are found in the query_params, return true.
	Matches(ctx context.Context, requestQueryParams QueryParams) bool
}

// HeaderMatcher match request's headers
type HeaderMatcher interface {
	// HeaderMatchCriteria returns the route's HeaderMatchCriteria
	HeaderMatchCriteria() api.KeyValueMatchCriteria

	// Matches  check whether the headers specified in the config are present in a request.
	// If all the headers (and values) in the header matcher  are found in the request_headers, return true.
	Matches(ctx context.Context, requestHeaders api.HeaderMap) bool
}
