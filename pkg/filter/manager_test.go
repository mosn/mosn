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
	"mosn.io/pkg/buffer"
)

var destroyCount = 0
var onReceiveCount = 0
var appendCount = 0
var logCount = 0

type mockStreamFilter1 struct{}

func (m *mockStreamFilter1) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	appendCount++
	return api.StreamFilterContinue
}

func (m *mockStreamFilter1) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
}

func (m *mockStreamFilter1) OnDestroy() {
	destroyCount++
}

func (m *mockStreamFilter1) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	onReceiveCount++
	return api.StreamFilterContinue
}

func (m *mockStreamFilter1) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
}


type mockStreamFilter2 struct{}

func (m *mockStreamFilter2) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	appendCount++
	return api.StreamFilterContinue
}

func (m *mockStreamFilter2) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
}

func (m *mockStreamFilter2) OnDestroy() {
	destroyCount++
}

func (m *mockStreamFilter2) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	onReceiveCount++
	return api.StreamFilterContinue
}

func (m *mockStreamFilter2) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
}


type mockStreamAccessLog struct{}

func (m mockStreamAccessLog) Log(ctx context.Context, reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	logCount++
}

func TestDefaultStreamFilterStatusHandler(t *testing.T) {
	tests := []struct {
		args api.StreamFilterStatus
		want StreamFilterChainStatus
	}{
		{api.StreamFilterContinue, StreamFilterChainContinue},
		{api.StreamFilterStop, StreamFilterChainReset},
		{api.StreamFiltertermination, StreamFilterChainReset},
		{api.StreamFilterReMatchRoute, StreamFilterChainContinue},
		{api.StreamFilterReChooseHost, StreamFilterChainContinue},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := DefaultStreamFilterStatusHandler(tt.args); got != tt.want {
				t.Errorf("DefaultStreamFilterStatusHandler() = %v, want %v", got, tt.want)
			}
		})
	}
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
	whateverReceiverFilterPhase := api.ReceiverFilterPhase(999)
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamReceiverFilter(&mockStreamFilter1{}, whateverReceiverFilterPhase)
	d.AddStreamReceiverFilter(&mockStreamFilter2{}, whateverReceiverFilterPhase)
	if len(d.receiverFilters) != 2 {
		t.Errorf("DefaultStreamFilterManagerImpl.AddStreamReceiverFilter failed, len: %v", len(d.receiverFilters))
	}
}

func TestDefaultStreamFilterManagerImpl_AddStreamSenderFilter(t *testing.T) {
	whateverSenderFilterPhase := api.SenderFilterPhase(999)
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamSenderFilter(&mockStreamFilter1{}, whateverSenderFilterPhase)
	d.AddStreamSenderFilter(&mockStreamFilter2{}, whateverSenderFilterPhase)
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
	whateverReceiverFilterPhase := api.ReceiverFilterPhase(998)
	whateverSenderFilterPhase := api.SenderFilterPhase(999)
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamReceiverFilter(&mockStreamFilter1{}, whateverReceiverFilterPhase)
	d.AddStreamSenderFilter(&mockStreamFilter2{}, whateverSenderFilterPhase)
	d.OnDestroy()
	if destroyCount != 2 {
		t.Errorf("DefaultStreamFilterManagerImpl.OnDestroy failed, destroyCount: %v", destroyCount)
	}
}

func TestDefaultStreamFilterManagerImpl_RunReceiverFilter(t *testing.T) {
	receiverFilterPhase1 := api.ReceiverFilterPhase(991)
	receiverFilterPhase2 := api.ReceiverFilterPhase(992)
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamReceiverFilter(&mockStreamFilter1{}, receiverFilterPhase1)
	d.AddStreamReceiverFilter(&mockStreamFilter2{}, receiverFilterPhase2)
	d.AddStreamReceiverFilter(&mockStreamFilter1{}, receiverFilterPhase1)
	d.AddStreamReceiverFilter(&mockStreamFilter2{}, receiverFilterPhase2)
	d.RunReceiverFilter(context.TODO(), receiverFilterPhase1, nil, nil, nil, DefaultStreamFilterStatusHandler)
	if onReceiveCount != 2 {
		t.Error("DefaultStreamFilterManagerImpl.RunReceiverFilter failed")
	}
}

func TestDefaultStreamFilterManagerImpl_RunSenderFilter(t *testing.T) {
	whateverSenderFilterPhase := api.SenderFilterPhase(999)
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamSenderFilter(&mockStreamFilter1{}, whateverSenderFilterPhase)
	d.AddStreamSenderFilter(&mockStreamFilter2{}, whateverSenderFilterPhase)
	d.RunSenderFilter(context.TODO(), whateverSenderFilterPhase, nil, nil, nil, DefaultStreamFilterStatusHandler)
	if appendCount != 2 {
		t.Error("DefaultStreamFilterManagerImpl.RunSenderFilter failed")
	}
}