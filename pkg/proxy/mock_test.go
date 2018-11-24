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

package proxy

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/types"
)

// Mock interface for test
type mockRouterWrapper struct {
	types.RouterWrapper
}

func (rw *mockRouterWrapper) GetRouters() types.Routers {
	return &mockRouters{}
}

type mockRouters struct {
	types.Routers
}

func (r *mockRouters) Route(types.HeaderMap, uint64) types.Route {
	return &mockRoute{}
}

type mockRoute struct {
	types.Route
}

func (r *mockRoute) RouteRule() types.RouteRule {
	return &mockRouteRule{}
}

func (r *mockRoute) DirectResponseRule() types.DirectResponseRule {
	return nil
}

type mockRouteRule struct {
	types.RouteRule
}

func (r *mockRouteRule) ClusterName() string {
	return "test"
}

type mockClusterManager struct {
	types.ClusterManager
}

func (m *mockClusterManager) GetClusterSnapshot(ctx context.Context, name string) types.ClusterSnapshot {
	return &mockClusterSnapshot{}
}
func (m *mockClusterManager) PutClusterSnapshot(snapshot types.ClusterSnapshot) {
}

type mockClusterSnapshot struct {
	types.ClusterSnapshot
}
