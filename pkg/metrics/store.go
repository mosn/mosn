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
	"strings"
	"sync"

	"fmt"
	"sort"

	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/mosn/pkg/metrics/shm"
	"mosn.io/mosn/pkg/types"
)

const MaxLabelCount = 20

var (
	defaultStore          *store
	defaultMatcher        *metricsMatcher
	ErrLabelCountExceeded = fmt.Errorf("label count exceeded, max is %d", MaxLabelCount)
)

// stats memory store
type store struct {
	matcher *metricsMatcher

	metrics map[string]types.Metrics
	mutex   sync.RWMutex
}

// metrics is a wrapper of go-metrics registry, is an implement of types.Metrics
type metrics struct {
	typ    string
	labels map[string]string

	prefix    string
	labelKeys []string
	labelVals []string

	registry gometrics.Registry
}

func init() {
	defaultMatcher = &metricsMatcher{}

	defaultStore = &store{
		matcher: defaultMatcher,
		// TODO: default length configurable
		metrics: make(map[string]types.Metrics, 100),
	}
}

// SetStatsMatcher sets the exclusion labels and exclusion keys
// if a metrics labels/keys contains in exclusions, it will be ignored
func SetStatsMatcher(all bool, exclusionLabels, exclusionKeys []string) {
	defaultStore.mutex.Lock()
	defer defaultStore.mutex.Unlock()

	defaultStore.matcher = &metricsMatcher{
		rejectAll:       all,
		exclusionLabels: exclusionLabels,
		exclusionKeys:   exclusionKeys,
	}
}

// NewMetrics returns a metrics
// Same (type + labels) pair will leading to the same Metrics instance
func NewMetrics(typ string, labels map[string]string) (types.Metrics, error) {
	if len(labels) > MaxLabelCount {
		return nil, ErrLabelCountExceeded
	}

	defaultStore.mutex.Lock()
	defer defaultStore.mutex.Unlock()

	// support exclusion only
	if defaultStore.matcher.isExclusionLabels(labels) {
		return NewNilMetrics(typ, labels)
	}

	// check existence
	name, keys, values := fullName(typ, labels)
	if m, ok := defaultStore.metrics[name]; ok {
		return m, nil
	}

	stats := &metrics{
		typ:       typ,
		labels:    labels,
		labelKeys: keys,
		labelVals: values,
		prefix:    name + ".",
		registry:  gometrics.NewRegistry(),
	}

	defaultStore.metrics[name] = stats

	return stats, nil
}

func sortedLabels(labels map[string]string) (keys, values []string) {
	keys = make([]string, 0, len(labels))
	values = make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		values = append(values, labels[k])
	}
	return
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
	keys, values = sortedLabels(s.labels)
	s.labelKeys = keys
	s.labelVals = values

	return
}

func (s *metrics) Counter(key string) gometrics.Counter {
	// support exclusion only
	if defaultStore.matcher.isExclusionKey(key) {
		return gometrics.NilCounter{}
	}

	return s.registry.GetOrRegister(key, shm.NewShmCounterFunc(s.fullName(key))).(gometrics.Counter)
}

func (s *metrics) Gauge(key string) gometrics.Gauge {
	// support exclusion only
	if defaultStore.matcher.isExclusionKey(key) {
		return gometrics.NilGauge{}
	}

	return s.registry.GetOrRegister(key, shm.NewShmGaugeFunc(s.fullName(key))).(gometrics.Gauge)
}

func (s *metrics) Histogram(key string) gometrics.Histogram {
	// support exclusion only
	if defaultStore.matcher.isExclusionKey(key) {
		return gometrics.NilHistogram{}
	}

	// TODO: notice the histogram only keeps 100 values as we set
	return s.registry.GetOrRegister(key, func() gometrics.Histogram { return gometrics.NewHistogram(gometrics.NewUniformSample(100)) }).(gometrics.Histogram)
}

func (s *metrics) Each(f func(string, interface{})) {
	s.registry.Each(f)
}

func (s *metrics) UnregisterAll() {
	s.registry.UnregisterAll()
}

func (s *metrics) fullName(name string) string {
	return s.prefix + name
}

// GetAll returns all metrics data
func GetAll() (metrics []types.Metrics) {
	defaultStore.mutex.RLock()
	defer defaultStore.mutex.RUnlock()
	metrics = make([]types.Metrics, 0, len(defaultStore.metrics))
	for _, m := range defaultStore.metrics {
		metrics = append(metrics, m)
	}
	return
}

// GetProxyTotal returns proxy global metrics data
func GetProxyTotal() (metrics types.Metrics) {
	defaultStore.mutex.RLock()
	defer defaultStore.mutex.RUnlock()
	for _, m := range defaultStore.metrics {
		name, _, _ := fullName(m.Type(), m.Labels())
		if name == "downstream.proxy.global" {
			return m
		}
	}
	return nil
}

// ResetAll is only for test and internal usage. DO NOT use this if not sure.
func ResetAll() {
	defaultStore.mutex.Lock()
	defer defaultStore.mutex.Unlock()

	for _, m := range defaultStore.metrics {
		m.UnregisterAll()
	}
	defaultStore.metrics = make(map[string]types.Metrics, 100)
	defaultStore.matcher = defaultMatcher
}

func fullName(typ string, labels map[string]string) (fullName string, keys, values []string) {
	keys, values = sortedLabels(labels)

	pair := make([]string, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		pair = append(pair, keys[i]+"."+values[i])
	}
	fullName = typ + "." + strings.Join(pair, ".")
	return
}
