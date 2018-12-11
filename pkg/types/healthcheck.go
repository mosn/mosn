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

type HealthChecker interface {
	Start()
	Stop()
	AddHostCheckCompleteCb(cb HealthCheckCb)
	OnClusterMemberUpdate(hostsAdded []Host, hostDel []Host)
}

type HealthCheckSession interface {
	CheckHealth() bool
	OnTimeout()
}

type HealthCheckSessionFactory interface {
	NewSession(cfg map[string]interface{}, host Host) HealthCheckSession
}
