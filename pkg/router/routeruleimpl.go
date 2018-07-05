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
	"time"

	multimap "github.com/jwangsadinata/go-multimap/slicemultimap"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	//"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	httpmosn "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/http"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
)

func NewRouteRuleImplBase(vHost *VirtualHostImpl, route *v2.Router) RouteRuleImplBase {
	routeRuleImplBase := RouteRuleImplBase{
		vHost:        vHost,
		routerMatch:  route.Match,
		routerAction: route.Route,
		policy: &routerPolicy{
			retryOn:      false,
			retryTimeout: 0,
			numRetries:   0,
		},
	}

	// generate metadata match criteria from router's metadata
	if len(route.Route.MetadataMatch) > 0 {
		envoyLBMetaData := GetMosnLBMetaData(route)
		routeRuleImplBase.metadataMatchCriteria = NewMetadataMatchCriteriaImpl(envoyLBMetaData)

		routeRuleImplBase.metaData = GetClusterMosnLBMetaDataMap(route.Route.MetadataMatch)
	}

	return routeRuleImplBase
}

// Base implementation for all route entries.
type RouteRuleImplBase struct {
	caseSensitive               bool
	prefixRewrite               string
	hostRewrite                 string
	includeVirtualHostRateLimit bool
	corsPolicy                  types.CorsPolicy //todo
	vHost                       *VirtualHostImpl

	autoHostRewrite             bool
	useWebSocket                bool
	clusterName                 string //
	clusterHeaderName           LowerCaseString
	clusterNotFoundResponseCode httpmosn.HttpCode
	timeout                     time.Duration
	runtime                     v2.RuntimeUInt32
	hostRedirect                string
	pathRedirect                string
	httpsRedirect               bool
	retryPolicy                 *RetryPolicyImpl
	rateLimitPolicy             *RateLimitPolicyImpl

	routerAction v2.RouteAction
	routerMatch  v2.RouterMatch

	shadowPolicy          *ShadowPolicyImpl
	priority              types.ResourcePriority
	configHeaders         []*types.HeaderData //
	configQueryParameters []types.QueryParameterMatcher
	weightedClusters      []*WeightedClusterEntry
	totalClusterWeight    uint64
	hashPolicy            HashPolicyImpl

	metadataMatchCriteria *MetadataMatchCriteriaImpl
	metaData              types.RouteMetaData

	requestHeadersParser  *HeaderParser
	responseHeadersParser *HeaderParser

	opaqueConfig multimap.MultiMap

	decorator          *types.Decorator
	directResponseCode httpmosn.HttpCode
	directResponseBody string
	policy             *routerPolicy
	virtualClusters    *VirtualClusterEntry
}

// types.RouterInfo
func (rri *RouteRuleImplBase) GetRouterName() string {

	return ""
}

// types.Route
func (rri *RouteRuleImplBase) RedirectRule() types.RedirectRule {

	return rri.RedirectRule()
}

func (rri *RouteRuleImplBase) RouteRule() types.RouteRule {

	return rri
}

func (rri *RouteRuleImplBase) TraceDecorator() types.TraceDecorator {

	return nil
}

// types.RouteRule
// Select Cluster for Routing
// todo support weighted cluster
func (rri *RouteRuleImplBase) ClusterName() string {

	return rri.routerAction.ClusterName
}

func (rri *RouteRuleImplBase) GlobalTimeout() time.Duration {

	return rri.routerAction.Timeout
}

func (rri *RouteRuleImplBase) Priority() types.Priority {

	return 0
}

func (rri *RouteRuleImplBase) VirtualHost() types.VirtualHost {

	return rri.vHost
}

func (rri *RouteRuleImplBase) VirtualCluster(headers map[string]string) types.VirtualCluster {

	return rri.virtualClusters
}

func (rri *RouteRuleImplBase) Policy() types.Policy {

	return rri.policy
}

func (rri *RouteRuleImplBase) Metadata() types.RouteMetaData {
	return rri.metaData
}

func (rri *RouteRuleImplBase) MetadataMatchCriteria() types.MetadataMatchCriteria {
	return rri.metadataMatchCriteria
}

// todo
func (rri *RouteRuleImplBase) finalizePathHeader(headers map[string]string, matchedPath string) {

}

func (rri *RouteRuleImplBase) matchRoute(headers map[string]string, randomValue uint64) bool {
	// todo check runtime
	// 1. match headers' KV
	if !ConfigUtilityInst.MatchHeaders(headers, rri.configHeaders) {
		return false
	}

	// 2. match query parameters
	var queryParams types.QueryParams

	if QueryString, ok := headers[types.HeaderQueryString]; ok {
		queryParams = httpmosn.ParseQueryString(QueryString)
	}

	if len(queryParams) == 0 {
		return true
	} else {
		return ConfigUtilityInst.MatchQueryParams(&queryParams, rri.configQueryParameters)
	}

	return true
}

