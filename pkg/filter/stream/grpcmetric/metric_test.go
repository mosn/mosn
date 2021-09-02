package grpcmetric

import (
	"context"
	"testing"

	"mosn.io/mosn/pkg/types"

	"github.com/stretchr/testify/assert"

	"mosn.io/pkg/header"

	"mosn.io/api"
)

type mockMetricFilter struct {
	api.StreamFilterChainFactoryCallbacks
	rf api.StreamReceiverFilter
	sf api.StreamSenderFilter
	p1 api.ReceiverFilterPhase
	p2 api.SenderFilterPhase
}

func (m *mockMetricFilter) AddStreamReceiverFilter(filter api.StreamReceiverFilter, p api.ReceiverFilterPhase) {
	m.rf = filter
	m.p1 = p
}

func (m *mockMetricFilter) AddStreamSenderFilter(filter api.StreamSenderFilter, p api.SenderFilterPhase) {
	m.sf = filter
	m.p2 = p
}

func TestFactory(t *testing.T) {
	fac, err := buildStream(map[string]interface{}{})
	if err != nil {
		t.Fatalf("create factory failed: %v", err)
	}
	cb := &mockMetricFilter{}
	fac.CreateFilterChain(context.TODO(), cb)
	if cb.rf == nil || cb.p1 != api.AfterRoute {
		t.Fatalf("create filter chain failed")
	}
	if cb.sf == nil || cb.p2 != api.BeforeSend {
		t.Fatalf("create filter chain failed")
	}
}

func TestAppend(t *testing.T) {
	mf := &metricFilter{}
	h := &header.CommonHeader{}
	r := mf.Append(context.Background(), h, nil, nil)
	assert.Equal(t, api.StreamFilterContinue, r)
	assert.Equal(t, len(statsFactory), 0)

	h.Set(types.GrpcServiceName, "service1")
	h.Set(types.GrpcRequestResult, types.SUCCESS)
	mf.Append(context.Background(), h, nil, nil)
	state := statsFactory["service1"]
	assert.Equal(t, int(state.ResponseSuccess.Count()), 1)
	assert.Equal(t, int(state.RequestServiceTootle.Count()), 1)
	assert.Equal(t, int(state.ResponseFail.Count()), 0)

	h.Set(types.GrpcServiceName, "service1")
	h.Set(types.GrpcRequestResult, types.FAIL)
	mf.Append(context.Background(), h, nil, nil)
	state = statsFactory["service1"]
	assert.Equal(t, int(state.ResponseSuccess.Count()), 1)
	assert.Equal(t, int(state.RequestServiceTootle.Count()), 2)
	assert.Equal(t, int(state.ResponseFail.Count()), 1)

	h.Set(types.GrpcServiceName, "service2")
	h.Set(types.GrpcRequestResult, types.SUCCESS)
	mf.Append(context.Background(), h, nil, nil)
	state = statsFactory["service1"]
	assert.Equal(t, int(state.ResponseSuccess.Count()), 1)
	assert.Equal(t, int(state.RequestServiceTootle.Count()), 2)
	assert.Equal(t, int(state.ResponseFail.Count()), 1)
	state = statsFactory["service2"]
	assert.Equal(t, int(state.ResponseSuccess.Count()), 1)
	assert.Equal(t, int(state.RequestServiceTootle.Count()), 1)
	assert.Equal(t, int(state.ResponseFail.Count()), 0)
}
