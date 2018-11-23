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
	"time"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol"
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

type mockRouteRule struct {
	types.RouteRule
}

func (r *mockRouteRule) ClusterName() string {
	return "test"
}
func (r *mockRouteRule) Policy() types.Policy {
	return &mockRoutePolicy{}
}
func (r *mockRouteRule) GlobalTimeout() time.Duration {
	return time.Minute
}

type mockRoutePolicy struct {
	types.Policy
}

func (p *mockRoutePolicy) ShadowPolicy() types.ShadowPolicy {
	return &mockShadowPolicy{}
}

func (p *mockRoutePolicy) RetryPolicy() types.RetryPolicy {
	return &mockRetryPolicy{}
}

type mockShadowPolicy struct {
	types.ShadowPolicy
}

func (p *mockShadowPolicy) ClusterName() string {
	return "shadow"
}

type mockRetryPolicy struct {
	types.RetryPolicy
}

func (p *mockRetryPolicy) TryTimeout() time.Duration {
	return time.Minute
}

type mockClusterManager struct {
	types.ClusterManager
}

func (m *mockClusterManager) GetClusterSnapshot(ctx context.Context, name string) types.ClusterSnapshot {
	return &mockClusterSnapshot{}
}

func (m *mockClusterManager) PutClusterSnapshot(snapshot types.ClusterSnapshot) {
}

func (m *mockClusterManager) ConnPoolForCluster(ctx types.LoadBalancerContext, snapshot types.ClusterSnapshot, p types.Protocol) types.ConnectionPool {
	return &mockConnPool{}
}

type mockClusterSnapshot struct {
	types.ClusterSnapshot
}

func (s *mockClusterSnapshot) ClusterInfo() types.ClusterInfo {
	return nil
}

type mockConnPool struct {
	types.ConnectionPool
}

func (pool *mockConnPool) NewStream(context context.Context, streamID string,
	responseDecoder types.StreamReceiver, cb types.PoolEventListener) types.Cancellable {
	sender := &mockStreamSender{
		receiver: responseDecoder,
	}
	cb.OnReady(streamID, sender, nil)
	return nil
}

type mockStreamSender struct {
	receiver types.StreamReceiver
}

// mock a send and wait response
func (s *mockStreamSender) sendAndresponse() {
	ctx := context.Background()
	s.receiver.OnReceiveHeaders(ctx, protocol.CommonHeader{}, false)
	s.receiver.OnReceiveData(ctx, buffer.NewIoBuffer(0), false)
	s.receiver.OnReceiveTrailers(ctx, protocol.CommonHeader{})
}

func (s *mockStreamSender) AppendHeaders(ctx context.Context, headers types.HeaderMap, endStream bool) error {
	if endStream {
		s.sendAndresponse()
	}
	return nil
}

func (s *mockStreamSender) AppendData(ctx context.Context, data types.IoBuffer, endStream bool) error {
	if endStream {
		s.sendAndresponse()
	}
	return nil
}

func (s *mockStreamSender) AppendTrailers(ctx context.Context, trailers types.HeaderMap) error {
	s.sendAndresponse()
	return nil
}

func (s *mockStreamSender) GetStream() types.Stream {
	return &mockStream{}
}

type mockStream struct {
	types.Stream
}

func (s *mockStream) AddEventListener(l types.StreamEventListener) {
}

func (s *mockStream) RemoveEventListener(l types.StreamEventListener) {
}

func (s *mockStream) ResetStream(r types.StreamResetReason) {
}
