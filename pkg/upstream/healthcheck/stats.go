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

package healthcheck

import (
	"github.com/alipay/sofa-mosn/pkg/stats"
	metrics "github.com/rcrowley/go-metrics"
)

type healthCheckStats struct {
	attempt metrics.Counter
	// total counts for check health returns ok
	success metrics.Counter
	// total counts for check health returns fail
	failure metrics.Counter
	// total counts for check health returns fail with reason
	passiveFailure metrics.Counter
	activeFailure  metrics.Counter
	networkFailure metrics.Counter
	verifyCluster  metrics.Counter
	healthy        metrics.Gauge
}

func newHealthCheckStats(namespace string) *healthCheckStats {
	m := stats.NewHealthStats(namespace)
	return &healthCheckStats{
		attempt:        m.Counter(stats.HealthCheckAttempt),
		success:        m.Counter(stats.HealthCheckSuccess),
		failure:        m.Counter(stats.HealthCheckFailure),
		activeFailure:  m.Counter(stats.HealthCheckActiveFailure),
		passiveFailure: m.Counter(stats.HealthCheckPassiveFailure),
		networkFailure: m.Counter(stats.HealthCheckNetworkFailure),
		verifyCluster:  m.Counter(stats.HealthCheckVeirfyCluster),
		healthy:        m.Gauge(stats.HealthCheckHealthy),
	}
}
