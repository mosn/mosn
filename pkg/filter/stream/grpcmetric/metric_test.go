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

package grpcmetric

import (
	"context"
	"strconv"
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

	variable.SetVariableValue(ctx, grpc.GrpcServiceName, "service1")
	variable.SetVariableValue(ctx, grpc.GrpcRequestResult, "true")
	variable.SetVariableValue(ctx, grpc.GrpcServiceCostTime, strconv.FormatInt(123, 10))
	mf.Append(ctx, h, nil, nil)
	state := mf.ft.s.getStats("service1")
	assert.Equal(t, int(state.responseSuccess.Count()), 1)
	assert.Equal(t, int(state.requestServiceTootle.Count()), 1)
	assert.Equal(t, int(state.responseFail.Count()), 0)

	variable.SetVariableValue(ctx, grpc.GrpcServiceName, "service1")
	variable.SetVariableValue(ctx, grpc.GrpcRequestResult, "false")
	variable.SetVariableValue(ctx, grpc.GrpcServiceCostTime, strconv.FormatInt(123, 10))
	mf.Append(ctx, h, nil, nil)
	state = mf.ft.s.getStats("service1")
	assert.Equal(t, int(state.responseSuccess.Count()), 1)
	assert.Equal(t, int(state.requestServiceTootle.Count()), 2)
	assert.Equal(t, int(state.responseFail.Count()), 1)

	variable.SetVariableValue(ctx, grpc.GrpcServiceName, "service2")
	variable.SetVariableValue(ctx, grpc.GrpcRequestResult, "true")
	variable.SetVariableValue(ctx, grpc.GrpcServiceCostTime, strconv.FormatInt(123, 10))
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
	variable.SetVariableValue(ctx, grpc.GrpcServiceName, "service1")
	variable.SetVariableValue(ctx, grpc.GrpcRequestResult, "true")
	variable.SetVariableValue(ctx, grpc.GrpcServiceCostTime, strconv.FormatInt(12312421421412, 10))
	for n := 0; n < b.N; n++ {
		mf.Append(ctx, h, nil, nil)
	}
}
