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

package metrix

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/model"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/rcrowley/go-metrics"
)

// namespace
const (
	NAMESPACE = "CommonRule"
	INVOKE    = "INVOKE"
	BLOCK     = "BLOCK"
)

// Stat as
type Stat struct {
	ruleConfig *model.RuleConfig
	counters   map[string]metrics.Counter
	ticker     *ticker
}

// NewStat new
func NewStat(ruleConfig *model.RuleConfig) *Stat {
	s := &Stat{
		ruleConfig: ruleConfig,
		counters:   make(map[string]metrics.Counter),
	}
	s.AddCounter(INVOKE)
	s.AddCounter(BLOCK)
	s.Start()
	return s
}

// Start ticker
func (s *Stat) Start() {
	s.ticker = NewTicker(s.callback)
	s.ticker.Start(time.Millisecond * time.Duration(s.ruleConfig.LimitConfig.PeriodMs))
}

// Stop ticker
func (s *Stat) Stop() {
	s.ticker.Stop()
}

func (s *Stat) callback() {
	if s.Counter(INVOKE).Count() > 0 {
		log.DefaultLogger.Infof(s.String())
		s.clear()
	}
}

//end ticker

// AddCounter add counter = name in Stats.counters
func (s *Stat) AddCounter(name string) *Stat {
	metricsKey := fmt.Sprintf("%s.%s", NAMESPACE, name)
	s.counters[name] = metrics.GetOrRegisterCounter(metricsKey, nil)

	return s
}

// SetCounter set gauges = name, Stats.counters = counter
func (s *Stat) SetCounter(name string, counter metrics.Counter) {
	s.counters[name] = counter
}

// Counter return s.counters[name]
func (s *Stat) Counter(name string) metrics.Counter {
	return s.counters[name]
}

func (s *Stat) clear() {
	s.counters[INVOKE].Clear()
	s.counters[BLOCK].Clear()
}

func (s *Stat) String() string {
	var buffer bytes.Buffer

	//buffer.WriteString(fmt.Sprintf("namespace: %s, ", s.namespace))
	buffer.WriteString("namespace: " + NAMESPACE + ", ")

	if len(s.counters) > 0 {
		buffer.WriteString(strconv.FormatInt(int64(s.ruleConfig.Id), 10))
		buffer.WriteString(",")
		buffer.WriteString(s.ruleConfig.Name)
		buffer.WriteString(",")
		buffer.WriteString(s.ruleConfig.RunMode)
		buffer.WriteString(",")
		buffer.WriteString(strconv.FormatInt(int64(s.ruleConfig.LimitConfig.PeriodMs), 10))
		buffer.WriteString(",")
		buffer.WriteString(strconv.FormatInt(int64(s.ruleConfig.LimitConfig.MaxAllows), 10))
		buffer.WriteString(",")

		buffer.WriteString(strconv.FormatInt(int64(s.counters[INVOKE].Count()), 10))
		buffer.WriteString(",")
		buffer.WriteString(strconv.FormatInt(int64(s.counters[INVOKE].Count()-s.counters[BLOCK].Count()), 10))
		buffer.WriteString(",")
		buffer.WriteString(strconv.FormatInt(int64(s.counters[BLOCK].Count()), 10))
		buffer.WriteString(",")
	}

	return buffer.String()
}
