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
	"time"
)

func StartHTTPHealthCheck(tickInterval time.Duration, timerTimeout time.Duration, checkPath string,
	intervalCB func(path string, successCall func()), timeoutCB func()) {

	shc := HTTP1HealthCheck{
		checkPath:  checkPath,
		intervalCB: intervalCB,
		timeout:    timerTimeout,
		interval:   tickInterval,
	}

	shc.intervalTimer = newTimer(shc.OnInterval)
	shc.timeoutTimer = newTimer(timeoutCB)

	shc.Start()
}

type HTTP1HealthCheck struct {
	intervalTimer *timer
	timeoutTimer  *timer
	checkPath     string
	intervalCB    func(string string, h1f func())
	timeout       time.Duration
	interval      time.Duration
}

func (shc *HTTP1HealthCheck) Start() {
	shc.OnInterval()
}

func (shc *HTTP1HealthCheck) OnInterval() {
	// call app's function
	shc.intervalCB(shc.checkPath, shc.OnTimeoutTimerRest)
	shc.timeoutTimer.start(shc.timeout)
}

func (shc *HTTP1HealthCheck) OnTimeoutTimerRest() {
	shc.timeoutTimer.stop()
	shc.intervalTimer.start(shc.interval)
}
