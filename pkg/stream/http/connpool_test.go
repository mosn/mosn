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

package http

import (
	"context"
	"sync"
	"testing"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
)

type fakeClusterInfo struct {
	types.ClusterInfo
	mgr types.ResourceManager
}

func (ci *fakeClusterInfo) ResourceManager() types.ResourceManager {
	return ci.mgr
}

func (ci *fakeClusterInfo) Name() string {
	return "test"
}

func (ci *fakeClusterInfo) Mark() uint32 {
	return 0
}

type fakeTLSContextManager struct {
	types.TLSContextManager
}

func (mg *fakeTLSContextManager) Enabled() bool {
	return false
}

func (mg *fakeTLSContextManager) HashValue() *types.HashValue {
	return nil
}

func (mg *fakeTLSContextManager) Fallback() bool {
	return false
}

func (ci *fakeClusterInfo) TLSMng() types.TLSClientContextManager {
	return &fakeTLSContextManager{}
}

func (ci *fakeClusterInfo) ConnectTimeout() time.Duration {
	return network.DefaultConnectTimeout
}

func (ci *fakeClusterInfo) IdleTimeout() time.Duration {
	return 0
}

func (ci *fakeClusterInfo) ConnBufferLimitBytes() uint32 {
	return 0
}

func (ci *fakeClusterInfo) Stats() *types.ClusterStats {
	return &types.ClusterStats{
		UpstreamRequestPendingOverflow:                 metrics.NewCounter(),
		UpstreamConnectionRemoteCloseWithActiveRequest: metrics.NewCounter(),
		UpstreamConnectionTotal:                        metrics.NewCounter(),
		UpstreamConnectionActive:                       metrics.NewCounter(),
		UpstreamConnectionConFail:                      metrics.NewCounter(),
	}
}

func (ci *fakeClusterInfo) SlowStart() types.SlowStart {
	return types.SlowStart{}
}

type fakeResourceManager struct {
	types.ResourceManager
	max uint64
}

func (mgr *fakeResourceManager) Connections() types.Resource {
	return &fakeResource{max: mgr.max}
}

type fakeResource struct {
	max uint64
}

func (r *fakeResource) CanCreate() bool {
	return true
}
func (r *fakeResource) Increase()       {}
func (r *fakeResource) Decrease()       {}
func (r *fakeResource) Cur() int64      { return 0 }
func (r *fakeResource) UpdateCur(int64) {}
func (r *fakeResource) Max() uint64     { return r.max }

func TestGetAvailableClient(t *testing.T) {

	var max uint64 = 2
	ci := &fakeClusterInfo{
		mgr: &fakeResourceManager{max: max},
	}

	hc := v2.Host{
		HostConfig: v2.HostConfig{
			Address:  "127.0.0.1:10002",
			Hostname: "127.0.0.1:10002",
		},
	}
	host := cluster.NewSimpleHost(hc, ci)
	pool := NewConnPool(context.TODO(), host).(*connPool)

	wg := sync.WaitGroup{}
	wg.Add(500)
	for i := 0; i < 500; i++ {
		go func() {
			pool.getAvailableClient(context.Background())
			wg.Done()
		}()
	}
	wg.Wait()
	if pool.totalClientCount > uint64(max) {
		t.Fatal("limit max connections failed")
	}
}
