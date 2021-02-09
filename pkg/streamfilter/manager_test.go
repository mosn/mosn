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

package streamfilter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream("testStreamFilter", CreateMockStreamFilterFactory)
}

var (
	destroyCount           = 0
	onReceiveCount         = 0
	appendCount            = 0
	logCount               = 0
	createFilterChainCount = 0
)

func CreateMockStreamFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	return &mockStreamFilterFactory{}, nil
}

type mockStreamFilterFactory struct{}

func (m *mockStreamFilterFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	createFilterChainCount++
}

func TestStreamFilterManager(t *testing.T) {
	manager := GetStreamFilterManager()
	assert.Equal(t, manager.AddOrUpdateStreamFilterConfig("", nil), ErrInvalidKey)
	configWith2Filter := StreamFiltersConfig{
		{Type: "testStreamFilter", Config: nil},
		{Type: "testStreamFilter", Config: nil},
	}
	manager.AddOrUpdateStreamFilterConfig("key", configWith2Filter)
	v := manager.GetStreamFilterFactory("key")
	if factory, ok := v.(*StreamFilterFactoryImpl); ok {
		if len(factory.factories.Load().([]api.StreamFilterChainFactory)) != 2 {
			t.Errorf("manager config factory len != 2")
		}
	} else {
		t.Errorf("manager unexpected object type")
	}

	configWith3Filter := StreamFiltersConfig{
		{Type: "testStreamFilter", Config: nil},
		{Type: "testStreamFilter", Config: nil},
		{Type: "testStreamFilter", Config: nil},
	}
	manager.AddOrUpdateStreamFilterConfig("key", configWith3Filter)
	v = manager.GetStreamFilterFactory("key")
	if factory, ok := v.(*StreamFilterFactoryImpl); ok {
		if len(factory.factories.Load().([]api.StreamFilterChainFactory)) != 3 {
			t.Errorf("manager config factory len != 3")
		}
	} else {
		t.Errorf("manager unexpected object type")
	}
}

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

func TestDefaultStreamFilterChainImpl_AddStreamAccessLog(t *testing.T) {
	d := &DefaultStreamFilterChainImpl{}
	d.AddStreamAccessLog(mockStreamAccessLog{})
	d.AddStreamAccessLog(mockStreamAccessLog{})
	if len(d.streamAccessLogs) != 2 {
		t.Errorf("DefaultStreamFilterChainImpl.AddStreamAccessLog failed, len: %v", len(d.streamAccessLogs))
	}
}

func TestDefaultStreamFilterChainImpl_AddStreamReceiverFilter(t *testing.T) {
	d := &DefaultStreamFilterChainImpl{}
	d.AddStreamReceiverFilter(&mockStreamFilter{onReceiveReturn: api.StreamFilterContinue}, api.BeforeRoute)
	d.AddStreamReceiverFilter(&mockStreamFilter{onReceiveReturn: api.StreamFilterContinue}, api.BeforeRoute)
	if len(d.receiverFilters) != 2 {
		t.Errorf("DefaultStreamFilterChainImpl.AddStreamReceiverFilter failed, len: %v", len(d.receiverFilters))
	}
}

func TestDefaultStreamFilterChainImpl_AddStreamSenderFilter(t *testing.T) {
	d := &DefaultStreamFilterChainImpl{}
	d.AddStreamSenderFilter(&mockStreamFilter{appendReturn: api.StreamFilterContinue}, api.BeforeSend)
	d.AddStreamSenderFilter(&mockStreamFilter{appendReturn: api.StreamFilterContinue}, api.BeforeSend)
	if len(d.senderFilters) != 2 {
		t.Errorf("DefaultStreamFilterChainImpl.AddStreamSenderFilter failed, len: %v", len(d.senderFilters))
	}
}

func TestDefaultStreamFilterChainImpl_Log(t *testing.T) {
	d := &DefaultStreamFilterChainImpl{}
	d.AddStreamAccessLog(mockStreamAccessLog{})
	d.AddStreamAccessLog(mockStreamAccessLog{})
	d.Log(context.TODO(), nil, nil, nil)
	if logCount != 2 {
		t.Error("DefaultStreamFilterChainImpl.Log failed")
	}
}

func TestDefaultStreamFilterChainImpl_OnDestroy(t *testing.T) {
	d := &DefaultStreamFilterChainImpl{}
	d.AddStreamReceiverFilter(&mockStreamFilter{onReceiveReturn: api.StreamFilterContinue}, api.BeforeRoute)
	d.AddStreamSenderFilter(&mockStreamFilter{appendReturn: api.StreamFilterContinue}, api.BeforeSend)
	d.OnDestroy()
	if destroyCount != 2 {
		t.Errorf("DefaultStreamFilterChainImpl.OnDestroy failed, destroyCount: %v", destroyCount)
	}
}

func TestDefaultStreamFilterChainImpl_RunReceiverFilter(t *testing.T) {
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
			d := &DefaultStreamFilterChainImpl{}
			for _, filter := range tt.receiverFilters {
				d.AddStreamReceiverFilter(filter, api.BeforeRoute)
			}
			gotFilterStatus := d.RunReceiverFilter(context.TODO(), api.BeforeRoute,
				protocol.CommonHeader{}, buffer.NewIoBuffer(0), protocol.CommonHeader{},
				func(phase api.ReceiverFilterPhase, status api.StreamFilterStatus) {}) // do nothing for coverage
			if gotFilterStatus != tt.wantStatus || d.receiverFiltersIndex != tt.wantIndex {
				t.Errorf("RunReceiverFilter() = %v, want %v; index = %v, want %v",
					gotFilterStatus, tt.wantStatus, d.receiverFiltersIndex, tt.wantIndex)
			}
		})
	}
}

func TestDefaultStreamFilterChainImpl_RunSenderFilter(t *testing.T) {
	tests := []struct {
		senderFilters []*mockStreamFilter
		wantStatus    api.StreamFilterStatus
		wantIndex     int
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
			d := &DefaultStreamFilterChainImpl{}
			for _, filter := range tt.senderFilters {
				d.AddStreamSenderFilter(filter, api.BeforeSend)
			}
			gotFilterStatus := d.RunSenderFilter(context.TODO(), api.BeforeSend,
				protocol.CommonHeader{}, buffer.NewIoBuffer(0), protocol.CommonHeader{},
				func(phase api.SenderFilterPhase, status api.StreamFilterStatus) {}) // do nothing for coverage
			if gotFilterStatus != tt.wantStatus || d.senderFiltersIndex != tt.wantIndex {
				t.Errorf("RunSenderFilter() = %v, want %v; index = %v, want %v",
					gotFilterStatus, tt.wantStatus, d.senderFiltersIndex, tt.wantIndex)
			}
		})
	}
}
