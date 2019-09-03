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
	"github.com/rcrowley/go-metrics"
	"sofastack.io/sofa-mosn/pkg/featuregate"
	"sofastack.io/sofa-mosn/pkg/types"
	"sync"
	"time"
)

const (
	channelSize = 2048
)

var (
	concurrencyMap = map[string]*Concurrency{}
	mutex          sync.RWMutex
)

type Concurrency struct {
	metricKey                      string
	lastCalculateTime              time.Time
	lastConcurrency                int64
	timeOnConcurrency              map[int64]int64
	calculateChan                  chan *ConcurrentData
	updateTicker                   <-chan time.Time
	totalConcurrentTimeAreaCounter metrics.Counter
	mutex                          sync.Mutex
}

func getOrNewConcurrency(metricKey string, s types.Metrics) *Concurrency {
	if !featuregate.DefaultFeatureGate.Enabled(featuregate.ConcurrencyMetricsEnable) {
		return nil
	}

	mutex.Lock()
	defer mutex.Unlock()

	c, found := concurrencyMap[metricKey]
	if found {
		return c
	}

	c = &Concurrency{
		metricKey:         metricKey,
		timeOnConcurrency: make(map[int64]int64, 1),
		calculateChan:     make(chan *ConcurrentData, channelSize),
		// TODO: configurable
		updateTicker:                   time.NewTicker(time.Second).C,
		totalConcurrentTimeAreaCounter: s.Counter(metricKey),
		mutex:                          sync.Mutex{},
	}

	concurrencyMap[metricKey] = c
	go c.mainLoop()

	return c
}

func (c *Concurrency) calculate(t time.Time, reqType ReqEventType) {
	c.calculateChan <- &ConcurrentData{
		time:    t,
		reqType: reqType,
	}
}

func (c *Concurrency) mainLoop() {
	c.lastCalculateTime = time.Now()
	c.lastConcurrency = 0
	for {
		select {
		case data := <-c.calculateChan:
			c.calculateTimeOnConcurrency(data.time)
			if data.reqType == ReqIn {
				c.lastConcurrency += 1
			} else if data.reqType == ReqOut {
				c.lastConcurrency -= 1
			}
		case now := <-c.updateTicker:
			c.calculateTimeOnConcurrency(now)
			c.updateTotalConcurrentTimeArea()
		}
	}
}

func (c *Concurrency) calculateTimeOnConcurrency(time time.Time) {
	if time.After(c.lastCalculateTime) {
		durationSinceChange := time.Sub(c.lastCalculateTime).Nanoseconds() / 1.0e6
		c.timeOnConcurrency[c.lastConcurrency] += durationSinceChange
		c.lastCalculateTime = time
	}
}

func (c *Concurrency) updateTotalConcurrentTimeArea() {
	concurrentTimeArea := int64(0)
	for c, t := range c.timeOnConcurrency {
		concurrentTimeArea += t * c
	}

	c.totalConcurrentTimeAreaCounter.Inc(concurrentTimeArea)
	c.timeOnConcurrency = make(map[int64]int64, 1)
}

func (c *Concurrency) Value() int64 {
	return c.totalConcurrentTimeAreaCounter.Count()
}

type ReqEventType int

const (
	// ReqIn represents an incoming request
	ReqIn ReqEventType = iota
	// ReqOut represents a finished request
	ReqOut
)

type ConcurrentData struct {
	time    time.Time
	reqType ReqEventType
}

func calculateConcurrency(c *Concurrency, typ ReqEventType) {
	if !featuregate.DefaultFeatureGate.Enabled(featuregate.ConcurrencyMetricsEnable) {
		return
	}

	if c == nil {
		return
	}

	c.calculate(time.Now(), typ)
}
