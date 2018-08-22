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
	"strings"
	"time"

	"github.com/alipay/sofa-mosn/pkg/types"
)

type headerParser struct {
	headersToAdd    []types.Pair
	headersToRemove []*lowerCaseString
}

type matchable interface {
	Match(headers map[string]string, randomValue uint64) types.Route
}

type info interface {
	GetRouterName() string
}

type RouteBase interface {
	types.Route
	types.RouteRule
	matchable
	info
}

type shadowPolicyImpl struct {
	cluster    string
	runtimeKey string
}

func (spi *shadowPolicyImpl) ClusterName() string {
	return spi.cluster
}

func (spi *shadowPolicyImpl) RuntimeKey() string {
	return spi.runtimeKey
}

type lowerCaseString struct {
	str string
}

func (lcs *lowerCaseString) Lower() {
	lcs.str = strings.ToLower(lcs.str)
}

func (lcs *lowerCaseString) Equal(rhs types.LowerCaseString) bool {
	return lcs.str == rhs.Get()
}

func (lcs *lowerCaseString) Get() string {
	return lcs.str
}

type hashPolicyImpl struct {
	hashImpl []*hashMethod
}

type hashMethod struct {
}

type decoratorImpl struct {
	Operation string
}

func (di *decoratorImpl) apply(span types.Span) {
	if di.Operation != "" {
		span.SetOperation(di.Operation)
	}
}

func (di *decoratorImpl) getOperation() string {
	return di.Operation
}

type rateLimitPolicyImpl struct {
	rateLimitEntries []types.RateLimitPolicyEntry
	maxStageNumber   uint64
}

func (rp *rateLimitPolicyImpl) Enabled() bool {

	return true
}

func (rp *rateLimitPolicyImpl) GetApplicableRateLimit(stage string) []types.RateLimitPolicyEntry {

	return rp.rateLimitEntries
}

type retryPolicyImpl struct {
	retryOn      bool
	retryTimeout time.Duration
	numRetries   uint32
}

func (p *retryPolicyImpl) RetryOn() bool {
	return p.retryOn
}

func (p *retryPolicyImpl) TryTimeout() time.Duration {
	return p.retryTimeout
}

func (p *retryPolicyImpl) NumRetries() uint32 {
	return p.numRetries
}

// todo implement CorsPolicy

type runtimeData struct {
	key          string
	defaultvalue uint64
}

type rateLimitPolicyEntryImpl struct {
	stage      uint64
	disableKey string
	actions    rateLimitAction
}

func (rpei *rateLimitPolicyEntryImpl) Stage() uint64 {
	return rpei.stage
}

func (rpei *rateLimitPolicyEntryImpl) DisableKey() string {
	return rpei.disableKey
}

func (rpei *rateLimitPolicyEntryImpl) PopulateDescriptors(route types.RouteRule, descriptors []types.Descriptor, localSrvCluster string,
	headers map[string]string, remoteAddr string) {
}

type rateLimitAction interface{}

type weightedClusterEntry struct {
	clusterName                  string
	runtimeKey                   string
	loader                       types.Loader
	clusterWeight                uint32
	clusterMetadataMatchCriteria *MetadataMatchCriteriaImpl
}

func (wc *weightedClusterEntry) GetClusterMetadataMatchCriteria() *MetadataMatchCriteriaImpl {
	return wc.clusterMetadataMatchCriteria
}

type routerPolicy struct {
	retryOn      bool
	retryTimeout time.Duration
	numRetries   uint32
}

func (p *routerPolicy) RetryOn() bool {
	return p.retryOn
}

func (p *routerPolicy) TryTimeout() time.Duration {
	return p.retryTimeout
}

func (p *routerPolicy) NumRetries() uint32 {
	return p.numRetries
}

func (p *routerPolicy) RetryPolicy() types.RetryPolicy {
	return p
}

func (p *routerPolicy) ShadowPolicy() types.ShadowPolicy {
	return nil
}

func (p *routerPolicy) CorsPolicy() types.CorsPolicy {
	return nil
}

func (p *routerPolicy) LoadBalancerPolicy() types.LoadBalancerPolicy {
	return nil
}
