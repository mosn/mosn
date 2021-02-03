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

package metrics

import (
	"mosn.io/mosn/pkg/types"
)

// HealthCheckType represents health check metrics type
const HealthCheckType = "healthcheck"

// health check metrics key
const (
	HealthCheckAttempt        = "attempt"
	HealthCheckSuccess        = "success"
	HealthCheckFailure        = "failure"
	HealthCheckActiveFailure  = "active_failure"
	HealthCheckPassiveFailure = "passive_failure"
	HealthCheckNetworkFailure = "network_failure"
	HealthCheckVeirfyCluster  = "verify_cluster"
	HealthCheckHealthy        = "healty"
)

// NewHealthStats returns a stats with namespace prefix service
func NewHealthStats(serviceName string) types.Metrics {
	metrics, _ := NewMetrics(HealthCheckType, map[string]string{"service": serviceName})
	return metrics
}
