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
	"math"
	"testing"
	"time"

	monkey "github.com/cch123/supermonkey"
	"github.com/golang/mock/gomock"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/metrics/ewma"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
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
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableAccessLogs, []api.AccessLog{})
	_ = variable.Set(ctx, types.VariableListenerName, "test_listener")

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
			func(_ types.LoadBalancerContext, _ types.ClusterSnapshot, _ api.ProtocolName) (types.ConnectionPool, types.Host) {
				pool := mock.NewMockConnectionPool(ctrl)
				// mock connPool.NewStream to call upstreamRequest.OnReady (see stream/xprotocol/connpool.go:NewStream)
				h := mock.NewMockHost(ctrl)
				h.EXPECT().HostStats().DoAndReturn(func() *types.HostStats {
					s := metrics.NewHostStats("mockhost", "mockhost")
					return &types.HostStats{
						UpstreamRequestDuration:      s.Histogram(metrics.UpstreamRequestDuration),
						UpstreamRequestDurationTotal: s.Counter(metrics.UpstreamRequestDurationTotal),
						UpstreamRequestDurationEWMA:  s.EWMA(metrics.UpstreamRequestDurationEWMA, ewma.Alpha(math.Exp(-5), time.Second)),

						UpstreamResponseFailed:  s.Counter(metrics.UpstreamResponseFailed),
						UpstreamResponseSuccess: s.Counter(metrics.UpstreamResponseSuccess),

						UpstreamResponseTotalEWMA:       s.EWMA(metrics.UpstreamResponseTotalEWMA, ewma.Alpha(math.Exp(-5), time.Second)),
						UpstreamResponseClientErrorEWMA: s.EWMA(metrics.UpstreamResponseClientErrorEWMA, ewma.Alpha(math.Exp(-5), time.Second)),
						UpstreamResponseServerErrorEWMA: s.EWMA(metrics.UpstreamResponseServerErrorEWMA, ewma.Alpha(math.Exp(-5), time.Second)),
					}
				}).AnyTimes()
				h.EXPECT().AddressString().Return("mockhost").AnyTimes()
				h.EXPECT().ClusterInfo().DoAndReturn(func() types.ClusterInfo {
					return gomockClusterInfo(ctrl)
				}).AnyTimes()
				pool.EXPECT().Host().DoAndReturn(func() types.Host {
					return h
				}).AnyTimes()
				pool.EXPECT().NewStream(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ types.StreamReceiveListener) (types.Host, types.StreamSender, types.PoolFailureReason) {
					// upstream encoder
					encoder := gomockStreamSender(ctrl)
					return pool.Host(), encoder, ""
				}).AnyTimes()
				return pool, h
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
				r.EXPECT().MatchRoute(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, headers api.HeaderMap) api.Route {
					if sn, ok := headers.Get("service"); ok && sn == tn {
						return gomockRouteMatchCluster(ctrl, "mock_cluster")
					}
					return nil
				}).AnyTimes()
				return r
			}).AnyTimes()
			return rw
		})
		return rm
	})

	// filter call record
	filterRecords := map[string]string{}
	// mock stream filters
	monkey.Patch(streamfilter.GetStreamFilterManager, func() streamfilter.StreamFilterManager {
		factory := streamfilter.NewMockStreamFilterFactory(ctrl)
		factory.EXPECT().CreateFilterChain(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
			for _, fact := range []struct {
				Phase            api.ReceiverFilterPhase
				ReceiveFilterGen func() api.StreamReceiverFilter
				SenderFilterGen  func() api.StreamSenderFilter
			}{
				{
					Phase: api.BeforeRoute,
					ReceiveFilterGen: func() api.StreamReceiverFilter {
						return gomockReceiverFilter(ctrl, func(h api.StreamReceiverFilterHandler, _ context.Context, header api.HeaderMap, _ buffer.IoBuffer, _ api.HeaderMap) {
							header.Set("service", "test_router")
							h.SetRequestHeaders(header)
						})
					},
				},
				{
					Phase: api.AfterRoute,
					ReceiveFilterGen: func() api.StreamReceiverFilter {
						return gomockReceiverFilter(ctrl, func(h api.StreamReceiverFilterHandler, _ context.Context, header api.HeaderMap, _ buffer.IoBuffer, _ api.HeaderMap) {
							if h.GetFilterCurrentPhase() == api.AfterRoute {
								filterRecords["after_route"] = "yes"
							}
						})
					},
				},
				{
					Phase: api.AfterChooseHost,
					ReceiveFilterGen: func() api.StreamReceiverFilter {
						return gomockReceiverFilter(ctrl, func(h api.StreamReceiverFilterHandler, _ context.Context, _ api.HeaderMap, _ buffer.IoBuffer, _ api.HeaderMap) {
							if h.RequestInfo().RouteEntry() != nil {
								filterRecords["route_cluster"] = h.RequestInfo().RouteEntry().ClusterName(context.TODO())
							}
							filterRecords["host"] = h.RequestInfo().UpstreamLocalAddress()
						})
					},
				},
				{
					SenderFilterGen: func() api.StreamSenderFilter {
						filter := mock.NewMockStreamSenderFilter(ctrl)
						filter.EXPECT().OnDestroy().AnyTimes()
						filter.EXPECT().SetSenderFilterHandler(gomock.Any()).AnyTimes()
						filter.EXPECT().Append(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
							DoAndReturn(func(_ context.Context, _ api.HeaderMap, data buffer.IoBuffer, _ api.HeaderMap) api.StreamFilterStatus {
								filterRecords["response"] = data.String()
								return api.StreamFilterContinue
							})
						return filter
					},
				},
			} {
				ff := fact // do a copy
				if ff.ReceiveFilterGen != nil {
					callbacks.AddStreamReceiverFilter(ff.ReceiveFilterGen(), ff.Phase)
				}
				if ff.SenderFilterGen != nil {
					callbacks.AddStreamSenderFilter(ff.SenderFilterGen(), api.BeforeSend)
				}
			}
		})
		filterManager := streamfilter.NewMockStreamFilterManager(ctrl)
		filterManager.EXPECT().GetStreamFilterFactory(gomock.Any()).Return(factory).AnyTimes()
		return filterManager
	})
	// mock stream connection
	monkey.Patch(stream.CreateServerStreamConnection,
		func(ctx context.Context, _ types.ProtocolName, _ api.Connection, proxy types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
			sconn := mock.NewMockServerStreamConnection(ctrl)
			sconn.EXPECT().Protocol().Return(types.ProtocolName("Http1")).AnyTimes()
			sconn.EXPECT().EnableWorkerPool().Return(true).AnyTimes()
			return sconn
		})
	// mock a span
	monkey.Patch(trace.IsEnabled, func() bool {
		return true
	})
	mockSpan := func() api.Span {
		sp := mock.NewMockSpan(ctrl)
		sp.EXPECT().TraceId().Return("1").AnyTimes()
		sp.EXPECT().SpanId().Return("1").AnyTimes()
		sp.EXPECT().InjectContext(gomock.Any(), gomock.Any())
		sp.EXPECT().SetRequestInfo(gomock.Any())
		sp.EXPECT().FinishSpan()
		return sp
	}
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
	// mock response sender
	sender := gomockStreamSender(ctrl)
	// set variable for proxy parsed
	variable.Set(ctx, types.VarProtocolConfig, []api.ProtocolName{api.ProtocolName("Http1")})

	// finish mock, start test
	pv := NewProxy(ctx, &v2.Proxy{
		Name:               "test",
		DownstreamProtocol: "Http1",
		UpstreamProtocol:   "Http1",
		RouterConfigName:   "test_router",
	})
	proxy := pv.(*proxy)
	pv.InitializeReadFilterCallbacks(cb)
	ss := proxy.NewStreamDetect(ctx, sender, mockSpan())
	downstream := ss.(*downStream)
	headers := protocol.CommonHeader{}
	trailer := mock.NewMockHeaderMap(ctrl)
	downstream.OnReceive(ctx, headers, buffer.NewIoBufferString("12345"), trailer)
	// OnReceive is scheduled by goroutine pool, we needs wait a moment
	time.Sleep(2 * time.Second)
	// Verify OnReceive states
	// requestInfo should contains information:
	// 1. RouteEntry: RouteRule
	// 2. Bytes Received: data len
	// 3. Protocol: Http1
	// stream filters called:
	// 1. choose host: selected host & routerule.clustername
	// 2. after route: test records
	if !(downstream.requestInfo.RouteEntry() != nil &&
		downstream.requestInfo.BytesReceived() == uint64(5) &&
		downstream.requestInfo.Protocol() == api.ProtocolName("Http1")) {
		t.Fatalf("after send request, the request info is not expected: %v, %v, %v",
			downstream.requestInfo.RouteEntry() != nil,
			downstream.requestInfo.BytesReceived(),
			downstream.requestInfo.Protocol())
	}
	if !(filterRecords["route_cluster"] == "mock_cluster" &&
		filterRecords["host"] == "mockhost" &&
		filterRecords["after_route"] == "yes") {
		t.Fatalf("the stream filter is not called what we want: %v", filterRecords)
	}
	// mock wait response
	time.Sleep(300 * time.Millisecond)
	upstreamRequest := downstream.upstreamRequest
	// upstreamRequest.OnReceive ( see stream/xprotocol/conn.go: handleResponse)

	upstreamRequest.downStream.context = variable.NewVariableContext(upstreamRequest.downStream.context)
	variable.SetString(upstreamRequest.downStream.context, types.VarHeaderStatus, "200")

	upstreamRequest.OnReceive(ctx, protocol.CommonHeader{}, buffer.NewIoBufferString("123"), trailer)
	// wait givestream
	time.Sleep(time.Second)
	// Veirfy OnReceive response states
	// stream filter calls
	if filterRecords["response"] != "123" {
		t.Fatalf("the sender stream filter is not called what we want: %v", filterRecords)
	}
	// request info verify
	// 1. resposne size
	// 2. status code
	// 3. is request failed
	if !(downstream.requestInfo.BytesSent() == uint64(3) &&
		downstream.requestInfo.ResponseCode() == 200 &&
		!downstream.isRequestFailed()) {
		t.Fatalf("after receive response ,the request info is not expected: %v, %v, %v",
			downstream.requestInfo.BytesSent(),
			downstream.requestInfo.ResponseCode(),
			downstream.isRequestFailed())
	}
	// TODO: more verify
}
