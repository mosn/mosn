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

package dubbo

import (
	"context"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/istio"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

func init() {
	api.RegisterStream(v2.DubboStream, buildStream)
}

type factory struct{}

func buildStream(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	Init(conf)
	return &factory{}, nil
}

func (f *factory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := buildDubboFilter(ctx)
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
	callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
}

type dubboFilter struct {
	handler api.StreamReceiverFilterHandler
}

func buildDubboFilter(ctx context.Context) *dubboFilter {
	return &dubboFilter{}
}

func (d *dubboFilter) OnDestroy() {}

func (d *dubboFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {

	proto, err := variable.Get(ctx, types.VariableDownStreamProtocol)
	if err != nil || proto == nil || dubbo.ProtocolName != proto {
		return api.StreamFilterContinue
	}

	lv, err := variable.Get(ctx, types.VariableListenerName)
	if err != nil || lv == nil {
		return api.StreamFilterContinue
	}
	listener := lv.(string)

	service, ok := headers.Get(dubbo.ServiceNameHeader)
	if !ok {
		log.DefaultLogger.Errorf("%s is empty, may be the protocol is not dubbo", dubbo.ServiceNameHeader)
		return api.StreamFiltertermination
	}

	// adapt dubbo service to http host
	variable.SetString(ctx, types.VarHost, service)
	// because use http rule, so should add default path
	variable.SetString(ctx, types.VarPath, "/")

	method, _ := headers.Get(dubbo.MethodNameHeader)
	stats := getStats(listener, service, method)
	if stats != nil {
		stats.RequestServiceInfo.Inc(1)

		variable.SetString(ctx, VarDubboRequestService, service)
		variable.SetString(ctx, VarDubboRequestMethod, method)
	}

	for k, v := range istio.GetPodLabels() {
		headers.Set(k, v)
	}

	return api.StreamFilterContinue
}

func (d *dubboFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	d.handler = handler
}

func (d *dubboFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	lv, err := variable.Get(ctx, types.VariableListenerName)
	if err != nil || lv == nil {
		return api.StreamFilterContinue
	}
	listener := lv.(string)
	service, err := variable.GetString(ctx, VarDubboRequestService)
	if err != nil {
		log.DefaultLogger.Warnf("Get request service info failed: %+v", err)
		return api.StreamFilterContinue
	}
	method, err := variable.GetString(ctx, VarDubboRequestMethod)
	if err != nil {
		log.DefaultLogger.Warnf("Get request method info failed: %+v", err)
		return api.StreamFilterContinue
	}

	stats := getStats(listener, service, method)
	if stats == nil {
		return api.StreamFilterContinue
	}

	var isSuccess bool
	switch frame := headers.(type) {
	case *dubbo.Frame:
		isSuccess = frame.GetStatusCode() == dubbo.RespStatusOK
	default:
		log.DefaultLogger.Errorf("this filter {%s} just for dubbo protocol, please check your config.", v2.DubboStream)
		return api.StreamFiltertermination
	}

	if isSuccess {
		stats.ResponseSucc.Inc(1)
	} else {
		stats.ResponseFail.Inc(1)
	}
	return api.StreamFilterContinue
}

func (d *dubboFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {}
