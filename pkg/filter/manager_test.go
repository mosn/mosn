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

func (m *mockStreamFilter1) CheckPhase(phase api.FilterPhase) bool {
	return int(phase) == 101
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

func (m *mockStreamFilter2) CheckPhase(phase api.FilterPhase) bool {
	return int(phase) == 102
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
	d.AddStreamReceiverFilter(&mockStreamFilter1{}, api.FilterPhase(101))
	d.AddStreamReceiverFilter(&mockStreamFilter2{}, api.FilterPhase(102))
	if len(d.receiverFilters) != 2 {
		t.Errorf("DefaultStreamFilterManagerImpl.AddStreamReceiverFilter failed, len: %v", len(d.receiverFilters))
	}
}

func TestDefaultStreamFilterManagerImpl_AddStreamSenderFilter(t *testing.T) {
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamSenderFilter(&mockStreamFilter1{})
	d.AddStreamSenderFilter(&mockStreamFilter2{})
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
	d.AddStreamReceiverFilter(&mockStreamFilter1{}, api.FilterPhase(101))
	d.AddStreamSenderFilter(&mockStreamFilter2{})
	d.OnDestroy()
	if destroyCount != 2 {
		t.Errorf("DefaultStreamFilterManagerImpl.OnDestroy failed, destroyCount: %v", destroyCount)
	}
}

func TestDefaultStreamFilterManagerImpl_RunReceiverFilter(t *testing.T) {
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamReceiverFilter(&mockStreamFilter1{}, api.FilterPhase(101))
	d.AddStreamReceiverFilter(&mockStreamFilter2{}, api.FilterPhase(102))
	d.AddStreamReceiverFilter(&mockStreamFilter1{}, api.FilterPhase(101))
	d.AddStreamReceiverFilter(&mockStreamFilter2{}, api.FilterPhase(102))
	d.RunReceiverFilter(context.TODO(), api.FilterPhase(101), nil, nil, nil, DefaultStreamFilterStatusHandler)
	if onReceiveCount != 2 {
		t.Error("DefaultStreamFilterManagerImpl.RunReceiverFilter failed")
	}
}

func TestDefaultStreamFilterManagerImpl_RunSenderFilter(t *testing.T) {
	d := &DefaultStreamFilterManagerImpl{}
	d.AddStreamSenderFilter(&mockStreamFilter1{})
	d.AddStreamSenderFilter(&mockStreamFilter2{})
	d.RunSenderFilter(context.TODO(), UndefinedFilterPhase, nil, nil, nil, DefaultStreamFilterStatusHandler)
	if appendCount != 2 {
		t.Error("DefaultStreamFilterManagerImpl.RunSenderFilter failed")
	}
}
