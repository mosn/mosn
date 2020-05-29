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

package faulttolerance

import (
	"context"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/filter/stream/faulttolerance/regulator"
	"mosn.io/pkg/buffer"
)

type SendFilter struct {
	config            *v2.FaultToleranceFilterConfig
	invocationFactory *regulator.InvocationStatFactory
	handler           api.StreamSenderFilterHandler
}

func NewSendFilter(config *v2.FaultToleranceFilterConfig, invocationStatFactory *regulator.InvocationStatFactory) *SendFilter {
	filter := &SendFilter{
		config:            config,
		invocationFactory: invocationStatFactory,
	}
	return filter
}

func (f *SendFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if !f.config.Enabled {
		return api.StreamFilterContinue
	}

	newDimension := regulator.GetNewDimensionFunc()
	requestInfo := f.handler.RequestInfo()
	if newDimension == nil || requestInfo == nil {
		return api.StreamFilterContinue
	}

	host := requestInfo.UpstreamHost()
	if host != nil {
		dimension := newDimension(requestInfo)
		isException := f.IsException(requestInfo)
		stat := f.invocationFactory.GetInvocationStat(host, dimension)
		stat.Call(isException)
	}
	return api.StreamFilterContinue
}

func (f *SendFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.handler = handler
}

func (f *SendFilter) OnDestroy() {

}

func (f *SendFilter) IsException(requestInfo api.RequestInfo) bool {
	responseCode := requestInfo.ResponseCode()
	if _, ok := f.config.ExceptionTypes[uint32(responseCode)]; ok {
		return true
	}
	return false
}
