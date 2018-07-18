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

package types

import (
	"github.com/alipay/sofa-mosn/internal/api/v2"
)

// Health check interfaces

type FailureType string

const (
	FailureNetwork FailureType = "Network"
	FailurePassive FailureType = "Passive"
	FailureActive  FailureType = "Active"
)

type HealthCheckCb func(host Host, changedState bool)

// A health checker for an upstream cluster
type HealthChecker interface {
	// Start starts health checking, which will continually monitor hosts in upstream cluster
	Start()

	// Stop stops cluster health check. Client can use it to start/stop health check as a heartbeat
	Stop()

	// Add a health check callback, which will be called on a check round-trip is completed for a specified host.
	AddHostCheckCompleteCb(cb HealthCheckCb)

	// Used to update cluster's hosts for health checking
	OnClusterMemberUpdate(hostsAdded []Host, hostDel []Host)

	// Add cluster to healthChecker
	SetCluster(cluster Cluster)
}

// A health check session for an upstream host
type HealthCheckSession interface {
	// Start starts host health check
	Start()

	// Stop stops host health check
	Stop()

	// Set session as unhealthy for a specified reason
	SetUnhealthy(fType FailureType)
}

type HealthCheckHostMonitor interface {
}

// todo: move facotry instance to a factory package
var HealthCheckFactoryInstance HealthCheckerFactory

type HealthCheckerFactory interface {
	New(config v2.HealthCheck) HealthChecker
}
