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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/filter"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// todo: support cached pass through

func init() {
	filter.RegisterStream("healthcheck", CreateHealthCheckFilterFactory)
}

// types.StreamSenderFilter
type healthCheckFilter struct {
	context context.Context

	// config
	passThrough                  bool
	cacheTime                    time.Duration
	clusterMinHealthyPercentages map[string]float32
	// request properties
	protocol       byte
	requestID      uint64
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
	if cmd, ok := headers.(sofarpc.SofaRpcCmd); ok {
		if cmd.CommandCode() == sofarpc.HEARTBEAT {
			f.protocol = cmd.ProtocolCode()
			f.requestID = cmd.RequestID()
			f.healthCheckReq = true
			f.handler.RequestInfo().SetHealthCheck(true)

			if !f.passThrough {
				f.handleIntercept()
				return types.StreamFilterStop
			}
		}
	}
	return types.StreamFilterContinue
}

func (f *healthCheckFilter) handleIntercept() {
	// todo: cal status based on cluster healthy host stats and f.clusterMinHealthyPercentages
	hbAck := sofarpc.NewHeartbeatAck(f.protocol)
	if hbAck == nil {
		log.ByContext(f.context).Errorf("Unknown protocol code: [%x] while intercept healthcheck.", f.protocol)
		//TODO: set hijack reply - codec error, actually this would happen at codec stage which is before this
	}
	f.handler.AppendHeaders(hbAck, true)
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
	callbacks.AddStreamReceiverFilter(filter, types.DownFilterAfterRoute)
}

// CreateHealthCheckFilterFactory
func CreateHealthCheckFilterFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	return &HealthCheckFilterConfigFactory{
		FilterConfig: config.ParseHealthCheckFilter(conf),
	}, nil
}
