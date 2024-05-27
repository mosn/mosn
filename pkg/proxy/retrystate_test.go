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
	"testing"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

func doNothing() {}

type fakeClusterInfo struct {
	types.ClusterInfo
	mgr types.ResourceManager
}

func (ci *fakeClusterInfo) ResourceManager() types.ResourceManager {
	return ci.mgr
}
func (ci *fakeClusterInfo) Stats() *types.ClusterStats {
	return &types.ClusterStats{
		UpstreamRequestRetryOverflow: metrics.NewCounter(),
		UpstreamRequestRetry:         metrics.NewCounter(),
	}
}

type fakeResourceManager struct {
	types.ResourceManager
}

func (mgr *fakeResourceManager) Retries() types.Resource {
	return &fakeResource{}
}

type fakeResource struct{}

func (r *fakeResource) CanCreate() bool {
	return true
}
func (r *fakeResource) Increase()           {}
func (r *fakeResource) Decrease()           {}
func (r *fakeResource) Max() uint64         { return 10 }
func (r *fakeResource) Cur() int64          { return 5 }
func (r *fakeResource) UpdateCur(cur int64) {}

func TestRetryState(t *testing.T) {
	rcfg := &v2.Router{}
	pcfg := &v2.RetryPolicy{
		RetryPolicyConfig: v2.RetryPolicyConfig{
			RetryOn:    true,
			NumRetries: 10,
		},
		RetryTimeout: time.Second,
	}
	rcfg.Route = v2.RouteAction{}
	rcfg.Route.RetryPolicy = pcfg
	r, _ := router.NewRouteRuleImplBase(nil, rcfg)
	policy := r.Policy().RetryPolicy()
	clusterInfo := &fakeClusterInfo{
		mgr: &fakeResourceManager{},
	}
	rs := newRetryState(policy, nil, clusterInfo, protocol.HTTP1)
	variable.Register(variable.NewStringVariable(types.VarHeaderStatus, nil, nil, variable.DefaultStringSetter, 0))
	ctx1 := variable.NewVariableContext(context.Background())
	variable.SetString(ctx1, types.VarHeaderStatus, "200")
	ctx2 := variable.NewVariableContext(context.Background())
	variable.SetString(ctx2, types.VarHeaderStatus, "500")
	testcases := []struct {
		ctx      context.Context
		Reason   types.StreamResetReason
		Expected api.RetryCheckStatus
	}{
		{nil, types.StreamConnectionFailed, api.ShouldRetry},
		{ctx2, "", api.ShouldRetry},
		{ctx1, "", api.NoRetry},
	}
	for i, tc := range testcases {
		if rs.retry(tc.ctx, nil, tc.Reason) != tc.Expected {
			t.Errorf("#%d retry state failed", i)
		}
	}
}

func TestRetryConnetionFailed(t *testing.T) {
	rcfg := &v2.Router{}
	pcfg := &v2.RetryPolicy{
		RetryPolicyConfig: v2.RetryPolicyConfig{
			RetryOn:    false,
			NumRetries: 10,
		},
		RetryTimeout: time.Second,
	}
	rcfg.Route = v2.RouteAction{}
	rcfg.Route.RetryPolicy = pcfg
	r, _ := router.NewRouteRuleImplBase(nil, rcfg)
	policy := r.Policy().RetryPolicy()
	clusterInfo := &fakeClusterInfo{
		mgr: &fakeResourceManager{},
	}
	rs := newRetryState(policy, nil, clusterInfo, protocol.HTTP1)
	testcases := []struct {
		Header   types.HeaderMap
		Reason   types.StreamResetReason
		Expected api.RetryCheckStatus
	}{
		{nil, types.StreamConnectionFailed, api.ShouldRetry},
	}
	for i, tc := range testcases {
		if rs.retry(context.Background(), tc.Header, tc.Reason) != tc.Expected {
			t.Errorf("#%d retry state failed", i)
		}
	}
}

func TestRetryStateStatusCode(t *testing.T) {
	rcfg := &v2.Router{}
	pcfg := &v2.RetryPolicy{
		RetryPolicyConfig: v2.RetryPolicyConfig{
			RetryOn:     true,
			NumRetries:  10,
			StatusCodes: []uint32{404, 500},
		},
		RetryTimeout: time.Second,
	}
	rcfg.Route = v2.RouteAction{}
	rcfg.Route.RetryPolicy = pcfg
	r, _ := router.NewRouteRuleImplBase(nil, rcfg)
	policy := r.Policy().RetryPolicy()
	clusterInfo := &fakeClusterInfo{
		mgr: &fakeResourceManager{},
	}
	rs := newRetryState(policy, nil, clusterInfo, protocol.HTTP1)

	variable.Register(variable.NewStringVariable(types.VarHeaderStatus, nil, nil, variable.DefaultStringSetter, 0))
	ctx1 := variable.NewVariableContext(context.Background())
	variable.SetString(ctx1, types.VarHeaderStatus, "200")
	ctx2 := variable.NewVariableContext(context.Background())
	variable.SetString(ctx2, types.VarHeaderStatus, "500")
	ctx3 := variable.NewVariableContext(context.Background())
	variable.SetString(ctx3, types.VarHeaderStatus, "404")
	ctx4 := variable.NewVariableContext(context.Background())
	variable.SetString(ctx4, types.VarHeaderStatus, "504")

	testcases := []struct {
		ctx      context.Context
		Reason   types.StreamResetReason
		Expected api.RetryCheckStatus
	}{
		{nil, types.StreamConnectionFailed, api.ShouldRetry},
		{ctx1, "", api.NoRetry},
		{ctx2, "", api.ShouldRetry},
		{ctx3, "", api.ShouldRetry},
		{ctx4, "", api.NoRetry},
	}

	for i, tc := range testcases {
		if rs.retry(tc.ctx, nil, tc.Reason) != tc.Expected {
			t.Errorf("#%d retry state failed", i)
		}
	}
}
