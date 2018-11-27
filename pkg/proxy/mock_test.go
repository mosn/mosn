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
	routers types.Routers
}

func (rw *mockRouterWrapper) GetRouters() types.Routers {
	if rw.routers != nil {
		return rw.routers
	}
	return &mockRouters{}
}

type mockRouters struct {
	types.Routers
	route types.Route
}

func (r *mockRouters) Route(types.HeaderMap, uint64) types.Route {
	if r.route != nil {
		return r.route
	}
	return &mockRoute{}
}

type mockRoute struct {
	types.Route
	rule   types.RouteRule
	direct types.DirectResponseRule
}

func (r *mockRoute) RouteRule() types.RouteRule {
	if r.rule != nil {
		return r.rule
	}
	return &mockRouteRule{}
}

func (r *mockRoute) DirectResponseRule() types.DirectResponseRule {
	if r.direct != nil {
		return r.direct
	}
	return nil
}

type mockRouteRule struct {
	types.RouteRule
}

func (r *mockRouteRule) ClusterName() string {
	return "test"
}

type mockDirectRule struct {
	status int
	body   string
}

func (r *mockDirectRule) StatusCode() int {
	return r.status
}

func (r *mockDirectRule) Body() string {
	return r.body
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

type mockResponseSender struct {
	// receive data
	headers  types.HeaderMap
	data     types.IoBuffer
	trailers types.HeaderMap
}

func (s *mockResponseSender) AppendHeaders(ctx context.Context, headers types.HeaderMap, endStream bool) error {
	s.headers = headers
	return nil
}

func (s *mockResponseSender) AppendData(ctx context.Context, data types.IoBuffer, endStream bool) error {
	s.data = data
	return nil
}

func (s *mockResponseSender) AppendTrailers(ctx context.Context, trailers types.HeaderMap) error {
	s.trailers = trailers
	return nil
}

func (s *mockResponseSender) GetStream() types.Stream {
	return &mockStream{}
}

type mockStream struct {
	types.Stream
}

func (s *mockStream) ResetStream(reason types.StreamResetReason) {
	// do nothing
}

type mockReadFilterCallbacks struct {
	types.ReadFilterCallbacks
}

func (cb *mockReadFilterCallbacks) Connection() types.Connection {
	return &mockConnection{}
}

type mockConnection struct {
	types.Connection
}

func (c *mockConnection) ID() uint64 {
	return 0
}
