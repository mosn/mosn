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
	"math/rand"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/rcrowley/go-metrics"
)

type sessionFactory interface {
	newSession(host types.Host) types.HealthCheckSession
}

// types.HealthChecker
type healthChecker struct {
	serviceName         string
	healthCheckCbs      []types.HealthCheckCb
	cluster             types.Cluster
	healthCheckSessions map[types.Host]types.HealthCheckSession

	timeout        time.Duration
	interval       time.Duration
	intervalJitter time.Duration

	localProcessHealthy uint64

	healthyThreshold   uint32
	unhealthyThreshold uint32

	sessionFactory sessionFactory
	stats          *healthCheckStats
}

func newHealthChecker(config v2.HealthCheck) *healthChecker {
	hc := &healthChecker{
		healthCheckSessions: make(map[types.Host]types.HealthCheckSession),
		timeout:             config.Timeout,
		interval:            config.Interval,
		intervalJitter:      config.IntervalJitter,
		healthyThreshold:    config.HealthyThreshold,
		unhealthyThreshold:  config.UnhealthyThreshold,
		stats:               newHealthCheckStats(config.ServiceName),
	}

	if config.ServiceName != "" {
		hc.serviceName = config.ServiceName
	}

	return hc
}

func (c *healthChecker) SetCluster(cluster types.Cluster) {
	c.cluster = cluster
}

func (c *healthChecker) Start() {
	for _, hostSet := range c.cluster.PrioritySet().HostSetsByPriority() {
		c.addHosts(hostSet.Hosts())
	}
}

func (c *healthChecker) Stop() {
	// todo
}

func (c *healthChecker) AddHostCheckCompleteCb(cb types.HealthCheckCb) {
	c.healthCheckCbs = append(c.healthCheckCbs, cb)
}

func (c *healthChecker) newSession(host types.Host) types.HealthCheckSession {
	if c.sessionFactory != nil {
		return c.sessionFactory.newSession(host)
	}

	return &healthCheckSession{
		healthChecker: c,
		host:          host,
	}
}

func (c *healthChecker) addHosts(hosts []types.Host) {
	for _, host := range hosts {
		h := host

		go func() {
			var ns types.HealthCheckSession

			if ns = c.newSession(h); ns == nil {
				log.DefaultLogger.Errorf("Create Health Check Session Error, Remote Address = %s", h.AddressString())
				return
			}

			c.healthCheckSessions[h] = ns
			c.healthCheckSessions[h].Start()
		}()
	}
}

func (c *healthChecker) delHosts(hosts []types.Host) {}

func (c *healthChecker) OnClusterMemberUpdate(hostsAdded []types.Host, hostDel []types.Host) {
	c.addHosts(hostsAdded)
	c.delHosts(hostDel)
}

func (c *healthChecker) decHealthy() {
	c.localProcessHealthy--
	c.refreshHealthyStat()
}

func (c *healthChecker) incHealthy() {
	c.localProcessHealthy++
	c.refreshHealthyStat()
}

func (c *healthChecker) refreshHealthyStat() {
	c.stats.healthy.Update(int64(c.localProcessHealthy))
}

func (c *healthChecker) getStats() *healthCheckStats {
	return c.stats
}

func (c *healthChecker) getInterval() time.Duration {
	baseInterval := c.interval

	if c.intervalJitter > 0 {
		jitter := int(rand.Float32() * float32(c.intervalJitter))
		baseInterval += time.Duration(jitter)
	}

	if baseInterval < 0 {
		baseInterval = 0
	}

	maxUint := ^uint(0)
	if uint(baseInterval) > maxUint {
		baseInterval = time.Duration(maxUint)
	}

	return baseInterval
}

func (c *healthChecker) getTimeoutDuration() time.Duration {
	baseInterval := c.timeout

	if baseInterval < 0 {
		baseInterval = 0
	}

	maxUint := ^uint(0)
	if uint(baseInterval) > maxUint {
		baseInterval = time.Duration(maxUint)
	}

	return baseInterval
}

