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

package common

import (
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/stats"
)

const FilterMetricsType = "filter"
const RbacEngineMetricsNamespace = "rbac.engine"
const RbacShadowEngineMetricsNamespace = "rbac.shadow_engine"

type RbacStatus struct {
	RawConfig           *v2.RBAC
	EngineMetrics       *stats.Stats
	ShadowEngineMetrics *stats.Stats
}

func NewRbacStatus(rawConfig *v2.RBAC) *RbacStatus {
	return &RbacStatus{
		RawConfig:           rawConfig,
		EngineMetrics:       stats.NewStats(FilterMetricsType, RbacEngineMetricsNamespace),
		ShadowEngineMetrics: stats.NewStats(FilterMetricsType, RbacShadowEngineMetricsNamespace),
	}
}
