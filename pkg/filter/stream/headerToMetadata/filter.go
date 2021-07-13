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

package headerToMetadata

import (
	"context"
	"mosn.io/mosn/pkg/router"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	_ "mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

type Filter struct {
	rules   []Rule
	handler api.StreamReceiverFilterHandler
}

var _ api.StreamReceiverFilter = (*Filter)(nil)

func NewFilter(factory *FilterFactory) *Filter {
	return &Filter{
		rules: factory.Rules,
	}
}

func (f *Filter) OnDestroy() {}

func (f *Filter) OnReceive(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	m := make(map[string]string, len(f.rules))

	for _, rule := range f.rules {
		value, exist := headers.Get(rule.Header)

		if exist && rule.OnPresent != nil {
			if rule.OnPresent.Value != "" {
				m[rule.OnPresent.Key] = rule.OnPresent.Value
			} else {
				m[rule.OnPresent.Key] = value
			}
		} else if !exist && rule.OnMissing != nil {
			m[rule.OnMissing.Key] = rule.OnMissing.Value
		}

		if rule.Remove {
			headers.Del(rule.Header)
		}
	}

	if len(m) > 0 {
		v, err := variable.Get(ctx, types.VarInternalRouterMeta)
		if err != nil {
			log.DefaultLogger.Errorf("[headerToMetadata] fail to get router meta from variable, err: %v", err)
			return api.StreamFilterContinue
		}

		if v == nil {
			// in case we do not have any metadata
			variable.Set(ctx, types.VarInternalRouterMeta, router.NewMetadataMatchCriteriaImpl(m))
		} else {
			meta, ok := v.(api.MetadataMatchCriteria)
			if !ok {
				log.DefaultLogger.Errorf("[headerToMetadata] variable value is not api.MetadataMatchCriteria")
				return api.StreamFilterContinue
			}

			meta.MergeMatchCriteria(m)
		}
	}

	return api.StreamFilterContinue
}

func (f *Filter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}
