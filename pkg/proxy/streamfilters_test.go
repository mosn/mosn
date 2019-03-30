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
	"testing"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
	"time"
)

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
					status: types.StreamFilterContinue,
				},
				// this filter like fault inject filter matched condition
				// in fault inject, it will call ContinueReceiving/SendHijackReply
				// this test will ignore it
				{
					status: types.StreamFilterStop,
				},
			},
		},

		{
			filters: []*mockStreamReceiverFilter{
				{
					status: types.StreamFilterContinue,
				},
				{
					status: types.StreamFilterReMatchRoute,
				},
				// to prevent proxy. if a real stream filter returns all stop,
				// it should call SendHijackReply, or the stream will be hung up
				// this test will ignore it
				{
					status: types.StreamFilterStop,
				},
			},
		},
		{
			filters: []*mockStreamReceiverFilter{
				{
					status: types.StreamFilterReMatchRoute,
				},
				{
					status: types.StreamFilterStop,
				},
			},
		},
	}
	for i, tc := range testCases {
		s := &downStream{
			proxy: &proxy{
				routersWrapper: &mockRouterWrapper{},
				clusterManager: &mockClusterManager{},
			},
			logger:      log.DefaultLogger,
			requestInfo: &network.RequestInfo{},
			notify:      make(chan struct{}, 1),
		}
		for _, f := range tc.filters {
			s.AddStreamReceiverFilter(f, types.DownFilterAfterRoute)
		}
		// mock run
		s.downstreamReqHeaders = protocol.CommonHeader{}
		s.downstreamReqDataBuf = buffer.NewIoBuffer(0)
		s.downstreamReqTrailers = protocol.CommonHeader{}
		s.OnReceive(context.Background(), s.downstreamReqHeaders, s.downstreamReqDataBuf, s.downstreamReqTrailers)

		time.Sleep(100 * time.Millisecond)

		for j, f := range tc.filters {
			if f.on != 1 {
				t.Errorf("#%d.%d stream filter is not called; On:%d", i, j, f.on)
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
					status:  types.StreamFilterContinue,
				},
				{
					status:  types.StreamFilterStop,
				},
			},
		},
		{
			filters: []*mockStreamSenderFilter{
				{
					status:  types.StreamFilterContinue,
				},
				{
					status:  types.StreamFilterContinue,
				},
				{
					status:  types.StreamFilterStop,
				},
			},
		},
	}
	for i, tc := range testCases {
		s := &downStream{
			proxy: &proxy{
				routersWrapper: &mockRouterWrapper{},
				clusterManager: &mockClusterManager{},
			},
		}
		for _, f := range tc.filters {
			s.AddStreamSenderFilter(f)
		}
		// mock run
		s.downstreamRespDataBuf = buffer.NewIoBuffer(0)
		s.downstreamRespTrailers = protocol.CommonHeader{}

	    s.runAppendFilters(0, nil, s.downstreamRespDataBuf, s.downstreamReqTrailers)
		for j, f := range tc.filters {
			if f.on != 1 {
				t.Errorf("#%d.%d stream filter is not called; On:%d", i, j, f.on)
			}
		}
	}
}

// Mock stream filters
type mockStreamReceiverFilter struct {
	handler types.StreamReceiverFilterHandler
	// api called count
	on int
	// returns status
	status types.StreamFilterStatus
}

func (f *mockStreamReceiverFilter) OnDestroy() {}

func (f *mockStreamReceiverFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) types.StreamFilterStatus {
	f.on++
	return f.status
}

func (f *mockStreamReceiverFilter) SetReceiveFilterHandler(handler types.StreamReceiverFilterHandler) {
	f.handler = handler
}

type mockStreamSenderFilter struct {
	handler types.StreamSenderFilterHandler
	// api called count
	on int
	// returns status
	status types.StreamFilterStatus
}

func (f *mockStreamSenderFilter) OnDestroy() {}

func (f *mockStreamSenderFilter) Append(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) types.StreamFilterStatus {
	f.on++
	return f.status
}

func (f *mockStreamSenderFilter) SetSenderFilterHandler(handler types.StreamSenderFilterHandler) {
	f.handler = handler
}
