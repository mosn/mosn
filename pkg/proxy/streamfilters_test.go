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
	"testing"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
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
					headersStatus:  types.StreamHeadersFilterContinue,
					dataStatus:     types.StreamDataFilterContinue,
					trailersStatus: types.StreamTrailersFilterContinue,
				},
				// this filter like fault inject filter matched condition
				// in fault inject, it will call ContinueReceiving/SendHijackReply
				// this test will ignore it
				{
					headersStatus:  types.StreamHeadersFilterStop,
					dataStatus:     types.StreamDataFilterStopAndBuffer,
					trailersStatus: types.StreamTrailersFilterStop,
				},
			},
		},
		// The Header filter returns stop to run next filter,
		// but the data/trailer filter wants to be continue
		{
			filters: []*mockStreamReceiverFilter{
				{
					headersStatus:  types.StreamHeadersFilterStop,
					dataStatus:     types.StreamDataFilterContinue,
					trailersStatus: types.StreamTrailersFilterContinue,
				},
				{
					headersStatus:  types.StreamHeadersFilterContinue,
					dataStatus:     types.StreamDataFilterContinue,
					trailersStatus: types.StreamTrailersFilterContinue,
				},
				// to prevent proxy. if a real stream filter returns all stop,
				// it should call ContinueReceiving or SendHijackReply, or the stream will be hung up
				// this test will ignore it
				{
					headersStatus:  types.StreamHeadersFilterStop,
					dataStatus:     types.StreamDataFilterStop,
					trailersStatus: types.StreamTrailersFilterStop,
				},
			},
		},
		{
			filters: []*mockStreamReceiverFilter{
				{
					headersStatus:  types.StreamHeadersFilterStop,
					dataStatus:     types.StreamDataFilterStop,
					trailersStatus: types.StreamTrailersFilterContinue,
				},
				{
					headersStatus:  types.StreamHeadersFilterContinue,
					dataStatus:     types.StreamDataFilterContinue,
					trailersStatus: types.StreamTrailersFilterContinue,
				},
				// to prevent proxy. if a real stream filter returns all stop,
				// it should call ContinueReceiving or SendHijackReply, or the stream will be hung up
				// this test will ignore it
				{
					headersStatus:  types.StreamHeadersFilterStop,
					dataStatus:     types.StreamDataFilterStop,
					trailersStatus: types.StreamTrailersFilterStop,
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
			s.AddStreamReceiverFilter(f)
		}
		// mock run
		s.doReceiveHeaders(nil, nil, false)
		// to continue data
		s.downstreamReqDataBuf = buffer.NewIoBuffer(0)
		s.doReceiveData(nil, s.downstreamReqDataBuf, false)
		// to continue trailer
		s.downstreamReqTrailers = protocol.CommonHeader{}
		s.doReceiveTrailers(nil, s.downstreamReqTrailers)
		for j, f := range tc.filters {
			if !(f.onHeaders == 1 && f.onData == 1 && f.onTrailers == 1) {
				t.Errorf("#%d.%d stream filter is not called; OnHeader:%d, OnData:%d, OnTrailer:%d", i, j, f.onHeaders, f.onData, f.onTrailers)
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
					headersStatus:  types.StreamHeadersFilterContinue,
					dataStatus:     types.StreamDataFilterContinue,
					trailersStatus: types.StreamTrailersFilterContinue,
				},
				{
					headersStatus:  types.StreamHeadersFilterStop,
					dataStatus:     types.StreamDataFilterStopAndBuffer,
					trailersStatus: types.StreamTrailersFilterStop,
				},
			},
		},
		{
			filters: []*mockStreamSenderFilter{
				{
					headersStatus:  types.StreamHeadersFilterStop,
					dataStatus:     types.StreamDataFilterContinue,
					trailersStatus: types.StreamTrailersFilterContinue,
				},
				{
					headersStatus:  types.StreamHeadersFilterContinue,
					dataStatus:     types.StreamDataFilterContinue,
					trailersStatus: types.StreamTrailersFilterContinue,
				},
				{
					headersStatus:  types.StreamHeadersFilterStop,
					dataStatus:     types.StreamDataFilterStop,
					trailersStatus: types.StreamTrailersFilterStop,
				},
			},
		},
		{
			filters: []*mockStreamSenderFilter{
				{
					headersStatus:  types.StreamHeadersFilterStop,
					dataStatus:     types.StreamDataFilterStop,
					trailersStatus: types.StreamTrailersFilterContinue,
				},
				{
					headersStatus:  types.StreamHeadersFilterContinue,
					dataStatus:     types.StreamDataFilterContinue,
					trailersStatus: types.StreamTrailersFilterContinue,
				},
				{
					headersStatus:  types.StreamHeadersFilterStop,
					dataStatus:     types.StreamDataFilterStop,
					trailersStatus: types.StreamTrailersFilterStop,
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
		s.doAppendHeaders(nil, nil, false)
		// to continue data
		s.downstreamRespDataBuf = buffer.NewIoBuffer(0)
		s.doAppendData(nil, s.downstreamRespDataBuf, false)
		// to continue trailer
		s.downstreamRespTrailers = protocol.CommonHeader{}
		s.doAppendTrailers(nil, s.downstreamRespTrailers)
		for j, f := range tc.filters {
			if !(f.onHeaders == 1 && f.onData == 1 && f.onTrailers == 1) {
				t.Errorf("#%d.%d stream filter is not called; OnHeader:%d, OnData:%d, OnTrailer:%d", i, j, f.onHeaders, f.onData, f.onTrailers)
			}
		}
	}

}

// Mock stream filters
type mockStreamReceiverFilter struct {
	handler types.StreamReceiverFilterHandler
	// api called count
	onHeaders  int
	onData     int
	onTrailers int
	// returns status
	headersStatus  types.StreamHeadersFilterStatus
	dataStatus     types.StreamDataFilterStatus
	trailersStatus types.StreamTrailersFilterStatus
}

func (f *mockStreamReceiverFilter) OnDestroy() {}

func (f *mockStreamReceiverFilter) OnReceiveHeaders(headers types.HeaderMap, endStream bool) types.StreamHeadersFilterStatus {
	f.onHeaders++
	return f.headersStatus
}

func (f *mockStreamReceiverFilter) OnReceiveData(buf types.IoBuffer, endStream bool) types.StreamDataFilterStatus {
	f.onData++
	return f.dataStatus
}

func (f *mockStreamReceiverFilter) OnReceiveTrailers(trailers types.HeaderMap) types.StreamTrailersFilterStatus {
	f.onTrailers++
	return f.trailersStatus
}

func (f *mockStreamReceiverFilter) SetReceiveFilterHandler(handler types.StreamReceiverFilterHandler) {
	f.handler = handler
}

type mockStreamSenderFilter struct {
	handler types.StreamSenderFilterHandler
	// api called count
	onHeaders  int
	onData     int
	onTrailers int
	// returns status
	headersStatus  types.StreamHeadersFilterStatus
	dataStatus     types.StreamDataFilterStatus
	trailersStatus types.StreamTrailersFilterStatus
}

func (f *mockStreamSenderFilter) OnDestroy() {}

func (f *mockStreamSenderFilter) AppendHeaders(headers types.HeaderMap, endStream bool) types.StreamHeadersFilterStatus {
	f.onHeaders++
	return f.headersStatus
}

func (f *mockStreamSenderFilter) AppendData(buf types.IoBuffer, endStream bool) types.StreamDataFilterStatus {
	f.onData++
	return f.dataStatus
}

func (f *mockStreamSenderFilter) AppendTrailers(trailers types.HeaderMap) types.StreamTrailersFilterStatus {
	f.onTrailers++
	return f.trailersStatus
}

func (f *mockStreamSenderFilter) SetSenderFilterHandler(handler types.StreamSenderFilterHandler) {
	f.handler = handler
}
