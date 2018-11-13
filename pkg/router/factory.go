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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	RegisterRouterRule(DefaultSofaRouterRuleFactory)
	RegisterMakeHandlerChain(DefaultMakeHandlerChain)
}

var defaultRouterRuleFactory RouterRuleFactory

func RegisterRouterRule(f RouterRuleFactory) {
	defaultRouterRuleFactory = f
}

func DefaultSofaRouterRuleFactory(base *RouteRuleImplBase, headers []v2.HeaderMatcher) RouteBase {
	for _, header := range headers {
		if header.Name == types.SofaRouteMatchKey {
			return &SofaRouteRuleImpl{
				RouteRuleImplBase: base,
				matchValue:        header.Value,
			}
		}
	}
	return nil
}

var makeHandlerChain MakeHandlerChain

func RegisterMakeHandlerChain(f MakeHandlerChain) {
	makeHandlerChain = f
}

type simpleHandler struct {
	route types.Route
}

func (h *simpleHandler) IsAvailable(ctx context.Context) bool {
	return true
}
func (h *simpleHandler) Route() types.Route {
	return h.route
}
func DefaultMakeHandlerChain(headers types.HeaderMap, routers types.Routers) *RouteHandlerChain {
	if r := routers.Route(headers, 1); r != nil {
		return NewRouteHandlerChain(context.Background(), []types.RouteHandler{
			&simpleHandler{route: r},
		})
	}
	return nil
}

func CallMakeHandlerChain(headers types.HeaderMap, routers types.Routers) *RouteHandlerChain {
	return makeHandlerChain(headers, routers)
}
