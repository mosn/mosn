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
	"net"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
)

var mockProtocol = types.ProtocolName("mockProtocol")

func init() {
	trace.RegisterDriver("SOFATracer", trace.NewDefaultDriverImpl())
	trace.RegisterTracerBuilder("SOFATracer", mockProtocol, func(config map[string]interface{}) (api.Tracer, error) {
		return &mockTracer{}, nil
	})
}

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
	route api.Route
}

func (r *mockRouters) MatchRoute(context.Context, types.HeaderMap) types.Route {
	if r.route != nil {
		return r.route
	}
	return &mockRoute{}
}

type mockRoute struct {
	api.Route
	rule   api.RouteRule
	direct api.DirectResponseRule
}

func (r *mockRoute) RouteRule() api.RouteRule {
	if r.rule != nil {
		return r.rule
	}
	return &mockRouteRule{}
}

func (r *mockRoute) DirectResponseRule() api.DirectResponseRule {
	if r.direct != nil {
		return r.direct
	}
	return nil
}

type mockRouteRule struct {
	api.RouteRule
	upstreamProtocol string
}

func (r *mockRouteRule) ClusterName(ctx context.Context) string {
	return "test"
}

func (r *mockRouteRule) UpstreamProtocol() string {
	return r.upstreamProtocol
}

func (c *mockRouteRule) FinalizeResponseHeaders(ctx context.Context, headers api.HeaderMap, requestInfo api.RequestInfo) {
	return
}

func (c *mockRouteRule) GlobalTimeout() time.Duration {
	return 10 ^ 6*time.Millisecond
}

func (c *mockRouteRule) Policy() api.Policy {
	return &mockPolicy{}
}

type mockPolicy struct {
	api.Policy
}

func (p *mockPolicy) RetryPolicy() api.RetryPolicy {
	return &mockRetryPolicy{}
}

type mockRetryPolicy struct {
	api.RetryPolicy
}

func (p *mockRetryPolicy) TryTimeout() time.Duration {
	return time.Millisecond
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

func (mcs *mockClusterSnapshot) ClusterInfo() types.ClusterInfo {
	return nil
}

type mockResponseSender struct {
	// receive data
	headers  api.HeaderMap
	data     types.IoBuffer
	trailers api.HeaderMap
}

func (s *mockResponseSender) AppendHeaders(ctx context.Context, headers api.HeaderMap, endStream bool) error {
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
	api.ReadFilterCallbacks
}

func (cb *mockReadFilterCallbacks) Connection() api.Connection {
	return &mockConnection{}
}

type mockConnection struct {
	api.Connection
}

func (c *mockConnection) ID() uint64 {
	return 0
}

func (c *mockConnection) LocalAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1")
	return addr
}

func (c *mockConnection) RemoteAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.2")
	return addr
}

type mockTracer struct {
}

func (tracer *mockTracer) Start(ctx context.Context, request interface{}, startTime time.Time) api.Span {
	return &mockSpan{}
}

type mockSpan struct {
	inject   bool
	finished bool
}

func (s *mockSpan) TraceId() string {
	return ""
}

func (s *mockSpan) SpanId() string {
	return ""
}

func (s *mockSpan) ParentSpanId() string {
	return ""
}

func (s *mockSpan) SetOperation(operation string) {
}

func (s *mockSpan) SetTag(key uint64, value string) {
}

// TODO: can be extend
func (s *mockSpan) SetRequestInfo(reqinfo types.RequestInfo) {
}

func (s *mockSpan) Tag(key uint64) string {
	return ""
}

func (s *mockSpan) FinishSpan() {
	s.finished = true
}

func (s *mockSpan) InjectContext(requestHeaders types.HeaderMap, requestInfo types.RequestInfo) {
	requestHeaders.Set("test-inject", "mock")
	s.inject = true
}

func (s *mockSpan) SpawnChild(operationName string, startTime time.Time) api.Span {
	return nil
}

type mockServerConn struct {
	types.ServerStreamConnection
}

func (s *mockServerConn) Protocol() api.ProtocolName {
	return "mockProtocol"
}

func (s *mockServerConn) EnableWorkerPool() bool {
	return true
}
