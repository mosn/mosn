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
	"fmt"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/rcrowley/go-metrics"
)

/*
 Metric:
	"filter.rbac.<version>": {
		"engine.allowed":                         Counter, 	 // total allowed count in engine
		"engine.denied":                          Counter, 	 // total denied count in engine
		"engine.hits.${policy1_name}":            Counter,   // policy1 hit count in engine
		"engine.hits.${policy2_name}":            Counter,   // policy2 hit count in engine
		"engine.hits.${policy3_name}":            Counter,   // policy3 hit count in engine
		"shadow_engine.allowed":                  Counter,   // total allowed count in shadow engine
		"shadow_engine.denied":                   Counter,   // total denied count in shadow engine
		"shadow_engine.hits.${policy1_name}":     Counter,   // policy1 hit count in shadow engine
		"shadow_engine.hits.${policy2_name}":     Counter,   // policy2 hit count in shadow engine
		"shadow_engine.hits.${policy3_name}":     Counter,   // policy3 hit count in shadow engine
	},
 */

const (
	FilterMetricsType           = "filter.rbac"
	EngineAllowedTotal          = "engine.allowed"
	EngineDeniedTotal           = "engine.denied"
	EnginePoliciesMetrics       = "engine.hits.%s"
	ShadowEngineAllowedTotal    = "shadow_engine.allowed"
	ShadowEngineDeniedTotal     = "shadow_engine.denied"
	ShadowEnginePoliciesMetrics = "shadow_engine.hits.%s"
)

// Status contains the metric logs for rbac filter
type Status struct {
	EngineAllowedTotal          metrics.Counter
	EngineDeniedTotal           metrics.Counter
	EnginePoliciesMetrics       map[string]metrics.Counter
	ShadowEngineAllowedTotal    metrics.Counter
	ShadowEngineDeniedTotal     metrics.Counter
	ShadowEnginePoliciesMetrics map[string]metrics.Counter
}

// NewStatus return the instance of RbacStatus
func NewStatus(config *v2.RBAC) *Status {
	status := &Status{}
	s := stats.NewStats(FilterMetricsType, config.Version)

	status.EngineAllowedTotal = s.Counter(EngineAllowedTotal)
	status.EngineDeniedTotal = s.Counter(EngineDeniedTotal)
	status.ShadowEngineAllowedTotal = s.Counter(ShadowEngineAllowedTotal)
	status.ShadowEngineDeniedTotal = s.Counter(ShadowEngineDeniedTotal)

	status.EnginePoliciesMetrics = make(map[string]metrics.Counter)
	for name := range config.Rules.Policies {
		status.EnginePoliciesMetrics[name] = s.Counter(fmt.Sprintf(EnginePoliciesMetrics, name))
	}

	status.ShadowEnginePoliciesMetrics = make(map[string]metrics.Counter)
	for name := range config.ShadowRules.Policies {
		status.ShadowEnginePoliciesMetrics[name] = s.Counter(fmt.Sprintf(ShadowEnginePoliciesMetrics, name))
	}

	return status
}
