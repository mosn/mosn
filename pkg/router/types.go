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
	"errors"
	"strings"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// [sub module] & [function] & msg
const RouterLogFormat = "[router] [%s] [%s] %v"

var (
	ErrNilRouterConfig      = errors.New("router config is nil")
	ErrNoVirtualHost        = errors.New("virtual host is nil")
	ErrNoRouters            = errors.New("routers is nil")
	ErrDuplicateVirtualHost = errors.New("duplicate domain virtual host")
	ErrUnexpected           = errors.New("an unexpected error occurs")
	ErrRouterFactory        = errors.New("default router factory create router failed")
)

type headerFormatter interface {
	format(requestInfo types.RequestInfo) string
	append() bool
}

type headerPair struct {
	headerName      *lowerCaseString
	headerFormatter headerFormatter
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

type weightedClusterEntry struct {
	clusterName                  string
	clusterWeight                uint32
	clusterMetadataMatchCriteria *MetadataMatchCriteriaImpl
}

type Matchable interface {
	Match(headers types.HeaderMap, randomValue uint64) types.Route
}

type RouteBase interface {
	types.Route
	types.PathMatchCriterion
	Matchable
}

// Policy
type policy struct {
	retryPolicy  *retryPolicyImpl
	shadowPolicy *shadowPolicyImpl //TODO: not implement yet
}

func (p *policy) RetryPolicy() types.RetryPolicy {
	return p.retryPolicy
}

func (p *policy) ShadowPolicy() types.ShadowPolicy {
	return p.shadowPolicy
}

type retryPolicyImpl struct {
	retryOn      bool
	retryTimeout time.Duration
	numRetries   uint32
}

func (p *retryPolicyImpl) RetryOn() bool {
	if p == nil {
		return false
	}
	return p.retryOn
}

func (p *retryPolicyImpl) TryTimeout() time.Duration {
	if p == nil {
		return 0
	}
	return p.retryTimeout
}

func (p *retryPolicyImpl) NumRetries() uint32 {
	if p == nil {
		return 0
	}
	return p.numRetries
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

// RouterRuleFactory creates a RouteBase
type RouterRuleFactory func(base *RouteRuleImplBase, header []v2.HeaderMatcher) RouteBase

// MakeHandlerChain creates a RouteHandlerChain, should not returns a nil handler chain, or the stream filters will be ignored
type MakeHandlerChain func(context.Context, types.HeaderMap, types.Routers, types.ClusterManager) *RouteHandlerChain

// The reigister order, is a wrapper of registered factory
// We register a factory with order, a new factory can replace old registered factory only if the register order
// ig greater than the old one.
type routerRuleFactoryOrder struct {
	factory RouterRuleFactory
	order   uint32
}
type handlerChainOrder struct {
	makeHandlerChain MakeHandlerChain
	order            uint32
}
