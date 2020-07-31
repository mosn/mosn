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
	"fmt"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

func init() {
	RegisterRouterRule(DefaultSofaRouterRuleFactory, 1)
	RegisterMakeHandlerChain(DefaultMakeHandlerChain, 1)
}

var defaultRouterRuleFactoryOrder routerRuleFactoryOrder

func RegisterRouterRule(f RouterRuleFactory, order uint32) {
	if defaultRouterRuleFactoryOrder.order < order {
		log.DefaultLogger.Infof(RouterLogFormat, "Extend", "RegisterRouterRule", fmt.Sprintf("order is %d", order))
		defaultRouterRuleFactoryOrder.factory = f
		defaultRouterRuleFactoryOrder.order = order
	} else {
		msg := fmt.Sprintf("current register order is %d, order %d register failed", defaultRouterRuleFactoryOrder.order, order)
		log.DefaultLogger.Errorf(RouterLogFormat, "Extend", "RegisterRouterRule", msg)
	}
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

var makeHandlerChainOrder handlerChainOrder

func RegisterMakeHandlerChain(f MakeHandlerChain, order uint32) {
	if makeHandlerChainOrder.order < order {
		log.DefaultLogger.Infof(RouterLogFormat, "Extend", "RegisterHandlerChain", fmt.Sprintf("order is %d", order))
		makeHandlerChainOrder.makeHandlerChain = f
		makeHandlerChainOrder.order = order
	} else {
		msg := fmt.Sprintf("current register order is %d, order %d register failed", makeHandlerChainOrder.order, order)
		log.DefaultLogger.Errorf(RouterLogFormat, "Extend", "RegisterHandlerChain", msg)
	}
}

type simpleHandler struct {
	route api.Route
}

func (h *simpleHandler) IsAvailable(ctx context.Context, manager types.ClusterManager) (types.ClusterSnapshot, types.HandlerStatus) {
	if h.route == nil {
		return nil, types.HandlerNotAvailable
	}
	clusterName := h.Route().RouteRule().ClusterName()
	snapshot := manager.GetClusterSnapshot(context.Background(), clusterName)
	return snapshot, types.HandlerAvailable
}

func (h *simpleHandler) Route() api.Route {
	return h.route
}

func DefaultMakeHandlerChain(ctx context.Context, headers api.HeaderMap, routers types.Routers, clusterManager types.ClusterManager) *RouteHandlerChain {
	var handlers []types.RouteHandler
	if r := routers.MatchRoute(headers, 1); r != nil {
		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(ctx, RouterLogFormat, "DefaultHandklerChain", "MatchRoute", fmt.Sprintf("matched a route: %v", r))
		}
		handlers = append(handlers, &simpleHandler{route: r})
	}
	return NewRouteHandlerChain(ctx, clusterManager, handlers)
}

func CallMakeHandlerChain(ctx context.Context, headers api.HeaderMap, routers types.Routers, clusterManager types.ClusterManager) *RouteHandlerChain {
	return makeHandlerChainOrder.makeHandlerChain(ctx, headers, routers, clusterManager)
}
