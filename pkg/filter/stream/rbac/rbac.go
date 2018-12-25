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

	"github.com/alipay/sofa-mosn/pkg/filter/stream/rbac/common"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/http"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// supported protocols
const (
	httpProtocol    = 1
	sofaRPCProtocol = 2
)

// rbacFilter is an implement of types.StreamReceiverFilter
type rbacFilter struct {
	context      context.Context
	cb           types.StreamReceiverFilterCallbacks
	status       *Status
	protocol     int
	engine       *common.RoleBasedAccessControlEngine
	shadowEngine *common.RoleBasedAccessControlEngine
}

// NewFilter return the instance of rbac filter for each request
func NewFilter(context context.Context, filterConfigFactory *filterConfigFactory) types.StreamReceiverFilter {
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
func (f *rbacFilter) OnDecodeHeaders(headers types.HeaderMap, endStream bool) (streamHeadersFilterStatus types.StreamHeadersFilterStatus) {
	defer func() {
		if err := recover(); err != nil {
			log.DefaultLogger.Errorf("recover from rbac filter, error: %v", err)
			streamHeadersFilterStatus = types.StreamHeadersFilterContinue
		}
	}()

	// TODO: protocol check
	//if !f.IsSupported(headers) {
	//	return types.StreamHeadersFilterContinue
	//}

	// rbac shadow engine handle
	allowed, matchPolicyName := f.shadowEngine.Allowed(f.cb, headers)
	if matchPolicyName != "" {
		log.DefaultLogger.Debugf("shoadow engine hit, policy name: %s", matchPolicyName)
		// record metric log
		f.status.ShadowEnginePoliciesMetrics[matchPolicyName].Inc(1)
		if allowed {
			f.status.ShadowEngineAllowedTotal.Inc(1)
		} else {
			f.status.ShadowEngineDeniedTotal.Inc(1)
		}
	}

	// rbac engine handle
	allowed, matchPolicyName = f.engine.Allowed(f.cb, headers)
	if matchPolicyName != "" {
		log.DefaultLogger.Debugf("engine hit, policy name: %s", matchPolicyName)
		// record metric log
		f.status.EnginePoliciesMetrics[matchPolicyName].Inc(1)
		if allowed {
			f.status.EngineAllowedTotal.Inc(1)
		} else {
			f.status.EngineDeniedTotal.Inc(1)
		}
	}
	if !allowed {
		f.Intercept(headers)
		return types.StreamHeadersFilterStop
	}

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

// TODO: make sure protocol inspection is correct
func (f *rbacFilter) IsSupported(header types.HeaderMap) bool {
	switch header.(type) {
	case http.RequestHeader:
		f.protocol = httpProtocol
		return true
	case sofarpc.SofaRpcCmd:
		f.protocol = sofaRPCProtocol
		return true
	default:
		return false
	}
}

func (f *rbacFilter) Intercept(headers types.HeaderMap) {
	f.cb.SendHijackReply(types.PermissionDeniedCode, headers)
	return
}
