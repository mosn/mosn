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
	"fmt"
	"sync/atomic"
	"testing"
	"time"

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
