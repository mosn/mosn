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
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/codec"
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
	intercept      bool
	protocol       byte
	requestID      uint32
	healthCheckReq bool
	// callbacks
	cb types.StreamReceiverFilterCallbacks
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

func (f *healthCheckFilter) OnDecodeHeaders(headers map[string]string, endStream bool) types.FilterHeadersStatus {
	if cmdCodeStr, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)]; ok {
		cmdCode := sofarpc.ConvertPropertyValueInt16(cmdCodeStr)

		//sofarpc.HEARTBEAT(0) is equal to sofarpc.TR_HEARTBEAT(0)
		if cmdCode == sofarpc.HEARTBEAT {
			protocolStr := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)]
			f.protocol = sofarpc.ConvertPropertyValueUint8(protocolStr)
			requestIDStr := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)]
			f.requestID = sofarpc.ConvertPropertyValueUint32(requestIDStr)
			f.healthCheckReq = true
			f.cb.RequestInfo().SetHealthCheck(true)

			if !f.passThrough {
				f.intercept = true
			}

			endStream = true
		}
	}

	if endStream && f.intercept {
		f.handleIntercept()
	}

	if f.intercept {
		return types.FilterHeadersStatusStopIteration
	}

	return types.FilterHeadersStatusContinue
}

func (f *healthCheckFilter) OnDecodeData(buf types.IoBuffer, endStream bool) types.FilterDataStatus {
	if endStream && f.intercept {
		f.handleIntercept()
	}

	if f.intercept {
		return types.FilterDataStatusStopIterationNoBuffer
	}

	return types.FilterDataStatusContinue
}

func (f *healthCheckFilter) OnDecodeTrailers(trailers map[string]string) types.FilterTrailersStatus {
	if f.intercept {
		f.handleIntercept()
	}

	if f.intercept {
		return types.FilterTrailersStatusStopIteration
	}

	return types.FilterTrailersStatusContinue
}

func (f *healthCheckFilter) handleIntercept() {
	// todo: cal status based on cluster healthy host stats and f.clusterMinHealthyPercentages

	var resp interface{}

	//TODO add protocol-level interface for heartbeat process, like Protocols.TriggerHeartbeat(protocolCode, requestID)&Protocols.ReplyHeartbeat(protocolCode, requestID)
	switch f.protocol {
	//case f.protocol == sofarpc.PROTOCOL_CODE:
	//resp = codec.NewTrHeartbeatAck( f.requestID)
	case sofarpc.PROTOCOL_CODE_V1, sofarpc.PROTOCOL_CODE_V2:
		//boltv1 and boltv2 use same heartbeat struct as BoltV1
		resp = codec.NewBoltHeartbeatAck(f.requestID)
	default:
		log.ByContext(f.context).Errorf("Unknown protocol code: [%x] while intercept healthcheck.", f.protocol)
		//TODO: set hijack reply - codec error, actually this would happen at codec stage which is before this
	}

	f.cb.AppendHeaders(resp, true)
}

func (f *healthCheckFilter) SetDecoderFilterCallbacks(cb types.StreamReceiverFilterCallbacks) {
	f.cb = cb
}

func (f *healthCheckFilter) OnDestroy() {}

// HealthCheckFilterConfigFactory Filter Config Factory
type HealthCheckFilterConfigFactory struct {
	FilterConfig *v2.HealthCheckFilter
}

func (f *HealthCheckFilterConfigFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	filter := NewHealthCheckFilter(context, f.FilterConfig)
	callbacks.AddStreamReceiverFilter(filter)
}

// CreateHealthCheckFilterFactory
func CreateHealthCheckFilterFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	return &HealthCheckFilterConfigFactory{
		FilterConfig: config.ParseHealthcheckFilter(conf),
	}, nil
}
