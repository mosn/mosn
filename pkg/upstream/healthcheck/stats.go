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
	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/mosn/pkg/metrics"
)

type healthCheckStats struct {
	attempt gometrics.Counter
	// total counts for check health returns ok
	success gometrics.Counter
	// total counts for check health returns fail
	failure gometrics.Counter
	// total counts for check health returns fail with reason
	passiveFailure gometrics.Counter
	activeFailure  gometrics.Counter
	networkFailure gometrics.Counter
	verifyCluster  gometrics.Counter
	healthy        gometrics.Gauge
}

func newHealthCheckStats(namespace string) *healthCheckStats {
	m := metrics.NewHealthStats(namespace)
	return &healthCheckStats{
		attempt:        m.Counter(metrics.HealthCheckAttempt),
		success:        m.Counter(metrics.HealthCheckSuccess),
		failure:        m.Counter(metrics.HealthCheckFailure),
		activeFailure:  m.Counter(metrics.HealthCheckActiveFailure),
		passiveFailure: m.Counter(metrics.HealthCheckPassiveFailure),
		networkFailure: m.Counter(metrics.HealthCheckNetworkFailure),
		verifyCluster:  m.Counter(metrics.HealthCheckVeirfyCluster),
		healthy:        m.Gauge(metrics.HealthCheckHealthy),
	}
}
