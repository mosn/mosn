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

package cluster

import (
	"sync/atomic"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

// default value is zero
const (
	DefaultMaxConnections     uint64 = 0
	DefaultMaxPendingRequests uint64 = 0
	DefaultMaxRequests        uint64 = 0
	DefaultMaxRetries         uint64 = 0
)

// ResourceManager
type resourcemanager struct {
	connections     *resource
	pendingRequests *resource
	requests        *resource
	retries         *resource
}

func NewResourceManager(circuitBreakers v2.CircuitBreakers) types.ResourceManager {
	maxConnections := DefaultMaxConnections
	maxPendingRequests := DefaultMaxPendingRequests
	maxRequests := DefaultMaxRequests
	maxRetries := DefaultMaxRetries

	// note: we don't support group cb by priority
	if circuitBreakers.Thresholds != nil && len(circuitBreakers.Thresholds) > 0 {
		maxConnections = uint64(circuitBreakers.Thresholds[0].MaxConnections)
		maxPendingRequests = uint64(circuitBreakers.Thresholds[0].MaxPendingRequests)
		maxRequests = uint64(circuitBreakers.Thresholds[0].MaxRequests)
		maxRetries = uint64(circuitBreakers.Thresholds[0].MaxRetries)
	}

	return &resourcemanager{
		connections: &resource{
			max: maxConnections,
		},
		pendingRequests: &resource{
			max: maxPendingRequests,
		},
		requests: &resource{
			max: maxRequests,
		},
		retries: &resource{
			max: maxRetries,
		},
	}
}

func (rm *resourcemanager) Connections() types.Resource {
	return rm.connections
}

func (rm *resourcemanager) PendingRequests() types.Resource {
	return rm.pendingRequests
}

func (rm *resourcemanager) Requests() types.Resource {
	return rm.requests
}

func (rm *resourcemanager) Retries() types.Resource {
	return rm.retries
}

func updateResourceValue(oldRM, newRM types.ResourceManager) {
	nrm := newRM.(*resourcemanager)
	orm := oldRM.(*resourcemanager)

	orm.connections.max = nrm.connections.max
	orm.pendingRequests.max = nrm.pendingRequests.max
	orm.requests.max = nrm.requests.max
	orm.retries.max = nrm.retries.max
}

// Resource
type resource struct {
	current int64
	max     uint64
}

func (r *resource) CanCreate() bool {
	if r.max == 0 {
		return true
	}
	curValue := atomic.LoadInt64(&r.current)

	if curValue < 0 {
		return true
	}

	return uint64(curValue) < r.Max()
}

func (r *resource) Increase() {
	if r.max != 0 {
		atomic.AddInt64(&r.current, 1)
	}
}

func (r *resource) Decrease() {
	if r.max != 0 {
		atomic.AddInt64(&r.current, -1)
	}
}

func (r *resource) Max() uint64 {
	return r.max
}

func (r *resource) Cur() int64 {
	return r.current
}

func (r *resource) UpdateCur(cur int64) {
	r.current = cur
}
