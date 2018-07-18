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

package proxy

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/alipay/sofa-mosn/pkg/types"
)

type retryState struct {
	retryPolicy     types.RetryPolicy
	requestHeaders  map[string]string
	cluster         types.ClusterInfo
	retryOn         bool
	retiesRemaining uint32
	retryFunc       func()
	retryTimer      *timer
}

func newRetryState(retryPolicy types.RetryPolicy,
	requestHeaders map[string]string, cluster types.ClusterInfo) *retryState {
	rs := &retryState{
		retryPolicy:     retryPolicy,
		requestHeaders:  requestHeaders,
		cluster:         cluster,
		retryOn:         retryPolicy.RetryOn(),
		retiesRemaining: 3,
	}

	if retryPolicy.NumRetries() > rs.retiesRemaining {
		rs.retiesRemaining = retryPolicy.NumRetries()
	}

	return rs
}

func (r *retryState) retry(headers map[string]string, reason types.StreamResetReason, doRetry func()) types.RetryCheckStatus {
	r.reset()

	check := r.shouldRetry(headers, reason)

	if check != 0 {
		return check
	}

	r.retryTimer = r.scheduleRetry(doRetry)

	return 0
}

func (r *retryState) shouldRetry(headers map[string]string, reason types.StreamResetReason) types.RetryCheckStatus {
	if r.retiesRemaining == 0 {
		return types.NoRetry
	}

	r.retiesRemaining--

	if !r.doRetryCheck(headers, reason) {
		return types.NoRetry
	}

	if r.cluster.ResourceManager().Retries().CanCreate() {
		r.cluster.Stats().UpstreamRequestRetryOverflow.Inc(1)

		return types.RetryOverflow
	}

	return types.ShouldRetry
}

func (r *retryState) scheduleRetry(doRetry func()) *timer {
	r.retryFunc = doRetry
	r.cluster.ResourceManager().Retries().Increase()
	r.cluster.Stats().UpstreamRequestRetry.Inc(1)

	// todo: use backoff alth
	timeout := rand.Intn(10)
	timer := newTimer(doRetry, time.Duration(timeout)*time.Second)
	timer.start()

	return timer
}

func (r *retryState) doRetryCheck(headers map[string]string, reason types.StreamResetReason) bool {
	if reason == types.StreamOverflow {
		return false
	}

	if r.retryOn {
		if code, ok := headers[types.HeaderStatus]; ok {
			codeValue, _ := strconv.Atoi(code)

			return codeValue >= 500
		}

		// todo: more conditions
	}

	return false
}

func (r *retryState) reset() {
	if r.retryFunc != nil {
		r.cluster.ResourceManager().Retries().Decrease()
		r.retryFunc = nil
	}
}
