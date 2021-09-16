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

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/filter/network/grpc"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream(v2.GrpcMetricFilter, buildStream)
}

type factory struct {
	s *state
}

func buildStream(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	return &factory{s: newState()}, nil
}

func (f *factory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := &metricFilter{st: f.s}
	callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
}

type metricFilter struct {
	sendHandler api.StreamSenderFilterHandler
	st          *state
}

func (d *metricFilter) OnDestroy() {}

func (d *metricFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	return api.StreamFilterContinue
}

func (d *metricFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	svcName, err := variable.GetVariableValue(ctx, grpc.VarGrpcServiceName)
	if err != nil {
		return api.StreamFilterContinue
	}
	success, err := variable.GetVariableValue(ctx, grpc.VarGrpcRequestResult)
	if err != nil {
		return api.StreamFilterContinue
	}

	stats := d.st.getStats(svcName)
	if stats == nil {
		return api.StreamFilterContinue
	}
	stats.requestServiceTotal.Inc(1)
	if success == "true" {
		stats.responseSuccess.Inc(1)
	} else {
		stats.responseFail.Inc(1)
	}
	return api.StreamFilterContinue
}

func (d *metricFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	d.sendHandler = handler
}
