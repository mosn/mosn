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
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/stats"
)

/*
 Metric:
	"filter.rbac.engine": {
		"allowed": Counter, 		  // total allowed count in engine
		"denied": Counter, 			  // total denied count in engine
		"${policy1_name}": Counter,   // policy1 hit count in engine
		"${policy2_name}": Counter,   // policy2 hit count in engine
		"${policy3_name}": Counter,   // policy3 hit count in engine
		...
	},
	"filter.rbac.shadow_engine": {
		"allowed": Counter, 		  // total allowed count in shadow engine
		"denied": Counter, 			  // total denied count in shadow engine
		"${policy1_name}": Counter,   // policy1 hit count in shadow engine
		"${policy2_name}": Counter,   // policy2 hit count in shadow engine
		"${policy3_name}": Counter,   // policy3 hit count in shadow engine
		...
	}
 */

const FilterMetricsType = "filter"
const EngineMetricsNamespace = "rbac.engine"
const ShadowEngineMetricsNamespace = "rbac.shadow_engine"
const AllowedMetricsNamespace = "allowed"
const DeniedMetricsNamespace = "denied"

type RbacStatus struct {
	RawConfig           *v2.RBAC
	EngineMetrics       *stats.Stats
	ShadowEngineMetrics *stats.Stats
}

func NewRbacStatus(rawConfig *v2.RBAC) *RbacStatus {
	return &RbacStatus{
		RawConfig:           rawConfig,
		EngineMetrics:       stats.NewStats(FilterMetricsType, EngineMetricsNamespace),
		ShadowEngineMetrics: stats.NewStats(FilterMetricsType, ShadowEngineMetricsNamespace),
	}
}
