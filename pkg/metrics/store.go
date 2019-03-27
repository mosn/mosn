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

	"github.com/alipay/sofa-mosn/pkg/types"
	gometrics "github.com/rcrowley/go-metrics"
	"github.com/alipay/sofa-mosn/pkg/metrics/shm"
	"log"
	"github.com/alipay/sofa-mosn/pkg/server/keeper"
)

const maxLabelCount = 10

var (
	defaultStore          *store
	defaultMatcher        *metricsMatcher
	errLabelCountExceeded = fmt.Errorf("label count exceeded, max is %d", maxLabelCount)
)

// stats memory store
type store struct {
	matcher *metricsMatcher

	metrics map[string]types.Metrics
	mutex   sync.RWMutex

	zone shm.MetricsZone
}

// metrics is a wrapper of go-metrics registry, is an implement of types.Metrics
type metrics struct {
	typ    string
	labels map[string]string

	prefix    string
	labelKeys []string
	labelVals []string

	ref map[string]interface{}
}

func init() {
	defaultMatcher = &metricsMatcher{}

	defaultStore = &store{
		matcher: defaultMatcher,
		// TODO: default length configurable
		metrics: make(map[string]types.Metrics, 100),
	}

	zone, err := shm.NewSharedMetrics("metrics", 10*1024*1024)
	if err != nil {
		log.Fatalln("open shared memory for metrics failed:", err)
	}

	defaultStore.zone = zone

	keeper.OnProcessShutDown(func() error {
		zone.Free()
		return nil
	})

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
	if len(labels) > maxLabelCount {
		return nil, errLabelCountExceeded
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
		ref:       make(map[string]interface{}),
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

	entry, err := defaultStore.zone.AllocEntry(s.fullName(key))
	if err != nil {
		// log & stats
		return gometrics.NilCounter{}
	}

	counter := shm.NewShmCounter(entry)
	s.ref[key] = counter
	return counter
}

func (s *metrics) Gauge(key string) gometrics.Gauge {
	// support exclusion only
	if defaultStore.matcher.isExclusionKey(key) {
		return gometrics.NilGauge{}
	}

	entry, err := defaultStore.zone.AllocEntry(s.fullName(key))
	if err != nil {
		// log & stats
		return gometrics.NilGauge{}
	}

	gauge := shm.NewShmGauge(entry)
	s.ref[key] = gauge
	return gauge
}

func (s *metrics) Histogram(key string) gometrics.Histogram {
	return gometrics.NilHistogram{}

	// support exclusion only
	//if defaultStore.matcher.isExclusionKey(key) {
	//	return gometrics.NilHistogram{}
	//}
	//return s.registry.GetOrRegister(key, func() gometrics.Histogram { return gometrics.NewHistogram(gometrics.NewUniformSample(100)) }).(gometrics.Histogram)
}

func (s *metrics) Each(f func(string, interface{})) {
	for k, v := range s.ref {
		f(k, v)
	}
}

func (s *metrics) UnregisterAll() {
	//for k, v := range s.ref {
	//	f(k, v)
	//}

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
