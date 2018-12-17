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

package rbac

import (
	"context"
	"encoding/json"

	"github.com/alipay/sofa-mosn/pkg/filter/stream/rbac/common"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// rbacFilter is an implement of types.StreamReceiverFilter
type rbacFilter struct {
	context      context.Context
	cb           types.StreamReceiverFilterCallbacks
	status       *RbacStatus
	engine       *common.RoleBasedAccessControlEngine
	shadowEngine *common.RoleBasedAccessControlEngine
}

func NewFilter(context context.Context, filterConfigFactory *FilterConfigFactory) types.StreamReceiverFilter {
	return &rbacFilter{
		context:      context,
		status:       filterConfigFactory.Status,
		engine:       filterConfigFactory.Engine,
		shadowEngine: filterConfigFactory.ShadowEngine,
	}
}

// ReadPerRouteConfig makes route-level configuration override filter-level configuration
func (f *rbacFilter) ReadPerRouteConfig(cfg map[string]interface{}) {
	// TODO: parse per route configuration implement
	return
}

// filter implementation
func (f *rbacFilter) OnDecodeHeaders(headers types.HeaderMap, endStream bool) types.StreamHeadersFilterStatus {
	defer func() {
		if err := recover(); err != nil {
			log.DefaultLogger.Errorf("recover from rbac filter, error: %v", err)
		}
	}()

	// print http header
	headers.Range(func(key, value string) bool {
		log.DefaultLogger.Debugf("Key: %s, Value: %s\n", key, value)
		return true
	})

	// print RBAC configuration
	rbacConf, _ := json.Marshal(f.status.RawConfig)
	log.DefaultLogger.Debugf(string(rbacConf))

	// rbac shadow engine handle
	allowed, matchPolicyName := f.shadowEngine.Allowed(f.cb, headers)
	if matchPolicyName != "" {
		log.DefaultLogger.Debugf("shoadow engine hit, policy name: %s", matchPolicyName)
		// TODO: record metric log
		f.status.ShadowEngineMetrics.Counter(matchPolicyName).Inc(1)
		if allowed {
			f.status.ShadowEngineMetrics.Counter(AllowedMetricsNamespace).Inc(1)
		} else {
			f.status.ShadowEngineMetrics.Counter(DeniedMetricsNamespace).Inc(1)
		}
	}

	// rbac engine handle
	allowed, matchPolicyName = f.engine.Allowed(f.cb, headers)
	if matchPolicyName != "" {
		log.DefaultLogger.Debugf("engine hit, policy name: %s", matchPolicyName)
		// TODO: record metric log
		f.status.EngineMetrics.Counter(matchPolicyName).Inc(1)
		if allowed {
			f.status.EngineMetrics.Counter(AllowedMetricsNamespace).Inc(1)
		} else {
			f.status.EngineMetrics.Counter(DeniedMetricsNamespace).Inc(1)
		}
	}
	if !allowed {
		return types.StreamHeadersFilterStop
	}

	log.DefaultLogger.Debugf("Metrics: %v", stats.GetMetricsData(FilterMetricsType)[EngineMetricsNamespace])
	log.DefaultLogger.Debugf("Metrics: %v", stats.GetMetricsData(FilterMetricsType)[ShadowEngineMetricsNamespace])

	return types.StreamHeadersFilterContinue
}

// http body handler
func (f *rbacFilter) OnDecodeData(buf types.IoBuffer, endStream bool) types.StreamDataFilterStatus {
	return types.StreamDataFilterContinue
}

func (f *rbacFilter) OnDecodeTrailers(trailers types.HeaderMap) types.StreamTrailersFilterStatus {
	//do filter
	return types.StreamTrailersFilterContinue
}

func (f *rbacFilter) SetDecoderFilterCallbacks(cb types.StreamReceiverFilterCallbacks) {
	f.cb = cb
}

func (f *rbacFilter) OnDestroy() {}
