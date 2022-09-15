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

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

func init() {
	initWorkerPool(nil, false)
}

// StreamReceiverFilter
// MOSN receive a request, run StreamReceiverFilters, and send request to upstream
func TestRunReiverFilters(t *testing.T) {
	testCases := []struct {
		filters []*mockStreamReceiverFilter
	}{
		{
			filters: []*mockStreamReceiverFilter{
				// this filter returns all continue, like mixer filter or fault inject filter not matched condition
				{
					status: api.StreamFilterContinue,
					phase:  api.BeforeRoute,
				},
				// this filter like fault inject filter matched condition
				// in fault inject, it will call ContinueReceiving/SendHijackReply
				// this test will ignore it
				{
					status: api.StreamFilterStop,
					phase:  api.BeforeRoute,
				},
			},
		},

		{
			filters: []*mockStreamReceiverFilter{
				{
					status: api.StreamFilterContinue,
					phase:  api.BeforeRoute,
				},
				{
					status: api.StreamFilterReMatchRoute,
					phase:  api.AfterRoute,
				},
				// to prevent proxy. if a real stream filter returns all stop,
				// it should call SendHijackReply, or the stream will be hung up
				// this test will ignore it
				{
					status: api.StreamFilterStop,
					phase:  api.AfterRoute,
				},
			},
		},
		{
			filters: []*mockStreamReceiverFilter{
				{
					status: api.StreamFilterReMatchRoute,
					phase:  api.AfterRoute,
				},
				{
					status: api.StreamFilterStop,
					phase:  api.AfterRoute,
				},
			},
		},
	}
	for i, tc := range testCases {
		s := &downStream{
			context: variable.NewVariableContext(context.Background()),
			proxy: &proxy{
				config:              &v2.Proxy{},
				routersWrapper:      &mockRouterWrapper{},
				clusterManager:      &mockClusterManager{},
				serverStreamConn:    &mockServerConn{},
				routeHandlerFactory: router.DefaultMakeHandler,
			},
			requestInfo: &network.RequestInfo{},
			notify:      make(chan struct{}, 1),
		}
		s.initStreamFilterChain()
		for _, f := range tc.filters {
			f.s = s
			s.streamFilterChain.AddStreamReceiverFilter(f, f.phase)
		}
		// mock run
		s.downstreamReqHeaders = protocol.CommonHeader{}
		s.downstreamReqDataBuf = buffer.NewIoBuffer(0)
		s.downstreamReqTrailers = protocol.CommonHeader{}
		s.OnReceive(s.context, s.downstreamReqHeaders, s.downstreamReqDataBuf, s.downstreamReqTrailers)

		time.Sleep(100 * time.Millisecond)

		for j, f := range tc.filters {
			if f.on != 1 {
				t.Errorf("#%d.%d stream filter is not called; On:%d", i, j, f.on)
			}
		}
	}
}

func TestRunReiverFiltersStop(t *testing.T) {
	tc := struct {
		filters []*mockStreamReceiverFilter
	}{
		filters: []*mockStreamReceiverFilter{
			{
				status: api.StreamFilterReMatchRoute,
				phase:  api.AfterRoute,
			},
			{
				status: api.StreamFilterStop,
				phase:  api.AfterRoute,
			},
			{
				status: api.StreamFilterContinue,
				phase:  api.AfterRoute,
			},
		},
	}
	s := &downStream{
		context: variable.NewVariableContext(context.Background()),
		proxy: &proxy{
			config:              &v2.Proxy{},
			routersWrapper:      &mockRouterWrapper{},
			clusterManager:      &mockClusterManager{},
			serverStreamConn:    &mockServerConn{},
			routeHandlerFactory: router.DefaultMakeHandler,
		},
		requestInfo: &network.RequestInfo{},
		notify:      make(chan struct{}, 1),
	}
	s.initStreamFilterChain()
	for _, f := range tc.filters {
		f.s = s
		s.streamFilterChain.AddStreamReceiverFilter(f, f.phase)
	}
	// mock run
	s.downstreamReqHeaders = protocol.CommonHeader{}
	s.downstreamReqDataBuf = buffer.NewIoBuffer(0)
	s.downstreamReqTrailers = protocol.CommonHeader{}
	s.OnReceive(s.context, s.downstreamReqHeaders, s.downstreamReqDataBuf, s.downstreamReqTrailers)

	time.Sleep(100 * time.Millisecond)

	if tc.filters[0].on != 1 || tc.filters[1].on != 1 || tc.filters[2].on != 0 {
		t.Errorf("streamReceiveFilter is error")
	}
}

