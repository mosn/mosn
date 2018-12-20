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
	"github.com/alipay/sofa-mosn/pkg/types"
)

// sessionChecker is a wrapper of types.HealthCheckSession for health check
type sessionChecker struct {
	Session       types.HealthCheckSession
	Host          types.Host
	HealthChecker *healthChecker
	//
	resp          chan checkResponse
	timeout       chan bool
	checkID       uint64
	stop          chan struct{}
	checkTimer    *timer
	checkTimeout  *timer
	unHealthCount uint32
	healthCount   uint32
}

type checkResponse struct {
	ID      uint64
	Healthy bool
}

func newChecker(s types.HealthCheckSession, h types.Host, hc *healthChecker) *sessionChecker {
	c := &sessionChecker{
		Session:       s,
		Host:          h,
		HealthChecker: hc,
		resp:          make(chan checkResponse),
		timeout:       make(chan bool),
		stop:          make(chan struct{}),
	}
	c.checkTimer = newTimer(c.OnCheck)
	c.checkTimeout = newTimer(c.OnTimeout)
	return c
}

func (c *sessionChecker) Start() {
	defer func() {
		// stop all the timer when start is finished
		c.checkTimer.stop()
		c.checkTimeout.stop()
	}()
	c.checkTimer.start(c.HealthChecker.getCheckInterval())
	for {
		select {
		case <-c.stop:
			return
		default:
			// prepare a check
			c.checkID++
			select {
			case <-c.stop:
				return
			case resp := <-c.resp:
				// if the ID is not equal, means we receive a timeout for this ID, ignore the response
				if resp.ID == c.checkID {
					c.checkTimeout.stop()
					if resp.Healthy {
						c.HandleSuccess()
					} else {
						c.HandleFailure(types.FailureActive)
					}
					// next health checker
					c.checkTimer.start(c.HealthChecker.getCheckInterval())
				}
			case <-c.timeout:
				c.checkTimer.stop()
				c.Session.OnTimeout() // session timeout callbacks
				c.HandleFailure(types.FailureNetwork)
				// next health checker
				c.checkTimer.start(c.HealthChecker.getCheckInterval())
			}
		}
	}
}

func (c *sessionChecker) Stop() {
	close(c.stop)
}

func (c *sessionChecker) HandleSuccess() {
	c.unHealthCount = 0
	changed := false
	if c.Host.ContainHealthFlag(types.FAILED_ACTIVE_HC) {
		c.healthCount++
		// check the threshold
		if c.healthCount == c.HealthChecker.healthyThreshold {
			changed = true
			c.Host.ClearHealthFlag(types.FAILED_ACTIVE_HC)
		}
	}
	c.HealthChecker.incHealthy(c.Host, changed)
}

func (c *sessionChecker) HandleFailure(reason types.FailureType) {
	c.healthCount = 0
	changed := false
	if !c.Host.ContainHealthFlag(types.FAILED_ACTIVE_HC) {
		c.unHealthCount++
		// check the threshold
		if c.unHealthCount == c.HealthChecker.unhealthyThreshold {
			changed = true
			c.Host.SetHealthFlag(types.FAILED_ACTIVE_HC)
		}
	}
	c.HealthChecker.decHealthy(c.Host, reason, changed)
}

func (c *sessionChecker) OnCheck() {
	// record current id
	id := c.checkID
	c.HealthChecker.stats.attempt.Inc(1)
	// start a timeout before check health
	c.checkTimeout.start(c.HealthChecker.timeout)
	c.resp <- checkResponse{
		ID:      id,
		Healthy: c.Session.CheckHealth(),
	}
}

func (c *sessionChecker) OnTimeout() {
	c.timeout <- true
}
