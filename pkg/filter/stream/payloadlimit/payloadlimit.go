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

package payloadlimit

import (
	"context"

	"encoding/json"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/buffer"
)

type payloadLimitConfig struct {
	maxEntitySize int32
	status        int32
}

// streamPayloadLimitFilter is an implement of StreamReceiverFilter
type streamPayloadLimitFilter struct {
	ctx     context.Context
	handler api.StreamReceiverFilterHandler
	config  *payloadLimitConfig
	headers api.HeaderMap
}

func NewFilter(ctx context.Context, cfg *v2.StreamPayloadLimit) api.StreamReceiverFilter {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("create a new payload limit filter")
	}
	return &streamPayloadLimitFilter{
		ctx:    ctx,
		config: makePayloadLimitConfig(cfg),
	}
}

func makePayloadLimitConfig(cfg *v2.StreamPayloadLimit) *payloadLimitConfig {
	config := &payloadLimitConfig{
		maxEntitySize: cfg.MaxEntitySize,
		status:        cfg.HttpStatus,
	}
	return config
}
func parseStreamPayloadLimitConfig(c interface{}) (*payloadLimitConfig, bool) {
	conf := make(map[string]interface{})
	b, err := json.Marshal(c)
	if err != nil {
		log.DefaultLogger.Errorf("config is not a json, %v", err)
		return nil, false
	}
	json.Unmarshal(b, &conf)
	cfg, err := ParseStreamPayloadLimitFilter(conf)
	if err != nil {
		log.DefaultLogger.Errorf("config is not stream payload limit", err)
		return nil, false
	}
	return makePayloadLimitConfig(cfg), true
}

// ReadPerRouteConfig makes route-level configuration override filter-level configuration
func (f *streamPayloadLimitFilter) ReadPerRouteConfig(cfg map[string]interface{}) {
	if cfg == nil {
		return
	}
	if payloadLimit, ok := cfg[v2.PayloadLimit]; ok {
		if config, ok := parseStreamPayloadLimitConfig(payloadLimit); ok {
			if log.DefaultLogger.GetLogLevel() >= log.INFO {
				log.DefaultLogger.Infof("use router config to replace stream filter config, config: %v", payloadLimit)
			}
			f.config = config
		}
	}
}

func (f *streamPayloadLimitFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *streamPayloadLimitFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("payload limit stream do receive headers")
	}
	if route := f.handler.Route(); route != nil {
		// TODO: makes ReadPerRouteConfig as the StreamReceiverFilter's function
		f.ReadPerRouteConfig(route.RouteRule().PerFilterConfig())
	}
	f.headers = headers

	// buf is nil means request method is GET?
	if buf != nil && f.config.maxEntitySize != 0 {
		if buf.Len() > int(f.config.maxEntitySize) {
			if log.Proxy.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("payload size too large,data size = %d ,limit = %d",
					buf.Len(), f.config.maxEntitySize)
			}
			f.handler.RequestInfo().SetResponseFlag(api.ReqEntityTooLarge)
			f.handler.SendHijackReply(int(f.config.status), f.headers)
			return api.StreamFilterStop
		}
	}

	return api.StreamFilterContinue
}

func (f *streamPayloadLimitFilter) OnDestroy() {}
