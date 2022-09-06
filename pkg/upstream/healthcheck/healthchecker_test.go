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

package healthcheck

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

type testCounter struct {
	changed   uint32
	unchanged uint32
}
type testResult struct {
	results map[string]*testCounter
}

func (r *testResult) testCallback(host types.Host, changed bool, isHealthy bool) {
	addr := host.AddressString()
	c, ok := r.results[addr]
	if !ok {
		c = &testCounter{}
		r.results[addr] = c
	}
	if changed {
		atomic.AddUint32(&c.changed, 1)
	} else {
		atomic.AddUint32(&c.unchanged, 1)
	}
}

type testCase struct {
	ServiceName string               // ServiceName is used to diff stats
	host        *mockHost            // mock host to check health
	running     func(host *mockHost) // running function, usually just wait some times
	verify      func(hc *healthChecker) error
}

func TestHealthCheck(t *testing.T) {
	log.InitDefaultLogger("", log.DEBUG)
	interval := 500 * time.Millisecond
	firstInterval = interval
	RegisterSessionFactory(types.ProtocolName("test"), &mockSessionFactory{})
	result := &testResult{
		results: map[string]*testCounter{},
	}
	// add common callbacks
	RegisterCommonCallbacks("test", result.testCallback)
	testCases := []testCase{
		testCase{
			ServiceName: "test_success",
			host: &mockHost{
				addr:   "test_success",
				status: true,
			},
			running: func(*mockHost) {
				time.Sleep(5 * interval)
			},
			verify: func(hc *healthChecker) error {
				// verify, sleep 5 intervals, at least 4 health check
				cbCounter := result.results["test_success"]
				unchanged := atomic.LoadUint32(&cbCounter.unchanged)
				if unchanged < 4 {
					return fmt.Errorf("do not call enough callbacks %d", unchanged)
				}
				if !(hc.stats.attempt.Count() >= 4 &&
					hc.stats.success.Count() >= 4 &&
					hc.stats.failure.Count() == 0 &&
					hc.stats.healthy.Value() == 1) {
					return fmt.Errorf("stats not expected, %d, %d, %d, %d", hc.stats.attempt.Count(), hc.stats.success.Count(),
						hc.stats.failure.Count(), hc.stats.healthy.Value())
				}
				return nil
			},
		},
		testCase{
			ServiceName: "test_fail",
			host: &mockHost{
				addr:   "test_fail",
				status: false,
			},
			running: func(*mockHost) {
				time.Sleep(5 * interval)
			},

			verify: func(hc *healthChecker) error {
				// verify, sleep 5 intervals, at least 4 health check
				// check fail, expected 1 changed
				cbCounter := result.results["test_fail"]
				changed := atomic.LoadUint32(&cbCounter.changed)
				unchanged := atomic.LoadUint32(&cbCounter.unchanged)
				if changed != 1 || changed+unchanged < 4 {
					return fmt.Errorf("do not call enough callbacks %d, %d", changed, unchanged+changed)
				}
				if !(hc.stats.attempt.Count() >= 4 &&
					hc.stats.success.Count() == 0 &&
					hc.stats.failure.Count() >= 4 &&
					hc.stats.activeFailure.Count() >= 4 &&
					hc.stats.healthy.Value() == 0) {
					return fmt.Errorf("stats not expected, %d, %d, %d, %d, %d", hc.stats.attempt.Count(), hc.stats.success.Count(),
						hc.stats.failure.Count(), hc.stats.activeFailure.Count(), hc.stats.healthy.Value())
				}
				return nil
			},
		},
		testCase{
			ServiceName: "test_timeout",
			host: &mockHost{
				addr:   "test_timeout",
				delay:  2 * time.Second,
				status: true,
			},
			running: func(*mockHost) {
				time.Sleep(3 * time.Second)
			},
			verify: func(hc *healthChecker) error {
				// verify, timeout is 2 seconds, last 3 seconds, expected get a timeout failure
				// should not retry more than 2
				cbCounter := result.results["test_timeout"]
				changed := atomic.LoadUint32(&cbCounter.changed)
				if changed != 1 {
					return fmt.Errorf("timeout changed not expected %d", changed)
				}
				if !(hc.stats.attempt.Count() >= 1 &&
					hc.stats.attempt.Count() <= 2 &&
					hc.stats.success.Count() == 0 &&
					hc.stats.failure.Count() <= 2 &&
					hc.stats.networkFailure.Count() <= 2 &&
					hc.stats.healthy.Value() == 0) {
					return fmt.Errorf("stats not expected, %d, %d, %d, %d, %d", hc.stats.attempt.Count(), hc.stats.success.Count(),
						hc.stats.failure.Count(), hc.stats.networkFailure.Count(), hc.stats.healthy.Value())
				}
				return nil
			},
		},
		testCase{
			ServiceName: "test_fail2good",
			host: &mockHost{
				addr:   "test_fail2good",
				status: false,
			},
			running: func(host *mockHost) {
				time.Sleep(interval + interval/2)
				host.SetHealth(true)
				time.Sleep(5 * interval)
			},
			verify: func(hc *healthChecker) error {
				// good - bad - good, 2 changed
				// at least 4 attempts
				cbCounter := result.results["test_fail2good"]
				changed := atomic.LoadUint32(&cbCounter.changed)
				unchanged := atomic.LoadUint32(&cbCounter.unchanged)
				if changed != 2 || changed+unchanged < 4 {
					return fmt.Errorf("do not call enough callbacks %d, %d", changed, unchanged+changed)
				}
				if !(hc.stats.attempt.Count() >= 4 &&
					hc.stats.success.Count() >= 4 &&
					hc.stats.failure.Count() == 1 &&
					hc.stats.activeFailure.Count() == 1 &&
					hc.stats.healthy.Value() == 1) {
					return fmt.Errorf("stats not expected, %d, %d, %d, %d, %d", hc.stats.attempt.Count(), hc.stats.success.Count(),
						hc.stats.failure.Count(), hc.stats.activeFailure.Count(), hc.stats.healthy.Value())
				}
				return nil
			},
		},
	}
	for i, tc := range testCases {
		cfg := v2.HealthCheck{
			HealthCheckConfig: v2.HealthCheckConfig{
				Protocol:           "test",
				HealthyThreshold:   1,
				UnhealthyThreshold: 1,
				ServiceName:        tc.ServiceName,
				CommonCallbacks:    []string{"test"},
			},
			Interval: interval,
		}
		cluster := &mockCluster{
			hs: &mockHostSet{
				hosts: []types.Host{
					tc.host,
				},
			},
		}
		hc := CreateHealthCheck(cfg)
		hc.SetHealthCheckerHostSet(cluster.hs)
		tc.running(tc.host)
		time.Sleep(100 * time.Millisecond) // make sure checks finish
		hc.Stop()
		raw := hc.(*healthChecker)
		if err := tc.verify(raw); err != nil {
			t.Errorf("#%d, %v", i, err)
		}
	}
}

