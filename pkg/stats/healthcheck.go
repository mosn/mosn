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

package stats

import (
	"fmt"

	"github.com/alipay/sofa-mosn/pkg/types"
)

// HealthCheckType represents health check metrics type
const HealthCheckType = "healthcheck"

// health check metrics key
const (
	HealthCheckAttempt        = "health_check_attempt"
	HealthCheckSuccess        = "health_check_success"
	HealthCheckFailure        = "health_check_failure"
	HealthCheckPassiveFailure = "health_check_passive_failure"
	HealthCheckNetworkFailure = "health_check_network_failure"
	HealthCheckVeirfyCluster  = "health_check_verify_cluster"
	HealthCheckHealthy        = "health_check_healty"
)

// NewHealthStats returns a stats with namespace prefix service
func NewHealthStats(serviceName string) types.Metrics {
	namespace := fmt.Sprintf("service.%s", serviceName)
	return NewStats(HealthCheckType, namespace)
}
