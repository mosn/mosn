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
	"sync/atomic"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/golang/mock/gomock"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/buffer"
)

// New Test Case
// mock a request stream through proxy

// TestProxyWithFilters
// A request passed by proxy and send a response
// Start At NewStreamDetect and OnReceive (see stream/xprotocol/conn.go: handleRequest)
// 1. stream filter: BeforeRoute, add a key for macth route
// 2. stream filter: AfterRoute, record sth after route
// 3. stream filter: AfterChooseHost, record sth after choose host
// 4. route match success
// 5. receive a response
// 6. stream filter: record response
// 7. send response to client
// 8. records: tracelog/metrics
func TestProxyWithFilters(t *testing.T) {
	// prepare
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer monkey.UnpatchAll()

	// mock context from connection
	ctx := context.Background()
	ctx = mosnctx.WithValue(ctx, types.ContextKeyAccessLogs, []api.AccessLog{})
	ctx = mosnctx.WithValue(ctx, types.ContextKeyListenerName, "test_listener")

	// mock cluster manager
	monkey.Patch(cluster.GetClusterMngAdapterInstance, func() *cluster.MngAdapter {
		cm := mock.NewMockClusterManager(ctrl)
		// mock cluster manager that contains a cluster: mock_cluster
		cm.EXPECT().GetClusterSnapshot(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, clusterName string) types.ClusterSnapshot {
			if clusterName == "mock_cluster" {
				snap := mock.NewMockClusterSnapshot(ctrl)
				// mock snapshot
				snap.EXPECT().ClusterInfo().DoAndReturn(func() types.ClusterInfo {
					return gomockClusterInfo(ctrl)
				}).AnyTimes()
				return snap
			}
			return nil
		}).AnyTimes() // gomcok and monkey patch is conflict, ignore the call times
		// mock ConnPoolForCluster returns connPool
		cm.EXPECT().ConnPoolForCluster(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ types.LoadBalancerContext, _ types.ClusterSnapshot, _ api.Protocol) types.ConnectionPool {
				pool := mock.NewMockConnectionPool(ctrl)
				// mock connPool.NewStream to call upstreamRequest.OnReady (see stream/xprotocol/connpool.go:NewStream)
				pool.EXPECT().Host().DoAndReturn(func() types.Host {
					h := mock.NewMockHost(ctrl)
					h.EXPECT().HostStats().DoAndReturn(func() *types.HostStats {
						s := metrics.NewHostStats("mockhost", "mockhost")
						return &types.HostStats{
							UpstreamRequestDuration:      s.Histogram(metrics.UpstreamRequestDuration),
							UpstreamRequestDurationTotal: s.Counter(metrics.UpstreamRequestDurationTotal),
						}
					}).AnyTimes()
					h.EXPECT().AddressString().Return("").AnyTimes()
					h.EXPECT().ClusterInfo().DoAndReturn(func() types.ClusterInfo {
						return gomockClusterInfo(ctrl)
					})
					return h
				}).AnyTimes()
				pool.EXPECT().NewStream(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ types.StreamReceiveListener, l types.PoolEventListener) {
					encoder := mock.NewMockStreamSender(ctrl)
					encoder.EXPECT().GetStream().DoAndReturn(func() types.Stream {
						s := mock.NewMockStream(ctrl)
						s.EXPECT().AddEventListener(gomock.Any())
						return s
					})
					encoder.EXPECT().AppendHeaders(gomock.Any(), gomock.Any(), gomock.Any())
					encoder.EXPECT().AppendData(gomock.Any(), gomock.Any(), gomock.Any())
					encoder.EXPECT().AppendTrailers(gomock.Any(), gomock.Any())
					l.OnReady(encoder, pool.Host())
				})
				return pool
			}).AnyTimes()
		return &cluster.MngAdapter{
			ClusterManager: cm,
		}
	})

	// mock routers
	monkey.Patch(router.GetRoutersMangerInstance, func() types.RouterManager {
		rm := mock.NewMockRouterManager(ctrl)
		rm.EXPECT().GetRouterWrapperByName("test_router").DoAndReturn(func(tn string) types.RouterWrapper {
			rw := mock.NewMockRouterWrapper(ctrl)
			rw.EXPECT().GetRouters().DoAndReturn(func() types.Routers {
				r := mock.NewMockRouters(ctrl)
				// mock routers can be matched route if a header contains key service and values equals config name.
				r.EXPECT().MatchRoute(gomock.Any(), gomock.Any()).DoAndReturn(func(headers api.HeaderMap, _ uint64) api.Route {
					if sn, ok := headers.Get("service"); ok && sn == tn {
						r := mock.NewMockRoute(ctrl)
						// mock route rule returns cluster name : mock_cluster
						r.EXPECT().RouteRule().DoAndReturn(func() api.RouteRule {
							rule := mock.NewMockRouteRule(ctrl)
							rule.EXPECT().ClusterName().Return("mock_cluster").AnyTimes()
							rule.EXPECT().UpstreamProtocol().Return("").AnyTimes()
							rule.EXPECT().GlobalTimeout().Return(3 * time.Second).AnyTimes()
							rule.EXPECT().Policy().DoAndReturn(func() api.Policy {
								p := mock.NewMockPolicy(ctrl)
								p.EXPECT().RetryPolicy().DoAndReturn(func() api.RetryPolicy {
									rp := mock.NewMockRetryPolicy(ctrl)
									rp.EXPECT().RetryOn().Return(false).AnyTimes()
									rp.EXPECT().TryTimeout().Return(time.Duration(0))
									rp.EXPECT().NumRetries().Return(uint32(3)).AnyTimes()
									return rp
								}).AnyTimes()
								return p
							}).AnyTimes()
							rule.EXPECT().FinalizeRequestHeaders(gomock.Any(), gomock.Any()).AnyTimes()
							return rule
						}).AnyTimes()
						r.EXPECT().DirectResponseRule().Return(nil)
						r.EXPECT().RedirectRule().Return(nil)
						return r
					}
					return nil
				}).AnyTimes()
				return r
			}).AnyTimes()
			return rw
		})
		return rm
	})

	// mock stream filters
	var factories []api.StreamFilterChainFactory
	for _, ff := range []struct {
		Phase            api.FilterPhase
		ReceiveFilterGen func() api.StreamReceiverFilter
		SenderFilterGen  func() api.StreamSenderFilter
	}{} {
		factory := mock.NewMockStreamFilterChainFactory(ctrl)
		factory.EXPECT().CreateFilterChain(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, cb api.StreamFilterChainFactoryCallbacks) {
			if ff.ReceiveFilterGen != nil {
				cb.AddStreamReceiverFilter(ff.ReceiveFilterGen(), ff.Phase)
			}
			if ff.SenderFilterGen != nil {
				cb.AddStreamSenderFilter(ff.SenderFilterGen())
			}
		})
		factories = append(factories, factory)
	}
	value := new(atomic.Value)
	value.Store(factories)
	// mock stream connection
	monkey.Patch(stream.CreateServerStreamConnection,
		func(ctx context.Context, _ types.ProtocolName, _ api.Connection, proxy types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
			ctx = mosnctx.WithValue(ctx, types.ContextKeyStreamFilterChainFactories, value)
			sconn := mock.NewMockServerStreamConnection(ctrl)
			sconn.EXPECT().Protocol().Return(types.ProtocolName("Http1")).AnyTimes()
			return sconn
		})

	pv := NewProxy(ctx, &v2.Proxy{
		Name:               "test",
		DownstreamProtocol: "Http1",
		UpstreamProtocol:   "Http1",
		RouterConfigName:   "test_router",
	})
	proxy := pv.(*proxy)
	// mock a span
	monkey.Patch(trace.IsEnabled, func() bool {
		return true
	})
	mockSpan := func() types.Span {
		sp := mock.NewMockSpan(ctrl)
		sp.EXPECT().TraceId().Return("1").AnyTimes()
		sp.EXPECT().SpanId().Return("1").AnyTimes()
		return sp
	}
	var sender types.StreamSender
	// mock connection and receive
	cb := mock.NewMockReadFilterCallbacks(ctrl)
	cb.EXPECT().Connection().DoAndReturn(func() api.Connection {
		conn := mock.NewMockConnection(ctrl)
		conn.EXPECT().SetCollector(gomock.Any(), gomock.Any()).AnyTimes()
		conn.EXPECT().AddConnectionEventListener(gomock.Any()).AnyTimes()
		conn.EXPECT().LocalAddr().Return(nil).AnyTimes()  // mock, no use
		conn.EXPECT().RemoteAddr().Return(nil).AnyTimes() // mock, no use
		return conn
	}).AnyTimes()
	pv.InitializeReadFilterCallbacks(cb)
	ss := proxy.NewStreamDetect(ctx, sender, mockSpan())
	downstream := ss.(*downStream)
	headers := mock.NewMockHeaderMap(ctrl)
	headers.EXPECT().Get("service").Return("test_router", true).AnyTimes()
	headers.EXPECT().Get(types.HeaderTryTimeout).Return("", false).AnyTimes()
	headers.EXPECT().Get(types.HeaderGlobalTimeout).Return("", false).AnyTimes()
	trailer := mock.NewMockHeaderMap(ctrl)
	downstream.OnReceive(ctx, headers, buffer.NewIoBuffer(0), trailer)
	// mock wait response
	// upstreamRequest.OnReceive ( see stream/xprotocol/conn.go: handleResponse)
	// upstream.OnReceive()

}
