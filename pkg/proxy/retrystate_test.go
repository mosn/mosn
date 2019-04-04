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
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/types"
	metrics "github.com/rcrowley/go-metrics"
)

func doNothing() {}

type fakeClusterInfo struct {
	types.ClusterInfo
	mgr types.ResourceManager
}

func (ci *fakeClusterInfo) ResourceManager() types.ResourceManager {
	return ci.mgr
}
func (ci *fakeClusterInfo) Stats() types.ClusterStats {
	return types.ClusterStats{
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
func (r *fakeResource) Increase()   {}
func (r *fakeResource) Decrease()   {}
func (r *fakeResource) Max() uint64 { return 10 }

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
	headerException := protocol.CommonHeader{
		types.HeaderStatus: "500",
	}
	headerOK := protocol.CommonHeader{
		types.HeaderStatus: "200",
	}
	testcases := []struct {
		Header   types.HeaderMap
		Reason   types.StreamResetReason
		Expected types.RetryCheckStatus
	}{
		{nil, types.StreamConnectionFailed, types.ShouldRetry},
		{headerException, "", types.ShouldRetry},
		{headerOK, "", types.NoRetry},
	}
	for i, tc := range testcases {
		if rs.retry(tc.Header, tc.Reason) != tc.Expected {
			t.Errorf("#%d retry state failed", i)
		}
	}
}
