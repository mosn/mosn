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
	"encoding/json"
	"io"
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/rcrowley/go-metrics"
	"strings"
)

// histogram output percents
var percents = []float64{0.5, 0.75, 0.95, 0.99, 0.999}

// NamespaceData represents a namespace's metrics data in string format
type NamespaceData map[string]string

type consoleSink struct {
	writer io.Writer
}

// ~ MetricsSink
func (sink *consoleSink) Flush(ms []types.Metrics) {
	// type -> namespace -> key -> value
	all := make(map[string]map[string]NamespaceData)

	for _, m := range ms {
		typeData, ok := all[m.Type()]
		if !ok {
			typeData = make(map[string]NamespaceData)
			all[m.Type()] = typeData
		}

		namespace := makeNamespace(m.SortedLabels())
		namespaceData, ok := typeData[namespace]
		if !ok {
			namespaceData = NamespaceData{}
			typeData[namespace] = namespaceData
		}

		m.Each(func(key string, i interface{}) {
			switch metric := i.(type) {
			case metrics.Counter:
				namespaceData[key] = strconv.FormatInt(metric.Count(), 10)
			case metrics.Gauge:
				namespaceData[key] = strconv.FormatInt(metric.Value(), 10)
			case metrics.Histogram:
				h := metric.Snapshot()
				ps := h.Percentiles(percents)
				for index := range percents {
					key := key + "." + strconv.FormatFloat(percents[index]*100, 'f', 2, 64) + "%"
					namespaceData[key] = strconv.FormatFloat(ps[index], 'f', 2, 64)
				}
				namespaceData[key+".min"] = strconv.FormatInt(h.Min(), 10)
				namespaceData[key+".max"] = strconv.FormatInt(h.Max(), 10)
			default: //unsupport metrics, ignore
				return
			}
		})
	}
	//TODO: performance optimize
	b, _ := json.MarshalIndent(all, "", "\t")
	sink.writer.Write(b)
}

// NewConsoleSink returns sink that convert metrics into human readable format
// Note: This func is not registered into sink factory, and should be use in certain scene.
func NewConsoleSink(writer io.Writer) types.MetricsSink {
	return &consoleSink{
		writer: writer,
	}
}

func makeNamespace(keys, vals []string) (namespace string) {
	pair := make([]string, 0, len(keys))

	for i := 0; i < len(vals); i++ {
		pair = append(pair, keys[i]+"."+vals[i])
	}
	return strings.Join(pair, ".")
}
