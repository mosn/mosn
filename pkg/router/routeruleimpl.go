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
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"fmt"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	httpmosn "github.com/alipay/sofa-mosn/pkg/protocol/http"
	"github.com/alipay/sofa-mosn/pkg/types"
	multimap "github.com/jwangsadinata/go-multimap/slicemultimap"
)

// NewRouteRuleImplBase
// new routerule implement basement
func NewRouteRuleImplBase(vHost *VirtualHostImpl, route *v2.Router) (*RouteRuleImplBase, error) {
	routeRuleImplBase := RouteRuleImplBase{
		vHost:                 vHost,
		routerMatch:           route.Match,
		routerAction:          route.Route,
		clusterName:           route.Route.ClusterName,
		randInstance:          rand.New(rand.NewSource(time.Now().UnixNano())),
		configHeaders:         getRouterHeaders(route.Match.Headers),
		prefixRewrite:         route.Route.PrefixRewrite,
		hostRewrite:           route.Route.HostRewrite,
		autoHostRewrite:       route.Route.AutoHostRewrite,
		requestHeadersParser:  getHeaderParser(route.Route.RequestHeadersToAdd, nil),
		responseHeadersParser: getHeaderParser(route.Route.ResponseHeadersToAdd, route.Route.ResponseHeadersToRemove),
		perFilterConfig:       route.PerFilterConfig,
		policy:                &routerPolicy{},
	}

	routeRuleImplBase.weightedClusters, routeRuleImplBase.totalClusterWeight = getWeightedClusterEntry(route.Route.WeightedClusters)
	if route.Route.RetryPolicy != nil {
		routeRuleImplBase.policy.retryOn = route.Route.RetryPolicy.RetryOn
		routeRuleImplBase.policy.retryTimeout = route.Route.RetryPolicy.RetryTimeout
		routeRuleImplBase.policy.numRetries = route.Route.RetryPolicy.NumRetries
	}

	// todo add header match to route base
	// generate metadata match criteria from router's metadata
	if len(route.Route.MetadataMatch) > 0 {
		subsetLBMetaData := route.Route.MetadataMatch
		routeRuleImplBase.metadataMatchCriteria = NewMetadataMatchCriteriaImpl(subsetLBMetaData)

		routeRuleImplBase.metaData = getClusterMosnLBMetaDataMap(subsetLBMetaData)
	}

	return &routeRuleImplBase, nil
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
	clusterHeaderName           lowerCaseString
	clusterNotFoundResponseCode httpmosn.Code
	timeout                     time.Duration
	runtime                     v2.RuntimeUInt32
	hostRedirect                string
	pathRedirect                string
	httpsRedirect               bool
	retryPolicy                 *retryPolicyImpl
	rateLimitPolicy             *rateLimitPolicyImpl

	routerAction v2.RouteAction
	routerMatch  v2.RouterMatch

	shadowPolicy          *shadowPolicyImpl
	priority              types.ResourcePriority
	configHeaders         []*types.HeaderData //
	configQueryParameters []types.QueryParameterMatcher
	weightedClusters      map[string]weightedClusterEntry //key is the weighted cluster's name
	totalClusterWeight    uint32
	hashPolicy            hashPolicyImpl

	metadataMatchCriteria *MetadataMatchCriteriaImpl // sorted and unique metadata
	metaData              types.RouteMetaData        // not sorted, only value in hashed

	requestHeadersParser  *headerParser
	responseHeadersParser *headerParser

	opaqueConfig multimap.MultiMap

	directResponseCode httpmosn.Code
	directResponseBody string
	policy             *routerPolicy
	virtualClusters    *VirtualClusterEntry
	randInstance       *rand.Rand
	randMutex          sync.Mutex

	perFilterConfig map[string]interface{}
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

// types.RouteRule
// Select Cluster for Routing
// if weighted cluster is nil, return clusterName directly, else
// select cluster from weighted-clusters
func (rri *RouteRuleImplBase) ClusterName() string {
	if len(rri.weightedClusters) == 0 {
		return rri.clusterName
	}

	// use randInstance to avoid global lock contention
	rri.randMutex.Lock()
	selectedValue := rri.randInstance.Intn(int(rri.totalClusterWeight))
	rri.randMutex.Unlock()

	for _, weightCluster := range rri.weightedClusters {

		selectedValue = selectedValue - int(weightCluster.clusterWeight)
		if selectedValue <= 0 {
			return weightCluster.clusterName
		}
	}

	log.DefaultLogger.Errorf("Something wrong when choosing weighted cluster")
	return rri.clusterName
}

func (rri *RouteRuleImplBase) GlobalTimeout() time.Duration {

	return rri.routerAction.Timeout
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

func (rri *RouteRuleImplBase) MetadataMatchCriteria(clusterName string) types.MetadataMatchCriteria {
	// if clusterName belongs to a weighted cluster
	if matchCriteria, ok := rri.weightedClusters[clusterName]; ok {
		return matchCriteria.clusterMetadataMatchCriteria
	}

	return rri.metadataMatchCriteria
}

func (rri *RouteRuleImplBase) UpdateMetaDataMatchCriteria(metadata map[string]string) error {
	if metadata == nil {
		return fmt.Errorf("UpdateMetaDataMatchCriteria fail: metadata is nil")
	}

	mmc := NewMetadataMatchCriteriaImpl(metadata)
	if mmc == nil {
		return fmt.Errorf("UpdateMetaDataMatchCriteria fail: NewMetadataMatchCriteriaImpl error")
	}

	rri.metadataMatchCriteria = mmc
	rri.metaData = getClusterMosnLBMetaDataMap(metadata)

	return nil
}

func (rri *RouteRuleImplBase) PerFilterConfig() map[string]interface{} {
	return rri.perFilterConfig
}

func (rri *RouteRuleImplBase) matchRoute(headers types.HeaderMap, randomValue uint64) bool {
	// todo check runtime
	// 1. match headers' KV
	if !ConfigUtilityInst.MatchHeaders(headers, rri.configHeaders) {
		log.DefaultLogger.Errorf("RouteRuleImplBase matchRoute, match headers error")
		return false
	}

	// 2. match query parameters
	var queryParams types.QueryParams

	if QueryString, ok := headers.Get(protocol.MosnHeaderQueryStringKey); ok {
		queryParams = httpmosn.ParseQueryString(QueryString)
	}

	if len(queryParams) == 0 {
		return true
	}

	if !ConfigUtilityInst.MatchQueryParams(queryParams, rri.configQueryParameters) {
		log.DefaultLogger.Errorf("RouteRuleImplBase matchRoute, match query params error")
		return false
	}

	return true
}

func (rri *RouteRuleImplBase) WeightedCluster() map[string]weightedClusterEntry {
	return rri.weightedClusters
}

func (rri *RouteRuleImplBase) finalizePathHeader(headers types.HeaderMap, matchedPath string) {
	if len(rri.prefixRewrite) < 1 {
		return
	}
	if path, ok := headers.Get(protocol.MosnHeaderPathKey); ok {
		if !strings.HasPrefix(path, matchedPath) {
			log.DefaultLogger.Errorf("expect the Path %s has prefix %s but not", path, matchedPath)
			return
		}
		headers.Set(protocol.MosnOriginalHeaderPathKey, path)
		headers.Set(protocol.MosnHeaderPathKey, rri.prefixRewrite+path[len(matchedPath):])
	}
}

func (rri *RouteRuleImplBase) FinalizeRequestHeaders(headers types.HeaderMap, requestInfo types.RequestInfo) {
	rri.finalizeRequestHeaders(headers, requestInfo)
}

func (rri *RouteRuleImplBase) finalizeRequestHeaders(headers types.HeaderMap, requestInfo types.RequestInfo) {
	rri.requestHeadersParser.evaluateHeaders(headers, requestInfo)
	rri.vHost.requestHeadersParser.evaluateHeaders(headers, requestInfo)
	rri.vHost.globalRouteConfig.requestHeadersParser.evaluateHeaders(headers, requestInfo)
	if len(rri.hostRewrite) > 0 {
		headers.Set(protocol.IstioHeaderHostKey, rri.hostRewrite)
	}
}

func (rri *RouteRuleImplBase) FinalizeResponseHeaders(headers types.HeaderMap, requestInfo types.RequestInfo) {
	rri.responseHeadersParser.evaluateHeaders(headers, requestInfo)
	rri.vHost.responseHeadersParser.evaluateHeaders(headers, requestInfo)
	rri.vHost.globalRouteConfig.responseHeadersParser.evaluateHeaders(headers, requestInfo)
}

func SofaRouterFactory(headers []v2.HeaderMatcher) RouteBase {
	for _, header := range headers {
		if header.Name == types.SofaRouteMatchKey {
			return &SofaRouteRuleImpl{
				matchName:  header.Name,
				matchValue: header.Value,
			}
		}
	}

	return nil
}

type SofaRouteRuleImpl struct {
	*RouteRuleImplBase
	matchName  string
	matchValue string
}

func (srri *SofaRouteRuleImpl) Matcher() string {

	return srri.matchValue
}

func (srri *SofaRouteRuleImpl) MatchType() types.PathMatchType {

	return types.SofaHeader
}

func (srri *SofaRouteRuleImpl) Match(headers types.HeaderMap, randomValue uint64) types.Route {
	if value, ok := headers.Get(types.SofaRouteMatchKey); ok {
		if value == srri.matchValue || srri.matchValue == ".*" {
			log.DefaultLogger.Debugf("Sofa router matches success")
			return srri
		}

		log.DefaultLogger.Warnf(" Sofa router matches failure, service name = %s", value)
	}

	log.DefaultLogger.Debugf("No service key found in header, sofa router matcher failure")

	return nil
}

func (srri *SofaRouteRuleImpl) RouteRule() types.RouteRule {
	return srri
}

func (srri *SofaRouteRuleImpl) FinalizeRequestHeaders(headers types.HeaderMap, requestInfo types.RequestInfo) {

}

type PathRouteRuleImpl struct {
	*RouteRuleImplBase
	path string
}

func (prri *PathRouteRuleImpl) Matcher() string {

	return prri.path
}

func (prri *PathRouteRuleImpl) MatchType() types.PathMatchType {

	return types.Exact
}

// PathRouteRuleImpl used to "match path" with "exact comparing"
func (prri *PathRouteRuleImpl) Match(headers types.HeaderMap, randomValue uint64) types.Route {
	// match base rule first
	log.StartLogger.Debugf("path route rule match invoked")
	if prri.matchRoute(headers, randomValue) {

		if headerPathValue, ok := headers.Get(strings.ToLower(protocol.MosnHeaderPathKey)); ok {

			if prri.caseSensitive {
				if headerPathValue == prri.path {
					log.DefaultLogger.Debugf("path route rule match success in caseSensitive scene")
					return prri
				}
			} else if strings.EqualFold(headerPathValue, prri.path) {
				log.DefaultLogger.Debugf("path route rule match success with exact matching ")
				return prri
			}
		}
	}
	log.DefaultLogger.Debugf("path route rule match failed")

	return nil
}

func (prri *PathRouteRuleImpl) RouteRule() types.RouteRule {
	return prri
}

func (prri *PathRouteRuleImpl) FinalizeRequestHeaders(headers types.HeaderMap, requestInfo types.RequestInfo) {
	prri.finalizeRequestHeaders(headers, requestInfo)
	prri.finalizePathHeader(headers, prri.path)
}

// PrefixRouteRuleImpl used to "match path" with "prefix match"
type PrefixRouteRuleImpl struct {
	*RouteRuleImplBase
	prefix string
}

func (prei *PrefixRouteRuleImpl) Matcher() string {

	return prei.prefix
}

func (prei *PrefixRouteRuleImpl) MatchType() types.PathMatchType {

	return types.Prefix
}

func (prei *PrefixRouteRuleImpl) Match(headers types.HeaderMap, randomValue uint64) types.Route {

	if prei.matchRoute(headers, randomValue) {

		if headerPathValue, ok := headers.Get(strings.ToLower(protocol.MosnHeaderPathKey)); ok {

			if strings.HasPrefix(headerPathValue, prei.prefix) {
				log.DefaultLogger.Debugf("prefix route rule match success")
				return prei
			}
		}
	}
	log.DefaultLogger.Debugf("prefix route rule match failed")

	return nil
}

func (prei *PrefixRouteRuleImpl) RouteRule() types.RouteRule {
	return prei
}

func (prei *PrefixRouteRuleImpl) FinalizeRequestHeaders(headers types.HeaderMap, requestInfo types.RequestInfo) {
	prei.finalizeRequestHeaders(headers, requestInfo)
	prei.finalizePathHeader(headers, prei.prefix)
}

// RegexRouteRuleImpl used to "match path" with "regex match"
type RegexRouteRuleImpl struct {
	*RouteRuleImplBase
	regexStr     string
	regexPattern *regexp.Regexp
}

func (rrei *RegexRouteRuleImpl) Matcher() string {

	return rrei.regexStr
}

func (rrei *RegexRouteRuleImpl) MatchType() types.PathMatchType {

	return types.Regex
}

func (rrei *RegexRouteRuleImpl) Match(headers types.HeaderMap, randomValue uint64) types.Route {
	if rrei.matchRoute(headers, randomValue) {
		if headerPathValue, ok := headers.Get(strings.ToLower(protocol.MosnHeaderPathKey)); ok {

			if rrei.regexPattern.MatchString(headerPathValue) {
				log.DefaultLogger.Debugf("regex route rule match success")

				return rrei
			}
		}
	}
	log.DefaultLogger.Debugf("regex route rule match failed")

	return nil
}

func (rrei *RegexRouteRuleImpl) RouteRule() types.RouteRule {
	return rrei
}

func (rrei *RegexRouteRuleImpl) FinalizeRequestHeaders(headers types.HeaderMap, requestInfo types.RequestInfo) {
	rrei.finalizeRequestHeaders(headers, requestInfo)
	rrei.finalizePathHeader(headers, rrei.regexStr)
}
