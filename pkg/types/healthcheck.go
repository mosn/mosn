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

// FailureType is the type of a failure
type FailureType string

// Failure types
const (
	FailureNetwork FailureType = "Network"
	FailurePassive FailureType = "Passive"
	FailureActive  FailureType = "Active"
)

// HealthCheckCb is the health check's callback function
type HealthCheckCb func(host Host, changedState bool, isHealthy bool)

// HealthChecker is a framework for connection management
// When NewCluster is called, and the config contains health check related, mosn will create
// a cluster with health check to make sure load balance always choose the "good" host
type HealthChecker interface {
	// Start makes health checker running
	Start()
	// Stop terminates health checker
	Stop()
	// AddHostCheckCompleteCb adds a new callback for health check
	AddHostCheckCompleteCb(cb HealthCheckCb)
	// SetHealthCheckerHostSet reset the health checker's hostset
	SetHealthCheckerHostSet(HostSet)
}

// HealthCheckSession is an interface for health check logic
// The health checker framework support register different session for different protocol.
// The default session implementation is tcp dial, for all non-registered protocol.
type HealthCheckSession interface {
	// CheckHealth returns true if session checks the server is ok, or returns false
	CheckHealth() bool
	// OnTimeout is called when a check health does not returned after timeout duration
	OnTimeout()
}

// HealthCheckSessionFactory creates a HealthCheckSession
type HealthCheckSessionFactory interface {
	NewSession(cfg map[string]interface{}, host Host) HealthCheckSession
}