func TestRunReiverFiltersTermination(t *testing.T) {
	tc := struct {
		filters []*mockStreamReceiverFilter
	}{
		filters: []*mockStreamReceiverFilter{
			{
				status: api.StreamFilterContinue,
				phase:  api.AfterRoute,
			},
			{
				status: api.StreamFiltertermination,
				phase:  api.AfterRoute,
			},
			{
				status: api.StreamFilterContinue,
				phase:  api.AfterRoute,
			},
		},
	}
	s := &downStream{
		context: variable.NewVariableContext(context.Background()),
		proxy: &proxy{
			config: &v2.Proxy{},
			routersWrapper: &mockRouterWrapper{
				routers: &mockRouters{
					route: &mockRoute{},
				},
			},
			clusterManager:      &mockClusterManager{},
			readCallbacks:       &mockReadFilterCallbacks{},
			stats:               globalStats,
			listenerStats:       newListenerStats("test"),
			serverStreamConn:    &mockServerConn{},
			routeHandlerFactory: router.DefaultMakeHandler,
		},
		responseSender: &mockResponseSender{},
		requestInfo:    &network.RequestInfo{},
		snapshot:       &mockClusterSnapshot{},
	}
	s.initStreamFilterChain()
	for _, f := range tc.filters {
		f.s = s
		s.streamFilterChain.AddStreamReceiverFilter(f, f.phase)
	}
	// mock run
	s.downstreamReqHeaders = protocol.CommonHeader{}
	s.downstreamReqDataBuf = buffer.NewIoBuffer(0)
	s.downstreamReqTrailers = protocol.CommonHeader{}
	s.OnReceive(s.context, s.downstreamReqHeaders, s.downstreamReqDataBuf, s.downstreamReqTrailers)

	time.Sleep(100 * time.Millisecond)

	if tc.filters[0].on != 1 || tc.filters[1].on != 1 || tc.filters[2].on != 0 {
		t.Errorf("streamReceiveFilter termination is error")
	}

	if s.downstreamCleaned != 1 {
		t.Errorf("streamReceiveFilter termination is error")
	}
}

func TestRunReiverFilterHandler(t *testing.T) {
	testCases := []struct {
		filters []*mockStreamReceiverFilter
	}{
		{
			filters: []*mockStreamReceiverFilter{
				{
					status: api.StreamFilterContinue,
					phase:  api.BeforeRoute,
				},
				{
					status: api.StreamFilterStop,
					phase:  api.BeforeRoute,
				},
			},
		},
		{
			filters: []*mockStreamReceiverFilter{
				{
					status: api.StreamFilterReMatchRoute,
					phase:  api.AfterRoute,
				},
				{
					status: api.StreamFilterStop,
					phase:  api.AfterRoute,
				},
			},
		},
	}
	for i, tc := range testCases {
		s := &downStream{
			proxy: &proxy{
				config:              &v2.Proxy{},
				routersWrapper:      &mockRouterWrapper{},
				clusterManager:      &mockClusterManager{},
				serverStreamConn:    &mockServerConn{},
				routeHandlerFactory: router.DefaultMakeHandler,
			},
			requestInfo: &network.RequestInfo{},
			notify:      make(chan struct{}, 1),
		}
		s.initStreamFilterChain()

		s.context = variable.NewVariableContext(context.Background())
		for _, f := range tc.filters {
			f.s = s
			s.streamFilterChain.AddStreamReceiverFilter(f, f.phase)
		}
		// mock run
		s.downstreamReqHeaders = protocol.CommonHeader{}
		s.downstreamReqDataBuf = buffer.NewIoBuffer(0)
		s.downstreamReqTrailers = protocol.CommonHeader{}
		s.OnReceive(s.context, s.downstreamReqHeaders, s.downstreamReqDataBuf, s.downstreamReqTrailers)

		time.Sleep(100 * time.Millisecond)

		for j, f := range tc.filters {
			if f.currentPhase != f.phase {
				t.Errorf("#%d.%d stream filter phase want: %d but got: %d", i, j, f.phase, f.currentPhase)
			}
		}
	}
}

// StreamSenderFilter
// MOSN receive the upstream response, run StreamSenderFilters, and send repsonse to downstream

// simple test, no real sender filter now
func TestRunSenderFilters(t *testing.T) {
	testCases := []struct {
		filters []*mockStreamSenderFilter
	}{
		{
			filters: []*mockStreamSenderFilter{
				{
					status: api.StreamFilterContinue,
				},
				{
					status: api.StreamFilterStop,
				},
			},
		},
		{
			filters: []*mockStreamSenderFilter{
				{
					status: api.StreamFilterContinue,
				},
				{
					status: api.StreamFilterContinue,
				},
				{
					status: api.StreamFilterStop,
				},
			},
		},
	}
	for i, tc := range testCases {
		s := &downStream{
			proxy: &proxy{
				config:              &v2.Proxy{},
				routersWrapper:      &mockRouterWrapper{},
				clusterManager:      &mockClusterManager{},
				routeHandlerFactory: router.DefaultMakeHandler,
			},
		}
		s.initStreamFilterChain()
		for _, f := range tc.filters {
			f.s = s
			s.streamFilterChain.AddStreamSenderFilter(f, api.BeforeSend)
		}
		// mock run
		s.downstreamRespDataBuf = buffer.NewIoBuffer(0)
		s.downstreamRespTrailers = protocol.CommonHeader{}

		s.streamFilterChain.RunSenderFilter(context.TODO(), api.BeforeSend, nil, s.downstreamRespDataBuf, s.downstreamReqTrailers, nil)
		for j, f := range tc.filters {
			if f.on != 1 {
				t.Errorf("#%d.%d stream filter is not called; On:%d", i, j, f.on)
			}
		}
	}
}