type SofaRouteRuleImpl struct {
	RouteRuleImplBase
	matchValue string
}

func (srri *SofaRouteRuleImpl) Matcher() string {

	return srri.matchValue
}

func (srri *SofaRouteRuleImpl) MatchType() types.PathMatchType {

	return types.SofaHeader
}

func (srri *SofaRouteRuleImpl) Match(headers map[string]string, randomValue uint64) types.Route {
	if value, ok := headers[types.SofaRouteMatchKey]; ok {
		if value == srri.matchValue || srri.matchValue == ".*" {
			log.DefaultLogger.Debugf("Sofa Router Matched")
			return srri
		} else {
			log.DefaultLogger.Warnf("No Sofa Router Matched, Service Value in Header = %s", value)
		}
	} else {
		log.DefaultLogger.Warnf("No service key found in header, sofa router matcher error")
	}

	return nil
}

type PathRouteRuleImpl struct {
	RouteRuleImplBase
	path string
}

func (prri *PathRouteRuleImpl) Matcher() string {

	return prri.path
}

func (prri *PathRouteRuleImpl) MatchType() types.PathMatchType {

	return types.Exact
}

// Exact Path Comparing
func (prri *PathRouteRuleImpl) Match(headers map[string]string, randomValue uint64) types.Route {
	// match base rule first
	log.StartLogger.Tracef("path route rule match invoked")
	if prri.matchRoute(headers, randomValue) {

		if headerPathValue, ok := headers[strings.ToLower(protocol.MosnHeaderPathKey)]; ok {

			if prri.caseSensitive {
				if headerPathValue == prri.path {
					return prri
				}
			} else if strings.EqualFold(headerPathValue, prri.path) {
				log.StartLogger.Tracef("path route rule match success")
				return prri
			}
		}
	}
	log.DefaultLogger.Warnf("path route rule match failed")
	
	return nil
}

// todo
func (prri *PathRouteRuleImpl) FinalizeRequestHeaders(headers map[string]string, requestInfo types.RequestInfo) {
	prri.finalizePathHeader(headers, prri.path)
}

type PrefixRouteRuleImpl struct {
	RouteRuleImplBase
	prefix string
}

func (prei *PrefixRouteRuleImpl) Matcher() string {

	return prei.prefix
}

func (prei *PrefixRouteRuleImpl) MatchType() types.PathMatchType {

	return types.Prefix
}

// Compare Path's Prefix
func (prei *PrefixRouteRuleImpl) Match(headers map[string]string, randomValue uint64) types.Route {

	if prei.matchRoute(headers, randomValue) {

		if headerPathValue, ok := headers[strings.ToLower(protocol.MosnHeaderPathKey)]; ok {

			if strings.HasPrefix(headerPathValue, prei.prefix) {
				log.DefaultLogger.Warnf("prefix route rule match success")
				
				return prei
			}
		}
	}
	log.DefaultLogger.Warnf("prefix route rule match failed")
	
	return nil
}

func (prei *PrefixRouteRuleImpl) FinalizeRequestHeaders(headers map[string]string, requestInfo types.RequestInfo) {
	prei.finalizePathHeader(headers, prei.prefix)
}

//
type RegexRouteRuleImpl struct {
	RouteRuleImplBase
	regexStr     string
	regexPattern regexp.Regexp
}

func (rrei *RegexRouteRuleImpl) Matcher() string {

	return rrei.regexStr
}

func (rrei *RegexRouteRuleImpl) MatchType() types.PathMatchType {

	return types.Regex
}

func (rrei *RegexRouteRuleImpl) Match(headers map[string]string, randomValue uint64) types.Route {
	if rrei.matchRoute(headers, randomValue) {
		if headerPathValue, ok := headers[strings.ToLower(protocol.MosnHeaderPathKey)]; ok {
			
			if rrei.regexPattern.MatchString(headerPathValue) {
				log.DefaultLogger.Warnf("regex route rule match success")
				
				return rrei
			}
		}
	}
	log.DefaultLogger.Warnf("regex route rule match failed")
	
	return nil
}

func (rrei *RegexRouteRuleImpl) FinalizeRequestHeaders(headers map[string]string, requestInfo types.RequestInfo) {
	rrei.finalizePathHeader(headers, rrei.regexStr)
}
