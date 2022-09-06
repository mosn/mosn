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
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

type simpleHandler struct {
	route api.Route
}

func (h *simpleHandler) IsAvailable(ctx context.Context, manager types.ClusterManager) (types.ClusterSnapshot, types.HandlerStatus) {
	if h.route == nil {
		return nil, types.HandlerNotAvailable
	}
	clusterName := h.Route().RouteRule().ClusterName(ctx)
	snapshot := manager.GetClusterSnapshot(ctx, clusterName)
	return snapshot, types.HandlerAvailable
}

func (h *simpleHandler) Route() api.Route {
	return h.route
}

func DefaultMakeHandler(ctx context.Context, headers api.HeaderMap, routers types.Routers) types.RouteHandler {
	r := routers.MatchRoute(ctx, headers)
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, RouterLogFormat, "DefaultMakeHandler", "MatchRoute", fmt.Sprintf("matched a route: %v", r))
	}
	return &simpleHandler{
		route: r,
	}

}

// MakeHandlerFunc creates a router handler for mosn
type MakeHandlerFunc func(ctx context.Context, headers api.HeaderMap, routers types.Routers) types.RouteHandler

func (factory MakeHandlerFunc) DoRouteHandler(ctx context.Context, headers api.HeaderMap, routers types.Routers, clusterManager types.ClusterManager) (types.ClusterSnapshot, api.Route) {
	handler := factory(ctx, headers, routers)
	if handler == nil {
		lv, _ := variable.Get(ctx, types.VariableListenerName)
		lname := lv.(string)
		log.Proxy.Errorf(ctx, RouterLogFormat, "DoRouteHandler", "create handler failed", lname)
		return nil, nil
	}
	snapshot, status := handler.IsAvailable(ctx, clusterManager)
	switch status {
	case types.HandlerAvailable:
		return snapshot, handler.Route()
	default:
		log.Proxy.Warnf(ctx, RouterLogFormat, "DoRouteHandler", "matched failed", status)
		return nil, nil
	}
}
