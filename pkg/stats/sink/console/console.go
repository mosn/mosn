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

package console

import (
	"github.com/rcrowley/go-metrics"
	"strconv"
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/alipay/sofa-mosn/pkg/types"
	"io"
	"encoding/json"
)

// histogram output percents
var percents = []float64{0.5, 0.75, 0.95, 0.99, 0.999}
// NamespaceData represents a namespace's metrics data in string format
type NamespaceData map[string]string

// PromSink extract metrics from stats registry with specified interval
type ConsoleSink struct {
	writer io.Writer
}

// ~ MetricsSink
func (sink *ConsoleSink) Flush(registries []metrics.Registry) {
	// type -> namespace -> key -> value
	all := make(map[string]map[string]NamespaceData)

	for _, registry := range registries {
		registry.Each(func(key string, i interface{}) {
			// TODO registry dimension optimize
			// TODO use dict to replace map
			statsType, namespace, metricsKey := stats.KeySplit(key)

			typeData, ok := all[statsType]
			if !ok {
				typeData = make(map[string]NamespaceData)
				all[statsType] = typeData
			}

			namespaceData, ok := typeData[namespace]
			if !ok {
				namespaceData = NamespaceData{}
				typeData[namespace] = namespaceData
			}
			switch metric := i.(type) {
			case metrics.Counter:
				namespaceData[metricsKey] = strconv.FormatInt(metric.Count(), 10)
			case metrics.Gauge:
				namespaceData[metricsKey] = strconv.FormatInt(metric.Value(), 10)
			case metrics.Histogram:
				h := metric.Snapshot()
				ps := h.Percentiles(percents)
				for index := range percents {
					key := metricsKey + "." + strconv.FormatFloat(percents[index]*100, 'f', 2, 64) + "%"
					namespaceData[key] = strconv.FormatFloat(ps[index], 'f', 2, 64)
				}
				namespaceData[metricsKey+".count"] = strconv.FormatInt(h.Count(), 10)
				namespaceData[metricsKey+".min"] = strconv.FormatInt(h.Min(), 10)
				namespaceData[metricsKey+".max"] = strconv.FormatInt(h.Max(), 10)
				namespaceData[metricsKey+".mean"] = strconv.FormatFloat(h.Mean(), 'f', 2, 64)
				namespaceData[metricsKey+".stddev"] = strconv.FormatFloat(h.StdDev(), 'f', 2, 64)

			default: //unsupport metrics, ignore
				return
			}
		})
	}
	b, _ := json.MarshalIndent(all, "", "\t")
	sink.writer.Write(b)
}

// NewPrometheusProvider returns a Provider that produces Prometheus metrics.
// Namespace and subsystem are applied to all produced metrics.
func NewConsoleSink(writer io.Writer) types.MetricsSink {
	return &ConsoleSink{
		writer: writer,
	}
}
