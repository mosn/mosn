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
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/upstream/cluster"
)

const (
	testHealthCheckInterval = 200 // ms
	testHealthCheckTimeout  = 500 // ms
	testHealthyThreshold    = 5
	testUnhealthyThreshold  = 5
)

var (
	testIntervalDeadline  = time.Duration(2*testHealthCheckInterval*testUnhealthyThreshold + 50)
	testTimeoutDeadline   = time.Duration(2*testHealthCheckTimeout*testUnhealthyThreshold + 50)
	failureToSuccessChain bool
	successToFailureChain bool
)

type mockHealthChecker struct {
	healthChecker

	mode int // 1:success, 2:failed, 3: failed -> success, 4: success -> failed, 5: timeout
}

func newMockHealthCheck(config v2.HealthCheck, mode int) types.HealthChecker {
	hc := newHealthChecker(config)

	mhc := &mockHealthChecker{
		healthChecker: *hc,
		mode:          mode,
	}

	mhc.sessionFactory = mhc

	return mhc
}

func (c *mockHealthChecker) newSession(host types.Host) types.HealthCheckSession {
	shcs := &mockHealthCheckSession{
		healthChecker:      c,
		healthCheckSession: *newHealthCheckSession(&c.healthChecker, host),
	}

	// add timer to trigger hb sending and timeout handling
	shcs.intervalTimer = newTimer(shcs.onInterval)
	shcs.timeoutTimer = newTimer(shcs.onTimeout)

	return shcs
}

type mockHealthCheckSession struct {
	healthCheckSession

	healthChecker *mockHealthChecker

	healthyThreshold   uint32
	unhealthyThreshold uint32

	failureCounter int
	successCounter int
}

func (s *mockHealthCheckSession) Start() {
	// start interval timer
	s.onInterval()
}

func (s *mockHealthCheckSession) onInterval() {
	s.mockHealthCheckAction()

	s.healthCheckSession.onInterval()
}

func (s *mockHealthCheckSession) mockHealthCheckAction() {
	switch s.healthChecker.mode {
	case 1, 2, 3, 4:
		go func() {
			select {
			case <-time.After(s.healthChecker.interval):
				switch s.healthChecker.mode {
				case 1:
					//fmt.Println("handle success")
					s.handleSuccess()
				case 2:
					// fmt.Println("handle failure")
					s.handleFailure(types.FailureActive)
				case 3:
					if failureToSuccessChain {
						//fmt.Println("handle success")
						s.handleSuccess()
					} else {
						// fmt.Println("handle failure")
						s.handleFailure(types.FailureActive)
					}
				case 4:
					if successToFailureChain {
						//fmt.Println("handle failure")
						s.handleFailure(types.FailureActive)
					} else {
						//fmt.Println("handle success")
						s.handleSuccess()
					}
				}
			}
		}()

	case 5:
		// do nothing
		// wait timeout
	}
}

func Test_success(t *testing.T) {
	c := mockEnv(1)

	checkLoop := 0

	for {
		// (interval + interval) * threshold
		time.Sleep(testIntervalDeadline * time.Millisecond)

		hostCounter := len(c.PrioritySet().GetOrCreateHostSet(1).Hosts())

		if hostCounter != 1 {
			t.Fatal("err host counter")
		}

		hhostCounter := len(c.PrioritySet().GetOrCreateHostSet(1).HealthyHosts())

		if hhostCounter != 1 {
			t.Fatal("err healthy host content")
		}

		checkLoop++
		if checkLoop > 3 {
			break
		}
	}
}

func Test_failure(t *testing.T) {
	c := mockEnv(2)

	checkLoop := 0

	for {
		// (interval + interval) * threshold
		time.Sleep(testIntervalDeadline * time.Millisecond)

		hostCounter := len(c.PrioritySet().GetOrCreateHostSet(1).Hosts())

		if hostCounter != 1 {
			t.Fatal("err host counter")
		}

		hhostCounter := len(c.PrioritySet().GetOrCreateHostSet(1).HealthyHosts())

		if hhostCounter != 0 {
			t.Fatal("err healthy host content")
		}

		checkLoop++
		if checkLoop > 3 {
			break
		}
	}
}

