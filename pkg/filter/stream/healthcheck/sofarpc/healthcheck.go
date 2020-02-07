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
	"encoding/json"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/rpc/sofarpc"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

// todo: support cached pass through

func init() {
	api.RegisterStream("healthcheck", CreateHealthCheckFilterFactory)
}

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
	handler api.StreamReceiverFilterHandler
}

// NewHealthCheckFilter used to create new health check filter
func NewHealthCheckFilter(context context.Context, config *v2.HealthCheckFilter) api.StreamReceiverFilter {
	return &healthCheckFilter{
		context:                      context,
		passThrough:                  config.PassThrough,
		cacheTime:                    config.CacheTime,
		clusterMinHealthyPercentages: config.ClusterMinHealthyPercentage,
	}
}

func (f *healthCheckFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if cmd, ok := headers.(sofarpc.SofaRpcCmd); ok {
		if cmd.CommandCode() == sofarpc.HEARTBEAT {
			f.protocol = cmd.ProtocolCode()
			f.requestID = cmd.RequestID()
			f.healthCheckReq = true
			f.handler.RequestInfo().SetHealthCheck(true)

			if !f.passThrough {
				f.handleIntercept()
				return api.StreamFilterStop
			}
		}
	}
	return api.StreamFilterContinue
}

func (f *healthCheckFilter) handleIntercept() {
	// todo: cal status based on cluster healthy host stats and f.clusterMinHealthyPercentages
	hbAck := sofarpc.NewHeartbeatAck(f.protocol)
	if hbAck == nil {
		log.DefaultLogger.Alertf(types.ErrorKeyHeartBeat, "unknown protocol code: [%x] while intercept healthcheck.", f.protocol)
		//TODO: set hijack reply - codec error, actually this would happen at codec stage which is before this
	}
	f.handler.AppendHeaders(hbAck, true)
}

func (f *healthCheckFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *healthCheckFilter) OnDestroy() {}

// HealthCheckFilterConfigFactory Filter Config Factory
type HealthCheckFilterConfigFactory struct {
	FilterConfig *v2.HealthCheckFilter
}

func (f *HealthCheckFilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewHealthCheckFilter(context, f.FilterConfig)
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
}

// CreateHealthCheckFilterFactory
func CreateHealthCheckFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	f, err := ParseHealthCheckFilter(conf)
	if err != nil {
		return nil, err
	}
	return &HealthCheckFilterConfigFactory{
		FilterConfig: f,
	}, nil
}

// ParseHealthCheckFilter
func ParseHealthCheckFilter(cfg map[string]interface{}) (*v2.HealthCheckFilter, error) {
	filterConfig := &v2.HealthCheckFilter{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, filterConfig); err != nil {
		return nil, err
	}

	return filterConfig, nil
}
