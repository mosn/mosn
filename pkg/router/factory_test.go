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
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type mockRouteBase struct {
	*RouteRuleImplBase
}

func (r *mockRouteBase) Match(headers types.HeaderMap, randomValue uint64) types.Route {
	return nil
}
func (r *mockRouteBase) Matcher() string {
	return ""
}
func (r *mockRouteBase) MatchType() types.PathMatchType {
	return 999
}

func _TestRouterRuleFactory(base *RouteRuleImplBase, headers []v2.HeaderMatcher) RouteBase {
	return &mockRouteBase{base}
}

func _TestLowerRouterRuleFactory(base *RouteRuleImplBase, headers []v2.HeaderMatcher) RouteBase {
	return nil
}

func resetRouteRuleFactory() {
	defaultRouterRuleFactoryOrder.factory = DefaultSofaRouterRuleFactory
	defaultRouterRuleFactoryOrder.order = 1
}

func TestRegisterRuleOrder(t *testing.T) {
	testCases := []struct {
		f     RouterRuleFactory
		order uint32
		check func(rb RouteBase) bool
	}{
		// Defaiult, register by init
		{
			f:     nil,
			order: 0,
			check: func(rb RouteBase) bool {
				_, ok := rb.(*SofaRouteRuleImpl)
				return ok
			},
		},
		// Register higher order
		{
			f:     _TestRouterRuleFactory,
			order: 2,
			check: func(rb RouteBase) bool {
				_, ok := rb.(*mockRouteBase)
				return ok
			},
		},
		// Register lower order, will failed
		{
			f:     _TestLowerRouterRuleFactory,
			order: 1,
			check: func(rb RouteBase) bool {
				return rb != nil
			},
		},
	}
	base := &RouteRuleImplBase{}
	headers := []v2.HeaderMatcher{
		v2.HeaderMatcher{
			Name:  types.SofaRouteMatchKey,
			Value: "test",
		},
	}
	for i, tc := range testCases {
		if tc.f != nil {
			RegisterRouterRule(tc.f, tc.order)
		}
		rb := defaultRouterRuleFactoryOrder.factory(base, headers)
		if !tc.check(rb) {
			t.Errorf("#%d register unexpected", i)
		}
	}
	// Clear Register
	resetRouteRuleFactory()
}

// HandlerChain Register test in handlerchain_test.go
