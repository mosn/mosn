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

package metrics

import (
	"sync"

	"fmt"
	"sort"

	"github.com/alipay/sofa-mosn/pkg/types"
	gometrics "github.com/rcrowley/go-metrics"
)

const maxLabelCount = 10

var (
	defaultStore *store

	errLabelCountExceeded = fmt.Errorf("label count exceeded, max is %d", maxLabelCount)
)

// stats memory store
type store struct {
	rejectAll       bool
	exclusionLabels []string
	metrics         []types.Metrics
	mutex           sync.RWMutex
}

// metrics is a wrapper of go-metrics registry, is an implement of types.Metrics
type metrics struct {
	typ       string
	labels    map[string]string
	labelKeys []string
	labelVals []string

	registry gometrics.Registry
}

func init() {
	defaultStore = &store{
		// TODO: default length configurable
		metrics: make([]types.Metrics, 0, 100),
	}
}

// SetStatsMatcher sets the exclusion labels
// if a metrics labels contains in exclusions, it will be ignored
func SetStatsMatcher(all bool, exclusions []string) {
	defaultStore.mutex.Lock()
	defer defaultStore.mutex.Unlock()
	if all {
		defaultStore.rejectAll = true
	}
	defaultStore.exclusionLabels = exclusions
}

// isExclusion returns the labels will be ignored or not
func isExclusion(labels map[string]string) bool {
	defaultStore.mutex.RLock()
	defer defaultStore.mutex.RUnlock()
	if defaultStore.rejectAll {
		return true
	}
	// TODO: support pattern
	for _, label := range defaultStore.exclusionLabels {
		if _, ok := labels[label]; ok {
			return true
		}
	}
	return false
}

// NewMetrics returns a metrics
// Same (type + labels) pair will leading to the same Metrics instance
func NewMetrics(typ string, labels map[string]string) (types.Metrics, error) {
	if len(labels) > maxLabelCount {
		return nil, errLabelCountExceeded
	}
	// support exclusion only
	if isExclusion(labels) {
		return NewNilMetrics(typ, labels)
	}

	defaultStore.mutex.Lock()
	defer defaultStore.mutex.Unlock()

	// check existence
	for _, metric := range defaultStore.metrics {
		if metric.Type() == typ && mapEqual(metric.Labels(), labels) {
			return metric, nil
		}
	}

	stats := &metrics{
		typ:      typ,
		labels:   labels,
		registry: gometrics.NewRegistry(),
	}

	defaultStore.metrics = append(defaultStore.metrics, stats)

	return stats, nil
}

func (s *metrics) Type() string {
	return s.typ
}

func (s *metrics) Labels() map[string]string {
	return s.labels
}

func (s *metrics) SortedLabels() (keys, values []string) {
	if s.labelKeys != nil && s.labelVals != nil {
		return s.labelKeys, s.labelVals
	}

	for k := range s.labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		values = append(values, s.labels[k])
	}
	return
}

func (s *metrics) Counter(key string) gometrics.Counter {
	return s.registry.GetOrRegister(key, gometrics.NewCounter).(gometrics.Counter)
}

func (s *metrics) Gauge(key string) gometrics.Gauge {
	return s.registry.GetOrRegister(key, gometrics.NewGauge).(gometrics.Gauge)
}

func (s *metrics) Histogram(key string) gometrics.Histogram {
	return s.registry.GetOrRegister(key, func() gometrics.Histogram { return gometrics.NewHistogram(gometrics.NewUniformSample(100)) }).(gometrics.Histogram)
}

func (s *metrics) Each(f func(string, interface{})) {
	s.registry.Each(f)
}

func (s *metrics) UnregisterAll() {
	s.registry.UnregisterAll()
}

// GetAll returns all metrics data
func GetAll() (metrics []types.Metrics) {
	defaultStore.mutex.RLock()
	defer defaultStore.mutex.RUnlock()
	return defaultStore.metrics
}

// ResetAll is only for test and internal usage. DO NOT use this if not sure.
func ResetAll() {
	defaultStore.mutex.Lock()
	defer defaultStore.mutex.Unlock()

	for _, m := range defaultStore.metrics {
		m.UnregisterAll()
	}
	defaultStore.metrics = defaultStore.metrics[:0]
	defaultStore.rejectAll = false
	defaultStore.exclusionLabels = nil
}

func mapEqual(x, y map[string]string) bool {
	if len(x) != len(y) {
		return false
	}
	for k, xv := range x {
		if yv, ok := y[k]; !ok || yv != xv {
			return false
		}
	}
	return true
}