func equal(originHosts, targetHosts []types.Host) bool {
	tmp := make(map[string]types.Host, len(originHosts))
	for _, origin := range originHosts {
		tmp[origin.AddressString()] = origin
	}

	for _, targetHost := range targetHosts {
		delete(tmp, targetHost.AddressString())
	}

	if len(tmp) == 0 {
		return true
	}
	return false
}

func Test_findNewAndDeleteHost(t *testing.T) {
	type args struct {
		old types.HostSet
		new types.HostSet
	}
	tests := []struct {
		name            string
		args            args
		wantDeleteHosts []types.Host
		wantNewHosts    []types.Host
	}{
		{
			name: "find_delete_and_new",
			args: args{
				old: newMockHostSet([]types.Host{
					&mockHost{
						addr: "addr1",
					},
					&mockHost{
						addr: "addr2",
					},
					&mockHost{
						addr: "addr3",
					},
				}),
				new: newMockHostSet([]types.Host{
					&mockHost{
						addr: "addr3",
					},
					&mockHost{
						addr: "addr4",
					},
					&mockHost{
						addr: "addr5",
					},
				}),
			},
			wantDeleteHosts: []types.Host{
				&mockHost{
					addr: "addr1",
				},
				&mockHost{
					addr: "addr2",
				},
			},
			wantNewHosts: []types.Host{
				&mockHost{
					addr: "addr4",
				},
				&mockHost{
					addr: "addr5",
				},
			},
		},
		{
			name: "find_delete",
			args: args{
				old: newMockHostSet(newMockHosts(0, 100)),
				new: newMockHostSet([]types.Host{}),
			},
			wantDeleteHosts: newMockHosts(0, 100),
			wantNewHosts:    []types.Host{},
		},
		{
			name: "find_new",
			args: args{
				old: newMockHostSet([]types.Host{}),
				new: newMockHostSet(newMockHosts(0, 100)),
			},
			wantDeleteHosts: []types.Host{},
			wantNewHosts:    newMockHosts(0, 100),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteHosts, newHosts := findNewAndDeleteHost(tt.args.old, tt.args.new)
			if !equal(deleteHosts, tt.wantDeleteHosts) {
				t.Errorf("findNewAndDeleteHost() deleteHosts = %v, wantDeleteHosts %v", deleteHosts, tt.wantDeleteHosts)
			}
			if !equal(newHosts, tt.wantNewHosts) {
				t.Errorf("findNewAndDeleteHost() newHosts = %v, want1NewHosts %v", newHosts, tt.wantNewHosts)
			}
		})
	}
}

