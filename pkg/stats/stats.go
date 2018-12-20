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
	"sort"
	"strings"
	"sync"

	"github.com/rcrowley/go-metrics"
)

type registry struct {
	registries map[string]metrics.Registry
	mutex      sync.RWMutex
}

var reg *registry

func init() {
	reg = &registry{
		registries: make(map[string]metrics.Registry),
		mutex:      sync.RWMutex{},
	}
}

// Stats is a wrapper of go-metrics registry, is an implement of types.Metrics
type Stats struct {
	typ       string
	namespace string
	registry  metrics.Registry
}

const sep = "@"

// NewStats returns a Stats
// metrics key prefix is "${type}.${namespace}"
// "@" is the reserved sep, any "@" in type and namespace will be dropped
func NewStats(typ, namespace string) *Stats {
	typ = strings.Replace(typ, sep, "", -1)
	namespace = strings.Replace(namespace, sep, "", -1)
	reg.mutex.Lock()
	defer reg.mutex.Unlock()
	if _, ok := reg.registries[typ]; !ok {
		reg.registries[typ] = metrics.NewRegistry()
	}
	return &Stats{
		typ:       typ,
		namespace: namespace,
		registry:  reg.registries[typ],
	}

}

// Counter creates or returns a go-metrics counter by key
// if the key is registered by other interface, it will be panic
func (s *Stats) Counter(key string) metrics.Counter {
	metricsKey := strings.Join([]string{s.typ, s.namespace, key}, sep)
	return s.registry.GetOrRegister(metricsKey, metrics.NewCounter).(metrics.Counter)
}

// Gauge creates or returns a go-metrics gauge by key
// if the key is registered by other interface, it will be panic
func (s *Stats) Gauge(key string) metrics.Gauge {
	metricsKey := strings.Join([]string{s.typ, s.namespace, key}, sep)
	return s.registry.GetOrRegister(metricsKey, metrics.NewGauge).(metrics.Gauge)
}

// Histogram creates or returns a go-metrics histogram by key
// if the key is registered by other interface, it will be panic
func (s *Stats) Histogram(key string) metrics.Histogram {
	metricsKey := strings.Join([]string{s.typ, s.namespace, key}, sep)
	return s.registry.GetOrRegister(metricsKey, func() metrics.Histogram { return metrics.NewHistogram(metrics.NewUniformSample(100)) }).(metrics.Histogram)
}

// LisTypes returns all registered types, sorted by dictionary order
func LisTypes() (ts []string) {
	reg.mutex.RLock()
	for key := range reg.registries {
		ts = append(ts, key)
	}
	reg.mutex.RUnlock()
	sort.Strings(ts)
	return
}

// GetAllRegistries returns all registries
func GetAllRegistries() (regs []metrics.Registry) {
	reg.mutex.RLock()
	for _, reg := range reg.registries {
		regs = append(regs, reg)
	}
	reg.mutex.RUnlock()
	return
}

func KeySplit(key string) (statsType, namespace, name string) {
	values := strings.SplitN(key, sep, 3)
	if len(values) != 3 { // unexepcted metrics, ignore
		return
	}
	return values[0], values[1], values[2]
}

// ResetAll is only for test and internal usage. DO NOT use this if not sure.
func ResetAll() {
	for _, r := range reg.registries {
		r.UnregisterAll()
	}
	reg.registries = make(map[string]metrics.Registry)
}
