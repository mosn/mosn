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
	"testing"

	"github.com/golang/mock/gomock"
	"mosn.io/api"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/mosn/pkg/types"
)

func TestDoRouteHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	routers := mock.NewMockRouters(ctrl)
	routers.EXPECT().MatchRoute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ api.HeaderMap) api.Route {
		route := mock.NewMockRoute(ctrl)
		route.EXPECT().RouteRule().DoAndReturn(func() api.RouteRule {
			rule := mock.NewMockRouteRule(ctrl)
			rule.EXPECT().ClusterName(context.TODO()).Return("test").AnyTimes()
			return rule
		}).AnyTimes()
		return route
	}).AnyTimes()
	cm := mock.NewMockClusterManager(ctrl)
	cm.EXPECT().GetClusterSnapshot(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string) types.ClusterSnapshot {
		snap := mock.NewMockClusterSnapshot(ctrl)
		return snap
	})
	headers := mock.NewMockHeaderMap(ctrl)
	// match default
	f := GetMakeHandlerFunc("")
	snap, route := f.DoRouteHandler(context.Background(), headers, routers, cm)
	if !(snap != nil &&
		route != nil) {
		t.Fatal("do route handler failed")
	}
}

type mockHandler struct {
	route api.Route
}

func (h *mockHandler) Route() api.Route {
	return h.route
}

func (h *mockHandler) IsAvailable(ctx context.Context, manager types.ClusterManager) (types.ClusterSnapshot, types.HandlerStatus) {
	return nil, types.HandlerAvailable
}

func TestDoRouteHandlerExtend(t *testing.T) {
	RegisterMakeHandler("test", func(ctx context.Context, headers api.HeaderMap, routers types.Routers) types.RouteHandler {
		return &mockHandler{
			route: routers.MatchRoute(ctx, headers),
		}
	}, false)
	ctrl := gomock.NewController(t)
	routers := mock.NewMockRouters(ctrl)
	routers.EXPECT().MatchRoute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ api.HeaderMap) api.Route {
		route := mock.NewMockRoute(ctrl)
		return route
	}).AnyTimes()
	cm := mock.NewMockClusterManager(ctrl)
	headers := mock.NewMockHeaderMap(ctrl)
	f := GetMakeHandlerFunc("test")
	_, route := f.DoRouteHandler(context.Background(), headers, routers, cm)
	if route == nil {
		t.Fatal("do route handler failed")
	}
}