func Test_failure_2_success(t *testing.T) {
	c := mockEnv(3)
	checkLoop := 0

	for {
		checkLoop++
		if checkLoop > 3 {
			failureToSuccessChain = true
		}

		// (interval + interval) * threshold
		time.Sleep(testIntervalDeadline * time.Millisecond)

		hostCounter := len(c.PrioritySet().GetOrCreateHostSet(1).Hosts())
		hhostCounter := len(c.PrioritySet().GetOrCreateHostSet(1).HealthyHosts())

		if hostCounter != 1 {
			t.Fatal("err host counter")
		}

		if checkLoop > 6 {
			break
		} else if checkLoop > 3 {
			if hhostCounter != 1 {
				t.Fatal("err healthy host content")
			}
		} else {
			if hhostCounter != 0 {
				t.Fatal("err healthy host content")
			}
		}
	}
}

func Test_success_2_failed(t *testing.T) {
	c := mockEnv(4)
	checkLoop := 0

	for {
		checkLoop++
		if checkLoop > 3 {
			successToFailureChain = true
		}

		// (interval + interval) * threshold
		time.Sleep(testIntervalDeadline * time.Millisecond)

		hostCounter := len(c.PrioritySet().GetOrCreateHostSet(1).Hosts())
		hhostCounter := len(c.PrioritySet().GetOrCreateHostSet(1).HealthyHosts())

		if hostCounter != 1 {
			t.Fatal("err host counter")
		}

		if checkLoop > 6 {
			break
		} else if checkLoop > 3 {
			if hhostCounter != 0 {
				t.Fatal("err healthy host content")
			}
		} else {
			if hhostCounter != 1 {
				t.Fatal("err healthy host content")
			}
		}
	}
}

func Test_timeout(t *testing.T) {
	c := mockEnv(5)

	// (timeout + timeout) * threshold
	time.Sleep(testTimeoutDeadline * time.Millisecond)

	hostCounter := len(c.PrioritySet().GetOrCreateHostSet(1).Hosts())

	if hostCounter != 1 {
		t.Fatal("err host counter")
	}

	hhostCounter := len(c.PrioritySet().GetOrCreateHostSet(1).HealthyHosts())

	if hhostCounter != 0 {
		t.Fatal("err healthy host content")
	}
}

func Test_cluster_modified(t *testing.T) {
	// todo
}

func mockEnv(mode int) types.Cluster {
	log.InitDefaultLogger("", log.DEBUG)

	config := v2.HealthCheck{
		HealthCheckConfig: v2.HealthCheckConfig{
			Protocol:           "mock",
			HealthyThreshold:   testHealthyThreshold,
			UnhealthyThreshold: testUnhealthyThreshold,
		},
		Timeout:        testHealthCheckTimeout * time.Millisecond,
		Interval:       testHealthCheckInterval * time.Millisecond,
		IntervalJitter: 1,
	}

	hc := newMockHealthCheck(config, mode)

	c := cluster.NewCluster(v2.Cluster{
		Name:        "test",
		ClusterType: v2.SIMPLE_CLUSTER,
		LbType:      v2.LB_RANDOM,
	}, nil, true)

	host := cluster.NewHost(v2.Host{
		HostConfig: v2.HostConfig{
			Address:  "127.0.0.1",
			Hostname: "hostname",
			Weight:   100,
		},
	}, nil)
	hosts := []types.Host{host}
	hostsPerLocality := [][]types.Host{hosts}

	c.PrioritySet().GetOrCreateHostSet(1).UpdateHosts(hosts, hosts,
		hostsPerLocality, hostsPerLocality, nil, nil)

	c.SetHealthChecker(hc)

	return c
}
