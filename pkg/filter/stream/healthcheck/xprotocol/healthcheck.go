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

package sofarpc

import (
	"context"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/filter/stream/healthcheck/sofarpc"
	"mosn.io/mosn/pkg/types"
	"mosn.io/api"
	"mosn.io/mosn/pkg/stream/xprotocol"
)

// todo: support cached pass through

func init() {
	api.RegisterStream("xprotocol_healthcheck", CreateHealthCheckFilterFactory)
}

// types.StreamSenderFilter
type healthCheckFilter struct {
	context context.Context

	// config
	passThrough                  bool
	cacheTime                    time.Duration
	clusterMinHealthyPercentages map[string]float32
	// request properties
	protocol       string
	healthCheckReq bool
	// callbacks
	handler types.StreamReceiverFilterHandler
}

// NewHealthCheckFilter used to create new health check filter
func NewHealthCheckFilter(context context.Context, config *v2.HealthCheckFilter) types.StreamReceiverFilter {
	return &healthCheckFilter{
		context:                      context,
		passThrough:                  config.PassThrough,
		cacheTime:                    config.CacheTime,
		clusterMinHealthyPercentages: config.ClusterMinHealthyPercentage,
	}
}

func (f *healthCheckFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) types.StreamFilterStatus {
	if protocol, ok := headers.Get(types.HeaderXprotocolHeartbeat); ok {
		f.protocol = protocol
		f.healthCheckReq = true
		f.handler.RequestInfo().SetHealthCheck(true)
		if !f.passThrough {
			f.handleIntercept(headers)
			return types.StreamFilterStop
		}
	}
	return types.StreamFilterContinue
}

func (f *healthCheckFilter) handleIntercept(headers types.HeaderMap) {
	// todo: cal status based on cluster healthy host stats and f.clusterMinHealthyPercentages
	headers.Set(xprotocol.X_PROTOCOL_HEARTBEAT_HIJACK, "true")
	f.handler.AppendHeaders(headers, true)
}

func (f *healthCheckFilter) SetReceiveFilterHandler(handler types.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *healthCheckFilter) OnDestroy() {}

// HealthCheckFilterConfigFactory Filter Config Factory
type HealthCheckFilterConfigFactory struct {
	FilterConfig *v2.HealthCheckFilter
}

func (f *HealthCheckFilterConfigFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	filter := NewHealthCheckFilter(context, f.FilterConfig)
	callbacks.AddStreamReceiverFilter(filter, types.DownFilter)
}

// CreateHealthCheckFilterFactory
func CreateHealthCheckFilterFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	return &HealthCheckFilterConfigFactory{
		FilterConfig: sofarpc.ParseHealthCheckFilter(conf),
	}, nil
}