// when health receive handling result
func (c *healthChecker) runCallbacks(host types.Host, changed bool) {
	c.refreshHealthyStat()

	for _, cb := range c.healthCheckCbs {
		cb(host, changed)
	}
}

type healthCheckSession struct {
	healthChecker *healthChecker

	intervalTimer *timer
	timeoutTimer  *timer

	numHealthy   uint32
	numUnHealthy uint32
	host         types.Host
}

func newHealthCheckSession(hc *healthChecker, host types.Host) *healthCheckSession {
	hcs := &healthCheckSession{
		healthChecker: hc,
		host:          host,
	}

	if !host.ContainHealthFlag(types.FAILED_ACTIVE_HC) {
		hcs.healthChecker.decHealthy()
	}

	return hcs
}

func (s *healthCheckSession) Start() {
	s.onInterval()
}

//// stop intervalTimer to stop sending health message
func (s *healthCheckSession) Stop() {
	s.intervalTimer.stop()
}

func (s *healthCheckSession) handleSuccess() {
	s.numUnHealthy = 0

	stateChanged := false
	if s.host.ContainHealthFlag(types.FAILED_ACTIVE_HC) {
		s.numHealthy++

		if s.numHealthy == s.healthChecker.healthyThreshold {
			s.host.ClearHealthFlag(types.FAILED_ACTIVE_HC)
			s.healthChecker.incHealthy()
			stateChanged = true
		}
	}

	s.healthChecker.stats.success.Inc(1)
	s.healthChecker.runCallbacks(s.host, stateChanged)

	// stop timeout timer
	s.timeoutTimer.stop()
	// start a new interval timer
	s.intervalTimer.start(s.healthChecker.getInterval())
}

func (s *healthCheckSession) SetUnhealthy(fType types.FailureType) {
	s.numHealthy = 0

	stateChanged := false
	if !s.host.ContainHealthFlag(types.FAILED_ACTIVE_HC) {
		s.numUnHealthy++

		if s.numUnHealthy == s.healthChecker.unhealthyThreshold {
			s.host.SetHealthFlag(types.FAILED_ACTIVE_HC)
			s.healthChecker.decHealthy()
			stateChanged = true
		}
	}

	s.healthChecker.stats.failure.Inc(1)

	switch fType {
	case types.FailureNetwork:
		s.healthChecker.stats.networkFailure.Inc(1)
	case types.FailurePassive:
		s.healthChecker.stats.passiveFailure.Inc(1)
	}

	s.healthChecker.runCallbacks(s.host, stateChanged)
}

func (s *healthCheckSession) handleFailure(fType types.FailureType) {
	s.SetUnhealthy(fType)

	// stop timeout timer
	s.timeoutTimer.stop()

	// start a new interval timer
	s.intervalTimer.start(s.healthChecker.getInterval())
}

func (s *healthCheckSession) onInterval() {
	s.timeoutTimer.start(s.healthChecker.getTimeoutDuration())
	s.healthChecker.stats.attempt.Inc(1)
}

func (s *healthCheckSession) onTimeout() {
	s.SetUnhealthy(types.FailureNetwork)
	// sending another health check
	s.intervalTimer.start(s.healthChecker.getInterval())
}

type healthCheckStats struct {
	attempt        metrics.Counter
	success        metrics.Counter
	failure        metrics.Counter
	passiveFailure metrics.Counter
	networkFailure metrics.Counter
	verifyCluster  metrics.Counter
	healthy        metrics.Gauge
}

func newHealthCheckStats(namespace string) *healthCheckStats {
	m := stats.NewHealthStats(namespace)
	return &healthCheckStats{
		attempt:        m.Counter(stats.HealthCheckAttempt),
		success:        m.Counter(stats.HealthCheckSuccess),
		failure:        m.Counter(stats.HealthCheckFailure),
		passiveFailure: m.Counter(stats.HealthCheckPassiveFailure),
		networkFailure: m.Counter(stats.HealthCheckNetworkFailure),
		verifyCluster:  m.Counter(stats.HealthCheckVeirfyCluster),
		healthy:        m.Gauge(stats.HealthCheckHealthy),
	}
}
