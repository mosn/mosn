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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	rbac_common "github.com/alipay/sofa-mosn/pkg/filter/stream/rbac/common"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// rbacFilter is an implement of types.StreamReceiverFilter
type rbacFilter struct {
	context      context.Context
	cb           types.StreamReceiverFilterCallbacks
	config       *v2.RBAC
	engine       *rbac_common.RoleBasedAccessControlEngine
	shadowEngine *rbac_common.RoleBasedAccessControlEngine
}

func NewFilter(context context.Context, config *v2.RBAC, engine *rbac_common.RoleBasedAccessControlEngine,
	shadowEngine *rbac_common.RoleBasedAccessControlEngine) types.StreamReceiverFilter {
	return &rbacFilter{
		context:      context,
		config:       config,
		engine:       engine,
		shadowEngine: shadowEngine,
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
	rbacConf, _ := json.Marshal(f.config)
	log.DefaultLogger.Debugf(string(rbacConf))

	// rbac shadow engine handle
	_, matchPolicyName := f.shadowEngine.Allowed(f.cb, headers)
	if matchPolicyName != "" {
		// TODO: record metric log
		log.DefaultLogger.Debugf("shoadow engine hit, policy name: %s", matchPolicyName)
	}

	// rbac engine handle
	allowed, matchPolicyName := f.engine.Allowed(f.cb, headers)
	if matchPolicyName != "" {
		// TODO: record metric log
		log.DefaultLogger.Debugf("engine hit, policy name: %s", matchPolicyName)
	}
	if !allowed {
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
