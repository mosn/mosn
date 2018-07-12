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

	"github.com/alipay/sofamosn/pkg/api/v2"
	"github.com/alipay/sofamosn/pkg/log"
	"github.com/alipay/sofamosn/pkg/types"
	"github.com/alipay/sofamosn/pkg/upstream/cluster"
)

const (
	testHealthCheckInterval = 2
	testHealthCheckTimeout  = 5
	testHealthyThreshold    = 5
	testUnhealthyThreshold  = 5
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
			case <-time.After(time.Duration(s.healthChecker.interval) * time.Second):
				switch s.healthChecker.mode {
				case 1:
					s.handleSuccess()
				case 2:
					s.handleFailure(types.FailureActive)
				case 3:
				case 4:
				}

			}
		}()
	case 5:
		// do nothing
	}
}

func Test_success(t *testing.T) {
	c := mockEnv(1)

	checkLoop := 0

	for {
		time.Sleep(time.Duration(testHealthCheckInterval*testUnhealthyThreshold+5) * time.Second)

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
		time.Sleep(time.Duration(testHealthCheckInterval*testUnhealthyThreshold+5) * time.Second)

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
	// todo
}

func Test_success_2_failed(t *testing.T) {
	// todo
}

func Test_timeout(t *testing.T) {
	c := mockEnv(5)

	time.Sleep(time.Duration(testHealthCheckTimeout*testUnhealthyThreshold+5) * time.Second)

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
	log.InitDefaultLogger("", log.INFO)

	config := v2.HealthCheck{
		Protocol:           "mock",
		Timeout:            testHealthCheckTimeout,
		Interval:           testHealthCheckInterval,
		IntervalJitter:     1,
		HealthyThreshold:   testHealthyThreshold,
		UnhealthyThreshold: testUnhealthyThreshold,
	}

	hc := newMockHealthCheck(config, mode)

	c := cluster.NewCluster(v2.Cluster{
		Name:        "test",
		ClusterType: v2.SIMPLE_CLUSTER,
		LbType:      v2.LB_RANDOM,
	}, nil, true)

	host := cluster.NewHost(v2.Host{
		Address:  "127.0.0.1",
		Hostname: "hostname",
		Weight:   100,
	}, nil)
	hosts := []types.Host{host}
	hostsPerLocality := [][]types.Host{hosts}

	c.PrioritySet().GetOrCreateHostSet(1).UpdateHosts(hosts, hosts,
		hostsPerLocality, hostsPerLocality, nil, nil)

	c.SetHealthChecker(hc)

	return c
}
