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

package filter

import (
	"context"
	"testing"

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/pkg/buffer"
)

var destroyCount = 0
var onReceiveCount = 0
var appendCount = 0
var logCount = 0

type mockStreamFilter struct {
	appendReturn    api.StreamFilterStatus
	onReceiveReturn api.StreamFilterStatus
}

func (m *mockStreamFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	appendCount++
	return m.appendReturn
}

func (m *mockStreamFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
}

func (m *mockStreamFilter) OnDestroy() {
	destroyCount++
}

func (m *mockStreamFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	onReceiveCount++
	return m.onReceiveReturn
}

func (m *mockStreamFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
}

type mockStreamAccessLog struct{}

func (m mockStreamAccessLog) Log(ctx context.Context, reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	logCount++
}

func TestDefaultStreamFilterManagerImpl_AddStreamAccessLog(t *testing.T) {
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamAccessLog(mockStreamAccessLog{})
	d.AddStreamAccessLog(mockStreamAccessLog{})
	if len(d.streamAccessLogs) != 2 {
		t.Errorf("DefaultStreamFilterManagerImpl.AddStreamAccessLog failed, len: %v", len(d.streamAccessLogs))
	}
}

func TestDefaultStreamFilterManagerImpl_AddStreamReceiverFilter(t *testing.T) {
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamReceiverFilter(&mockStreamFilter{onReceiveReturn: api.StreamFilterContinue}, api.BeforeRoute)
	d.AddStreamReceiverFilter(&mockStreamFilter{onReceiveReturn: api.StreamFilterContinue}, api.BeforeRoute)
	if len(d.receiverFilters) != 2 {
		t.Errorf("DefaultStreamFilterManagerImpl.AddStreamReceiverFilter failed, len: %v", len(d.receiverFilters))
	}
}

func TestDefaultStreamFilterManagerImpl_AddStreamSenderFilter(t *testing.T) {
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamSenderFilter(&mockStreamFilter{appendReturn: api.StreamFilterContinue}, api.BeforeSend)
	d.AddStreamSenderFilter(&mockStreamFilter{appendReturn: api.StreamFilterContinue}, api.BeforeSend)
	if len(d.senderFilters) != 2 {
		t.Errorf("DefaultStreamFilterManagerImpl.AddStreamSenderFilter failed, len: %v", len(d.senderFilters))
	}
}

func TestDefaultStreamFilterManagerImpl_Log(t *testing.T) {
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamAccessLog(mockStreamAccessLog{})
	d.AddStreamAccessLog(mockStreamAccessLog{})
	d.Log(context.TODO(), nil, nil, nil)
	if logCount != 2 {
		t.Error("DefaultStreamFilterManagerImpl.Log failed")
	}
}

func TestDefaultStreamFilterManagerImpl_OnDestroy(t *testing.T) {
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamReceiverFilter(&mockStreamFilter{onReceiveReturn: api.StreamFilterContinue}, api.BeforeRoute)
	d.AddStreamSenderFilter(&mockStreamFilter{appendReturn: api.StreamFilterContinue}, api.BeforeSend)
	d.OnDestroy()
	if destroyCount != 2 {
		t.Errorf("DefaultStreamFilterManagerImpl.OnDestroy failed, destroyCount: %v", destroyCount)
	}
}

func TestDefaultStreamFilterManagerImpl_RunReceiverFilter(t *testing.T) {
	tests := []struct {
		receiverFilters []*mockStreamFilter
		wantStatus      api.StreamFilterStatus
		wantIndex       int
	}{
		{
			[]*mockStreamFilter{
				{onReceiveReturn: api.StreamFilterContinue},
				{onReceiveReturn: api.StreamFilterContinue},
			},
			api.StreamFilterContinue,
			0,
		},
		{
			[]*mockStreamFilter{
				{onReceiveReturn: api.StreamFilterContinue},
				{onReceiveReturn: api.StreamFilterStop},
			},
			api.StreamFilterStop,
			0,
		},
		{
			[]*mockStreamFilter{
				{onReceiveReturn: api.StreamFilterContinue},
				{onReceiveReturn: api.StreamFiltertermination},
			},
			api.StreamFiltertermination,
			0,
		},
		{
			[]*mockStreamFilter{
				{onReceiveReturn: api.StreamFilterContinue},
				{onReceiveReturn: api.StreamFilterReMatchRoute},
			},
			api.StreamFilterReMatchRoute,
			1,
		},
		{
			[]*mockStreamFilter{
				{onReceiveReturn: api.StreamFilterContinue},
				{onReceiveReturn: api.StreamFilterReChooseHost},
			},
			api.StreamFilterReChooseHost,
			1,
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			d := &DefaultStreamFilterManagerImpl{}
			for _, filter := range tt.receiverFilters {
				d.AddStreamReceiverFilter(filter, api.BeforeRoute)
			}
			gotFilterStatus := d.RunReceiverFilter(context.TODO(), api.BeforeRoute,
				protocol.CommonHeader{}, buffer.NewIoBuffer(0), protocol.CommonHeader{},
				func(status api.StreamFilterStatus) {}) // do nothing for coverage
			if gotFilterStatus != tt.wantStatus || d.receiverFiltersIndex != tt.wantIndex {
				t.Errorf("RunReceiverFilter() = %v, want %v; index = %v, want %v",
					gotFilterStatus, tt.wantStatus, d.receiverFiltersIndex, tt.wantIndex)
			}
		})
	}
}

func TestDefaultStreamFilterManagerImpl_RunSenderFilter(t *testing.T) {
	tests := []struct {
		senderFilters []*mockStreamFilter
		wantStatus      api.StreamFilterStatus
		wantIndex       int
	}{
		{
			[]*mockStreamFilter{
				{appendReturn: api.StreamFilterContinue},
				{appendReturn: api.StreamFilterContinue},
			},
			api.StreamFilterContinue,
			0,
		},
		{
			[]*mockStreamFilter{
				{appendReturn: api.StreamFilterContinue},
				{appendReturn: api.StreamFilterStop},
			},
			api.StreamFilterStop,
			0,
		},
		{
			[]*mockStreamFilter{
				{appendReturn: api.StreamFilterContinue},
				{appendReturn: api.StreamFiltertermination},
			},
			api.StreamFiltertermination,
			0,
		},
		{
			[]*mockStreamFilter{
				{appendReturn: api.StreamFilterContinue},
				{appendReturn: api.StreamFilterReMatchRoute},
			},
			api.StreamFilterReMatchRoute,
			0,
		},
		{
			[]*mockStreamFilter{
				{appendReturn: api.StreamFilterContinue},
				{appendReturn: api.StreamFilterReChooseHost},
			},
			api.StreamFilterReChooseHost,
			0,
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			d := &DefaultStreamFilterManagerImpl{}
			for _, filter := range tt.senderFilters {
				d.AddStreamSenderFilter(filter, api.BeforeSend)
			}
			gotFilterStatus := d.RunSenderFilter(context.TODO(), api.BeforeSend,
				protocol.CommonHeader{}, buffer.NewIoBuffer(0), protocol.CommonHeader{},
				func(status api.StreamFilterStatus) {}) // do nothing for coverage
			if gotFilterStatus != tt.wantStatus || d.senderFiltersIndex != tt.wantIndex {
				t.Errorf("RunSenderFilter() = %v, want %v; index = %v, want %v",
					gotFilterStatus, tt.wantStatus, d.senderFiltersIndex, tt.wantIndex)
			}
		})
	}
}