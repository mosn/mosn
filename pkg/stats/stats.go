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

package stats

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/rcrowley/go-metrics"
)

// Stats for metrics recording
type Stats struct {
	namespace  string
	counters   map[string]metrics.Counter
	gauges     map[string]metrics.Gauge
	histograms map[string]metrics.Histogram
}

// NewStats for create new Stats
func NewStats(namespace string) *Stats {
	return &Stats{
		namespace:  namespace,
		counters:   make(map[string]metrics.Counter),
		gauges:     make(map[string]metrics.Gauge),
		histograms: make(map[string]metrics.Histogram),
	}
}

// AddCounter add counter = name in Stats.counters
func (s *Stats) AddCounter(name string) *Stats {
	metricsKey := fmt.Sprintf("%s.%s", s.namespace, name)
	s.counters[name] = metrics.GetOrRegisterCounter(metricsKey, nil)

	return s
}

// AddGauge add gauges = name in Stats.gauges
func (s *Stats) AddGauge(name string) *Stats {
	metricsKey := fmt.Sprintf("%s.%s", s.namespace, name)
	s.gauges[name] = metrics.GetOrRegisterGauge(metricsKey, nil)

	return s
}

// AddHistogram add histograms = name in Stats.histograms
func (s *Stats) AddHistogram(name string) *Stats {
	metricsKey := fmt.Sprintf("%s.%s", s.namespace, name)
	s.histograms[name] = metrics.GetOrRegisterHistogram(metricsKey, nil, metrics.NewUniformSample(100))

	return s
}

// SetCounter set gauges = name, Stats.counters = counter
func (s *Stats) SetCounter(name string, counter metrics.Counter) {
	s.counters[name] = counter
}

// SetGauge set gauges = name, Stats.gauges = gauge
func (s *Stats) SetGauge(name string, gauge metrics.Gauge) {
	s.gauges[name] = gauge
}

// SetHistogram set histograms = name, Stats.histograms = histograms
func (s *Stats) SetHistogram(name string, histogram metrics.Histogram) {
	s.histograms[name] = histogram
}

// Counter return s.counters[name]
func (s *Stats) Counter(name string) metrics.Counter {
	return s.counters[name]
}

// Gauge return s.gauges[name]
func (s *Stats) Gauge(name string) metrics.Gauge {
	return s.gauges[name]
}

// Histogram return s.histograms[name]
func (s *Stats) Histogram(name string) metrics.Histogram {
	return s.histograms[name]
}

func (s *Stats) String() string {
	var buffer bytes.Buffer

	//buffer.WriteString(fmt.Sprintf("namespace: %s, ", s.namespace))
	buffer.WriteString("namespace: " + s.namespace + ", ")

	if len(s.counters) > 0 {
		buffer.WriteString("counters: [")

		for name, counter := range s.counters {
			buffer.WriteString(name + ": " + strconv.FormatInt(counter.Count(), 10))
		}

		buffer.WriteString("], ")
	}

	if len(s.gauges) > 0 {
		buffer.WriteString("gauges: [")

		for name, gauge := range s.gauges {
			buffer.WriteString(name + ": " + strconv.FormatInt(gauge.Value(), 10))
		}

		buffer.WriteString("], ")
	}

	if len(s.histograms) > 0 {
		buffer.WriteString("histograms: [")

		for name, histogram := range s.histograms {
			buffer.WriteString(name + ": " + strconv.FormatInt(histogram.Count(), 10))
		}

		buffer.WriteString("]")
	}

	return buffer.String()
}
