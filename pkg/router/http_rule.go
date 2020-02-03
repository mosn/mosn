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
	"regexp"
	"strings"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
)

type PathRouteRuleImpl struct {
	*RouteRuleImplBase
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
func (prri *PathRouteRuleImpl) FinalizeRequestHeaders(headers api.HeaderMap, requestInfo api.RequestInfo) {
	prri.finalizeRequestHeaders(headers, requestInfo)
	prri.finalizePathHeader(headers, prri.path)
}

func (prri *PathRouteRuleImpl) Match(headers api.HeaderMap, randomValue uint64) api.Route {
	if prri.matchRoute(headers, randomValue) {
		if headerPathValue, ok := headers.Get(protocol.MosnHeaderPathKey); ok {
			// TODO: config to support case sensitive
			// case insensitive
			if strings.EqualFold(headerPathValue, prri.path) {
				return prri
			}
		}
	}
	log.DefaultLogger.Debugf(RouterLogFormat, "path route rule", "failed match", headers)
	return nil
}

// PrefixRouteRuleImpl used to "match path" with "prefix match"
type PrefixRouteRuleImpl struct {
	*RouteRuleImplBase
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
func (prei *PrefixRouteRuleImpl) FinalizeRequestHeaders(headers api.HeaderMap, requestInfo api.RequestInfo) {
	prei.finalizeRequestHeaders(headers, requestInfo)
	prei.finalizePathHeader(headers, prei.prefix)
}

func (prei *PrefixRouteRuleImpl) Match(headers api.HeaderMap, randomValue uint64) api.Route {
	if prei.matchRoute(headers, randomValue) {
		if headerPathValue, ok := headers.Get(protocol.MosnHeaderPathKey); ok {
			if strings.HasPrefix(headerPathValue, prei.prefix) {
				return prei
			}
		}
	}
	log.DefaultLogger.Debugf(RouterLogFormat, "prefxi route rule", "failed match", headers)
	return nil
}

// RegexRouteRuleImpl used to "match path" with "regex match"
type RegexRouteRuleImpl struct {
	*RouteRuleImplBase
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

func (rrei *RegexRouteRuleImpl) FinalizeRequestHeaders(headers api.HeaderMap, requestInfo api.RequestInfo) {
	rrei.finalizeRequestHeaders(headers, requestInfo)
	rrei.finalizePathHeader(headers, rrei.regexStr)
}

func (rrei *RegexRouteRuleImpl) Match(headers api.HeaderMap, randomValue uint64) api.Route {
	if rrei.matchRoute(headers, randomValue) {
		if headerPathValue, ok := headers.Get(protocol.MosnHeaderPathKey); ok {
			if rrei.regexPattern.MatchString(headerPathValue) {
				return rrei
			}
		}
	}
	log.DefaultLogger.Debugf(RouterLogFormat, "regex route rule", "failed match", headers)
	return nil
}
