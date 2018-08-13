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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

const (
	// note: 10x bigger than envoy default value
	DefaultMaxConnections     = uint64(10240)
	DefaultMaxPendingRequests = uint64(10240)
	DefaultMaxRequests        = uint64(10240)
	DefaultMaxRetries         = uint64(3)
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

// Resource
type resource struct {
	current int64
	max     uint64
}

func (r *resource) CanCreate() bool {
	curValue := atomic.LoadInt64(&r.current)

	if curValue < 0 {
		return true
	}

	return uint64(curValue) < r.Max()
}

func (r *resource) Increase() {
	atomic.AddInt64(&r.current, 1)
}

func (r *resource) Decrease() {
	atomic.AddInt64(&r.current, -1)
}

func (r *resource) Max() uint64 {
	return r.max
}
