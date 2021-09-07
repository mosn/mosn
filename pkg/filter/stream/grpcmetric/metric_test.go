package grpcmetric

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"

	"mosn.io/mosn/pkg/filter/network/grpc"
	"mosn.io/mosn/pkg/variable"

	"mosn.io/pkg/header"
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
	f, _ := buildStream(nil)
	mf := &metricFilter{ft: f.(*factory)}
	h := &header.CommonHeader{}
	ctx := variable.NewVariableContext(context.TODO())
	r := mf.Append(ctx, h, nil, nil)
	assert.Equal(t, api.StreamFilterContinue, r)
	assert.Equal(t, len(mf.ft.s.statsFactory), 0)

	variable.Set(ctx, grpc.GrpcServiceName, "service1")
	variable.Set(ctx, grpc.GrpcRequestResult, true)
	variable.Set(ctx, grpc.GrpcServiceCostTime, int64(123))
	mf.Append(ctx, h, nil, nil)
	state := mf.ft.s.getStats("service1")
	assert.Equal(t, int(state.responseSuccess.Count()), 1)
	assert.Equal(t, int(state.requestServiceTootle.Count()), 1)
	assert.Equal(t, int(state.responseFail.Count()), 0)

	variable.Set(ctx, grpc.GrpcServiceName, "service1")
	variable.Set(ctx, grpc.GrpcRequestResult, false)
	variable.Set(ctx, grpc.GrpcServiceCostTime, int64(123))
	mf.Append(ctx, h, nil, nil)
	state = mf.ft.s.getStats("service1")
	assert.Equal(t, int(state.responseSuccess.Count()), 1)
	assert.Equal(t, int(state.requestServiceTootle.Count()), 2)
	assert.Equal(t, int(state.responseFail.Count()), 1)

	variable.Set(ctx, grpc.GrpcServiceName, "service2")
	variable.Set(ctx, grpc.GrpcRequestResult, true)
	variable.Set(ctx, grpc.GrpcServiceCostTime, int64(123))
	mf.Append(ctx, h, nil, nil)
	state = mf.ft.s.getStats("service1")
	assert.Equal(t, int(state.responseSuccess.Count()), 1)
	assert.Equal(t, int(state.requestServiceTootle.Count()), 2)
	assert.Equal(t, int(state.responseFail.Count()), 1)
	state = mf.ft.s.getStats("service2")
	assert.Equal(t, int(state.responseSuccess.Count()), 1)
	assert.Equal(t, int(state.requestServiceTootle.Count()), 1)
	assert.Equal(t, int(state.responseFail.Count()), 0)
}

func BenchmarkWithTimer(b *testing.B) {
	f, _ := buildStream(nil)
	mf := &metricFilter{ft: f.(*factory)}
	h := &header.CommonHeader{}
	ctx := variable.NewVariableContext(context.TODO())
	variable.Set(ctx, grpc.GrpcServiceName, "service1")
	variable.Set(ctx, grpc.GrpcRequestResult, true)
	variable.Set(ctx, grpc.GrpcServiceCostTime, int64(12312312312312))
	for n := 0; n < b.N; n++ {
		mf.Append(ctx, h, nil, nil)
	}
}