func TestRunSenderFiltersStop(t *testing.T) {
	tc := struct {
		filters []*mockStreamSenderFilter
	}{
		filters: []*mockStreamSenderFilter{
			{
				status: api.StreamFilterContinue,
			},
			{
				status: api.StreamFilterStop,
			},
			{
				status: api.StreamFilterContinue,
			},
		},
	}
	s := &downStream{
		proxy: &proxy{
			config:              &v2.Proxy{},
			routersWrapper:      &mockRouterWrapper{},
			clusterManager:      &mockClusterManager{},
			routeHandlerFactory: router.DefaultMakeHandler,
		},
	}
	s.initStreamFilterChain()
	for _, f := range tc.filters {
		f.s = s
		s.streamFilterChain.AddStreamSenderFilter(f, api.BeforeSend)
	}

	s.streamFilterChain.RunSenderFilter(context.TODO(), api.BeforeSend, nil, nil, nil, nil)
	if s.downstreamRespHeaders == nil || s.downstreamRespDataBuf == nil {
		t.Errorf("streamSendFilter SetResponse error")
	}

	if tc.filters[0].on != 1 || tc.filters[1].on != 1 || tc.filters[2].on != 0 {
		t.Errorf("streamSendFilter is error")
	}
}

func TestRunSenderFiltersTermination(t *testing.T) {
	tc := struct {
		filters []*mockStreamSenderFilter
	}{
		filters: []*mockStreamSenderFilter{
			{
				status: api.StreamFilterContinue,
			},
			{
				status: api.StreamFiltertermination,
			},
			{
				status: api.StreamFilterContinue,
			},
		},
	}
	s := &downStream{
		context: variable.NewVariableContext(context.Background()),
		proxy: &proxy{
			config: &v2.Proxy{},
			routersWrapper: &mockRouterWrapper{
				routers: &mockRouters{
					route: &mockRoute{},
				},
			},
			clusterManager:   &mockClusterManager{},
			readCallbacks:    &mockReadFilterCallbacks{},
			stats:            globalStats,
			listenerStats:    newListenerStats("test"),
			serverStreamConn: &mockServerConn{},
		},
		responseSender: &mockResponseSender{},
		requestInfo:    &network.RequestInfo{},
		snapshot:       &mockClusterSnapshot{},
	}
	s.initStreamFilterChain()
	for _, f := range tc.filters {
		f.s = s
		s.streamFilterChain.AddStreamSenderFilter(f, api.BeforeSend)
	}

	s.streamFilterChain.RunSenderFilter(context.TODO(), api.BeforeSend, nil, nil, nil, nil)
	if s.downstreamRespHeaders == nil || s.downstreamRespDataBuf == nil {
		t.Errorf("streamSendFilter SetResponse error")
	}

	if tc.filters[0].on != 1 || tc.filters[1].on != 1 || tc.filters[2].on != 0 {
		t.Errorf("streamSendFilter is error")
	}

	if s.downstreamCleaned != 1 {
		t.Errorf("streamSendFilter termination is error")
	}
}

// Mock stream filters
type mockStreamReceiverFilter struct {
	handler api.StreamReceiverFilterHandler
	// api called count
	on int
	// current phase
	currentPhase api.ReceiverFilterPhase
	// returns status
	status api.StreamFilterStatus
	// mock for test
	phase api.ReceiverFilterPhase
	s     *downStream
}

func (f *mockStreamReceiverFilter) OnDestroy() {}

func (f *mockStreamReceiverFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {
	f.on++
	f.currentPhase = f.handler.GetFilterCurrentPhase()
	if f.status == api.StreamFilterStop || f.status == api.StreamFiltertermination {
		atomic.StoreUint32(&f.s.downstreamCleaned, 1)
	}
	if f.status == api.StreamFilterReMatchRoute || f.status == api.StreamFilterReChooseHost {
		return api.StreamFilterContinue
	} else {
		return f.status
	}
}

func (f *mockStreamReceiverFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

type mockStreamSenderFilter struct {
	handler api.StreamSenderFilterHandler
	// api called count
	on int
	// returns status
	status api.StreamFilterStatus
	// mock for test
	s *downStream
}

func (f *mockStreamSenderFilter) OnDestroy() {}

func (f *mockStreamSenderFilter) Append(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {
	f.on++
	f.handler.SetResponseHeaders(protocol.CommonHeader{})
	f.handler.SetResponseData(buffer.NewIoBuffer(1))
	if f.status == api.StreamFilterStop || f.status == api.StreamFiltertermination {
		atomic.StoreUint32(&f.s.downstreamCleaned, 1)
	}
	return f.status
}

func (f *mockStreamSenderFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.handler = handler
}
