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

package router

import (
	"context"
	"regexp"
	"strings"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

type BaseHTTPRouteRule struct {
	*RouteRuleImplBase
	configHeaders         types.HeaderMatcher
	configQueryParameters types.QueryParameterMatcher //TODO: not implement yet
}

func NewBaseHTTPRouteRule(base *RouteRuleImplBase, headers []v2.HeaderMatcher) *BaseHTTPRouteRule {
	return &BaseHTTPRouteRule{
		RouteRuleImplBase: base,
		configHeaders:     CreateHTTPHeaderMatcher(headers),
	}
}

func (rri *BaseHTTPRouteRule) HeaderMatchCriteria() api.KeyValueMatchCriteria {
	if rri.configHeaders != nil {
		return rri.configHeaders.HeaderMatchCriteria()
	}
	return nil
}

func (rri *BaseHTTPRouteRule) matchRoute(ctx context.Context, headers api.HeaderMap) bool {
	// 1. match headers' KV
	if !rri.configHeaders.Matches(ctx, headers) {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf(RouterLogFormat, "routerule", "match header", headers)
		}
		return false
	}
	// 2. match query parameters
	if rri.configQueryParameters != nil {
		var queryParams types.QueryParams
		QueryString, err := variable.GetString(ctx, types.VarQueryString)
		if err == nil && QueryString != "" {
			queryParams = http.ParseQueryString(QueryString)
		}
		if len(queryParams) != 0 {
			if !rri.configQueryParameters.Matches(ctx, queryParams) {
				if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
					log.DefaultLogger.Debugf(RouterLogFormat, "routerule", "match query params", queryParams)
				}
				return false
			}
		}
	}
	return true
}

type PathRouteRuleImpl struct {
	*BaseHTTPRouteRule
	path string
}

func (prri *PathRouteRuleImpl) PathMatchCriterion() api.PathMatchCriterion {
	return prri
}

func (prri *PathRouteRuleImpl) RouteRule() api.RouteRule {
	return prri
}

// types.PathMatchCriterion
func (prri *PathRouteRuleImpl) Matcher() string {
	return prri.path
}

func (prri *PathRouteRuleImpl) MatchType() api.PathMatchType {
	return api.Exact
}

// types.RouteRule
// override Base
func (prri *PathRouteRuleImpl) FinalizeRequestHeaders(ctx context.Context, headers api.HeaderMap, requestInfo api.RequestInfo) {
	prri.finalizeRequestHeaders(ctx, headers, requestInfo)
	prri.finalizePathHeader(ctx, headers, prri.path)
}

func (prri *PathRouteRuleImpl) Match(ctx context.Context, headers api.HeaderMap) api.Route {
	if prri.matchRoute(ctx, headers) {
		headerPathValue, err := variable.GetString(ctx, types.VarPath)
		if err == nil && headerPathValue != "" {
			// TODO: config to support case sensitive
			// case insensitive
			if strings.EqualFold(headerPathValue, prri.path) {
				return prri
			}
		}
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "path route rule", "failed match", headers)
	}
	return nil
}

// PrefixRouteRuleImpl used to "match path" with "prefix match"
type PrefixRouteRuleImpl struct {
	*BaseHTTPRouteRule
	prefix string
}

func (prei *PrefixRouteRuleImpl) PathMatchCriterion() api.PathMatchCriterion {
	return prei
}

func (prei *PrefixRouteRuleImpl) RouteRule() api.RouteRule {
	return prei
}

// types.PathMatchCriterion
func (prei *PrefixRouteRuleImpl) Matcher() string {
	return prei.prefix
}

func (prei *PrefixRouteRuleImpl) MatchType() api.PathMatchType {
	return api.Prefix
}

// types.RouteRule
// override Base
func (prei *PrefixRouteRuleImpl) FinalizeRequestHeaders(ctx context.Context, headers api.HeaderMap, requestInfo api.RequestInfo) {
	prei.finalizeRequestHeaders(ctx, headers, requestInfo)
	prei.finalizePathHeader(ctx, headers, prei.prefix)
}

func (prei *PrefixRouteRuleImpl) Match(ctx context.Context, headers api.HeaderMap) api.Route {
	if prei.matchRoute(ctx, headers) {
		headerPathValue, err := variable.GetString(ctx, types.VarPath)
		if err == nil && headerPathValue != "" {
			if strings.HasPrefix(headerPathValue, prei.prefix) {
				return prei
			}
		}
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "prefxi route rule", "failed match", headers)
	}
	return nil
}

// RegexRouteRuleImpl used to "match path" with "regex match"
type RegexRouteRuleImpl struct {
	*BaseHTTPRouteRule
	regexStr     string
	regexPattern *regexp.Regexp
}

func (rrei *RegexRouteRuleImpl) PathMatchCriterion() api.PathMatchCriterion {
	return rrei
}

func (rrei *RegexRouteRuleImpl) RouteRule() api.RouteRule {
	return rrei
}

func (rrei *RegexRouteRuleImpl) Matcher() string {
	return rrei.regexStr
}

func (rrei *RegexRouteRuleImpl) MatchType() api.PathMatchType {
	return api.Regex
}

func (rrei *RegexRouteRuleImpl) FinalizeRequestHeaders(ctx context.Context, headers api.HeaderMap, requestInfo api.RequestInfo) {
	rrei.finalizeRequestHeaders(ctx, headers, requestInfo)
	rrei.finalizePathHeader(ctx, headers, rrei.regexStr)
}

func (rrei *RegexRouteRuleImpl) Match(ctx context.Context, headers api.HeaderMap) api.Route {
	if rrei.matchRoute(ctx, headers) {
		headerPathValue, err := variable.GetString(ctx, types.VarPath)
		if err == nil && headerPathValue != "" {
			if rrei.regexPattern.MatchString(headerPathValue) {
				return rrei
			}
		}
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "regex route rule", "failed match", headers)
	}
	return nil
}