func newMockHosts(from, to int) []types.Host {
	mockHosts := make([]types.Host, 0, to-from)
	for i := from; i < to; i++ {
		mockHosts = append(mockHosts, &mockHost{
			addr: "address-" + strconv.Itoa(i),
		})
	}
	return mockHosts
}

func Benchmark_findNewAndDeleteHost1(b *testing.B) {
	type args struct {
		oldHostset types.HostSet
		newHostset types.HostSet
	}
	benchmarkCases := []struct {
		name string
		args args
	}{
		{
			name: "find-1newHost-1deleteHost-from10hosts",
			args: args{
				oldHostset: newMockHostSet(newMockHosts(0, 10)),
				newHostset: newMockHostSet(newMockHosts(1, 11)),
			},
		},
		{
			name: "find-10newHost-10deleteHost-from100hosts",
			args: args{
				oldHostset: newMockHostSet(newMockHosts(0, 100)),
				newHostset: newMockHostSet(newMockHosts(10, 110)),
			},
		},
		{
			name: "find-10newHost-10deleteHost-from1000hosts",
			args: args{
				oldHostset: newMockHostSet(newMockHosts(0, 1000)),
				newHostset: newMockHostSet(newMockHosts(10, 1010)),
			},
		},
		{
			name: "find-100newHost-100deleteHost-from1000hosts",
			args: args{
				oldHostset: newMockHostSet(newMockHosts(0, 1000)),
				newHostset: newMockHostSet(newMockHosts(100, 1100)),
			},
		},
		{
			name: "find-500newHost-500deleteHost-from1000hosts",
			args: args{
				oldHostset: newMockHostSet(newMockHosts(0, 1000)),
				newHostset: newMockHostSet(newMockHosts(500, 1500)),
			},
		},
	}
	for _, bc := range benchmarkCases {
		b.Run(bc.name, func(b *testing.B) {
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				findNewAndDeleteHost(bc.args.oldHostset, bc.args.newHostset)
			}
			b.StopTimer()
		})
	}
}

func Test_InitialDelaySeconds(t *testing.T) {
	cfg := v2.HealthCheck{
		HealthCheckConfig: v2.HealthCheckConfig{
			Protocol:           "testInitialDelay",
			HealthyThreshold:   1,
			UnhealthyThreshold: 1,
			ServiceName:        "testServiceName",
			CommonCallbacks:    []string{"test"},
		},
	}
	hc := CreateHealthCheck(cfg)
	hcx := hc.(*healthChecker)
	if hcx.initialDelay != firstInterval {
		t.Errorf("Test_InitialDelaySeconds Error %+v", hcx)
	}

	cfg = v2.HealthCheck{
		HealthCheckConfig: v2.HealthCheckConfig{
			Protocol:            "testInitialDelay",
			HealthyThreshold:    1,
			UnhealthyThreshold:  1,
			InitialDelaySeconds: api.DurationConfig{time.Second * 2},
			ServiceName:         "testServiceName",
			CommonCallbacks:     []string{"test"},
		},
	}
	hc = CreateHealthCheck(cfg)
	hcx = hc.(*healthChecker)
	if hcx.initialDelay != time.Second*2 {
		t.Errorf("Test_InitialDelaySeconds Error %+v", hcx)
	}
}

func Test_HttpHealthCheck(t *testing.T) {
	hcString := `{"protocol":"Http1","timeout":"20s","interval":"0s","interval_jitter":"0s","initial_delay_seconds":"0s","service_name":"testCluster","check_config":{"http_check_config":{"port":33333,"timeout":"2s","path":"/test"}}}`
	cfg := &v2.HealthCheck{}
	json.Unmarshal([]byte(hcString), cfg)
	hc := newHealthChecker(*cfg, &HTTPDialSessionFactory{})
	h := &mockHost{
		addr: "127.0.0.1:33333",
	}
	hs := &mockHostSet{
		hosts: []types.Host{
			h,
		},
	}
	hc.SetHealthCheckerHostSet(hs)
	hcc := hc.(*healthChecker)
	if hcc.sessionConfig[HTTPCheckConfigKey] == nil {
		t.Errorf("Test_HttpHealthCheck error")
	}
	hcs := hcc.sessionFactory.NewSession(hcc.sessionConfig, h)
	_, ok := hcs.(*HTTPDialSession)
	if !ok {
		t.Errorf("Test_HttpHealthCheck error")
	}
}
