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
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/protocol/http"
	"mosn.io/pkg/variable"
)

type retryState struct {
	retryPolicy      api.RetryPolicy
	requestHeaders   types.HeaderMap // TODO: support retry policy by header
	cluster          types.ClusterInfo
	retryOn          bool
	retiesRemaining  uint32
	upstreamProtocol types.ProtocolName
}

func newRetryState(retryPolicy api.RetryPolicy,
	requestHeaders api.HeaderMap, cluster types.ClusterInfo, proto api.ProtocolName) *retryState {
	rs := &retryState{
		retryPolicy:      retryPolicy,
		requestHeaders:   requestHeaders,
		cluster:          cluster,
		retryOn:          retryPolicy.RetryOn(),
		retiesRemaining:  3,
		upstreamProtocol: proto,
	}

	if retryPolicy.NumRetries() > rs.retiesRemaining {
		rs.retiesRemaining = retryPolicy.NumRetries()
	}

	return rs
}

func (r *retryState) retry(ctx context.Context, headers api.HeaderMap, reason types.StreamResetReason) api.RetryCheckStatus {
	r.reset()

	check := r.shouldRetry(ctx, headers, reason)

	if check != 0 {
		return check
	}

	r.cluster.ResourceManager().Retries().Increase()
	r.cluster.Stats().UpstreamRequestRetry.Inc(1)

	return 0
}

func (r *retryState) shouldRetry(ctx context.Context, headers api.HeaderMap, reason types.StreamResetReason) api.RetryCheckStatus {
	if r.retiesRemaining == 0 {
		return api.NoRetry
	}

	r.retiesRemaining--

	if !r.doRetryCheck(ctx, headers, reason) {
		return api.NoRetry
	}

	if !r.cluster.ResourceManager().Retries().CanCreate() {
		r.cluster.Stats().UpstreamRequestRetryOverflow.Inc(1)

		return api.RetryOverflow
	}

	return api.ShouldRetry
}

func (r *retryState) doRetryCheck(ctx context.Context, headers types.HeaderMap, reason types.StreamResetReason) bool {
	if ctx != nil {
		if disable, err := variable.Get(ctx, types.VarProxyDisableRetry); err == nil {
			if retryDisable, ok := disable.(bool); ok && retryDisable {
				return false
			}
		}
	}

	if reason == types.StreamOverflow {
		return false
	}

	if r.retryOn {
		if ctx != nil {
			code, err := protocol.MappingHeaderStatusCode(ctx, r.upstreamProtocol, headers)
			if err == nil {
				codes := r.retryPolicy.RetryableStatusCodes()
				if len(codes) > 0 {
					for _, it := range codes {
						if code == int(it) {
							return true
						}
					}
					return false
				}
				return code >= http.InternalServerError
			}
		}
		if reason == types.StreamConnectionFailed {
			return true
		}

		if reason == types.UpstreamPerTryTimeout {
			return true
		}

		if reason == types.StreamConnectionTermination {
			return true
		}
		// more policy
	} else {
		// default support connectionFailed retry
		if reason == types.StreamConnectionFailed {
			return true
		}
	}

	return false
}

func (r *retryState) reset() {
	r.cluster.ResourceManager().Retries().Decrease()
}
