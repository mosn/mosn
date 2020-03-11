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

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// RouteHandlerChain returns first available handler's router
type RouteHandlerChain struct {
	ctx            context.Context
	handlers       []types.RouteHandler
	clusterManager types.ClusterManager
	index          int
}

func NewRouteHandlerChain(ctx context.Context, clusterManager types.ClusterManager, handlers []types.RouteHandler) *RouteHandlerChain {
	return &RouteHandlerChain{
		ctx:            ctx,
		handlers:       handlers,
		clusterManager: clusterManager,
		index:          0,
	}
}

func (hc *RouteHandlerChain) DoNextHandler() (types.ClusterSnapshot, api.Route) {
	handler := hc.Next()
	if handler == nil {
		return nil, nil
	}
	snapshot, status := handler.IsAvailable(hc.ctx, hc.clusterManager)
	switch status {
	case types.HandlerAvailable:
		return snapshot, handler.Route()
	case types.HandlerNotAvailable:
	case types.HandlerStop:
		return nil, nil
	default:
		log.Proxy.Warnf(hc.ctx, RouterLogFormat, "default handler chain", "unexpected status", status)
	}
	return hc.DoNextHandler()
}
func (hc *RouteHandlerChain) Next() types.RouteHandler {
	if hc.index >= len(hc.handlers) {
		return nil
	}
	h := hc.handlers[hc.index]
	hc.index++
	return h
}
