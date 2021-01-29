// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package circuitbreaker

import (
	"fmt"

	"github.com/alibaba/sentinel-golang/util"
)

// Strategy represents the strategy of circuit breaker.
// Each strategy is associated with one rule type.
type Strategy uint32

const (
	// SlowRequestRatio strategy changes the circuit breaker state based on slow request ratio
	SlowRequestRatio Strategy = iota
	// ErrorRatio strategy changes the circuit breaker state based on error request ratio
	ErrorRatio
	// ErrorCount strategy changes the circuit breaker state based on error amount
	ErrorCount
)

func (s Strategy) String() string {
	switch s {
	case SlowRequestRatio:
		return "SlowRequestRatio"
	case ErrorRatio:
		return "ErrorRatio"
	case ErrorCount:
		return "ErrorCount"
	default:
		return "Undefined"
	}
}

// Rule encompasses the fields of circuit breaking rule.
type Rule struct {
	// unique id
	Id string `json:"id,omitempty"`
	// resource name
	Resource string   `json:"resource"`
	Strategy Strategy `json:"strategy"`
	// RetryTimeoutMs represents recovery timeout (in milliseconds) before the circuit breaker opens.
	// During the open period, no requests are permitted until the timeout has elapsed.
	// After that, the circuit breaker will transform to half-open state for trying a few "trial" requests.
	RetryTimeoutMs uint32 `json:"retryTimeoutMs"`
	// MinRequestAmount represents the minimum number of requests (in an active statistic time span)
	// that can trigger circuit breaking.
	MinRequestAmount uint64 `json:"minRequestAmount"`
	// StatIntervalMs represents statistic time interval of the internal circuit breaker (in ms).
	// Currently the statistic interval is collected by sliding window.
	StatIntervalMs uint32 `json:"statIntervalMs"`
	// StatSlidingWindowBucketCount represents the bucket count of statistic sliding window.
	// The statistic will be more precise as the bucket count increases, but the memory cost increases too.
	// The following must be true — “StatIntervalMs % StatSlidingWindowBucketCount == 0”,
	// otherwise StatSlidingWindowBucketCount will be replaced by 1.
	// If it is not set, default value 1 will be used.
	StatSlidingWindowBucketCount uint32 `json:"statSlidingWindowBucketCount"`
	// MaxAllowedRtMs indicates that any invocation whose response time exceeds this value (in ms)
	// will be recorded as a slow request.
	// MaxAllowedRtMs only takes effect for SlowRequestRatio strategy
	MaxAllowedRtMs uint64 `json:"maxAllowedRtMs"`
	// Threshold represents the threshold of circuit breaker.
	// for SlowRequestRatio, it represents the max slow request ratio
	// for ErrorRatio, it represents the max error request ratio
	// for ErrorCount, it represents the max error request count
	Threshold float64 `json:"threshold"`
}

func (r *Rule) String() string {
	// fallback string
	return fmt.Sprintf("{id=%s, resource=%s, strategy=%s, RetryTimeoutMs=%d, MinRequestAmount=%d, StatIntervalMs=%d, StatSlidingWindowBucketCount=%d, MaxAllowedRtMs=%d, Threshold=%f}",
		r.Id, r.Resource, r.Strategy, r.RetryTimeoutMs, r.MinRequestAmount, r.StatIntervalMs, r.StatSlidingWindowBucketCount, r.MaxAllowedRtMs, r.Threshold)
}

func (r *Rule) isStatReusable(newRule *Rule) bool {
	if newRule == nil {
		return false
	}
	return r.Resource == newRule.Resource && r.Strategy == newRule.Strategy && r.StatIntervalMs == newRule.StatIntervalMs &&
		r.StatSlidingWindowBucketCount == newRule.StatSlidingWindowBucketCount
}

func (r *Rule) ResourceName() string {
	return r.Resource
}

func (r *Rule) isEqualsToBase(newRule *Rule) bool {
	if newRule == nil {
		return false
	}
	return r.Resource == newRule.Resource && r.Strategy == newRule.Strategy && r.RetryTimeoutMs == newRule.RetryTimeoutMs &&
		r.MinRequestAmount == newRule.MinRequestAmount && r.StatIntervalMs == newRule.StatIntervalMs && r.StatSlidingWindowBucketCount == newRule.StatSlidingWindowBucketCount
}

func (r *Rule) isEqualsTo(newRule *Rule) bool {
	if !r.isEqualsToBase(newRule) {
		return false
	}

	switch newRule.Strategy {
	case SlowRequestRatio:
		return r.MaxAllowedRtMs == newRule.MaxAllowedRtMs && util.Float64Equals(r.Threshold, newRule.Threshold)
	case ErrorRatio:
		return util.Float64Equals(r.Threshold, newRule.Threshold)
	case ErrorCount:
		return util.Float64Equals(r.Threshold, newRule.Threshold)
	default:
		return false
	}
}

func getRuleStatSlidingWindowBucketCount(r *Rule) uint32 {
	interval := r.StatIntervalMs
	bucketCount := r.StatSlidingWindowBucketCount
	if bucketCount == 0 {
		bucketCount = 1
	} else {
		if interval%bucketCount != 0 {
			bucketCount = 1
		}
	}
	return bucketCount
}
