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
	"strconv"
	"strings"
	"sync"

	metrics "github.com/rcrowley/go-metrics"
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

// histogram output percents
var percents = []float64{0.5, 0.75, 0.95, 0.99, 0.999}

// NamespaceData represents a namespace's metrics data in string format
type NamespaceData map[string]string

// GetMetricsData returns a type of registry data as map
// map's key is namespace, value is the namespace's all metrics
func GetMetricsData(typ string) map[string]NamespaceData {
	var r metrics.Registry
	var ok bool
	reg.mutex.RLock()
	r, ok = reg.registries[typ]
	reg.mutex.RUnlock()
	if !ok {
		// no such type
		return nil
	}
	res := make(map[string]NamespaceData)
	r.Each(func(key string, i interface{}) {
		values := strings.SplitN(key, sep, 3)
		if len(values) != 3 { // unexepcted metrics, ignore
			return
		}
		namespace := values[1]
		metricsKey := values[2]
		data, ok := res[namespace]
		if !ok {
			data = NamespaceData{}
		}
		switch metric := i.(type) {
		case metrics.Counter:
			data[metricsKey] = strconv.FormatInt(metric.Count(), 10)
		case metrics.Gauge:
			data[metricsKey] = strconv.FormatInt(metric.Value(), 10)
		case metrics.Histogram:
			h := metric.Snapshot()
			ps := h.Percentiles(percents)
			for index := range percents {
				key := metricsKey + "." + strconv.FormatFloat(percents[index]*100, 'f', 2, 64) + "%"
				data[key] = strconv.FormatFloat(ps[index], 'f', 2, 64)
			}
			data[metricsKey+".count"] = strconv.FormatInt(h.Count(), 10)
			data[metricsKey+".min"] = strconv.FormatInt(h.Min(), 10)
			data[metricsKey+".max"] = strconv.FormatInt(h.Max(), 10)
			data[metricsKey+".mean"] = strconv.FormatFloat(h.Mean(), 'f', 2, 64)
			data[metricsKey+".stddev"] = strconv.FormatFloat(h.StdDev(), 'f', 2, 64)

		default: //unsupport metrics, ignore
			return
		}
		res[namespace] = data
	})
	return res
}

// GetAllMetricsData returns all registered metrics data
// the first map is "type" to namespace map
// the namespace map is "namespace" to metrics data
func GetAllMetricsData() map[string]map[string]NamespaceData {
	res := make(map[string]map[string]NamespaceData)
	ts := LisTypes()
	for _, typ := range ts {
		res[typ] = GetMetricsData(typ)
	}
	return res
}
