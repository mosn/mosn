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
	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/mosn/pkg/types"
)

// NilMetrics is an implementation of types.Metrics
// it stores nothing except the metrics type and labels
type NilMetrics struct {
	*metrics
}

func NewNilMetrics(typ string, labels map[string]string) (types.Metrics, error) {
	return &NilMetrics{
		metrics: &metrics{
			typ:    typ,
			labels: labels,
		},
	}, nil
}

func (m *NilMetrics) Counter(key string) gometrics.Counter {
	return gometrics.NilCounter{}
}

func (m *NilMetrics) Gauge(key string) gometrics.Gauge {
	return gometrics.NilGauge{}
}

func (m *NilMetrics) Histogram(key string) gometrics.Histogram {
	return gometrics.NilHistogram{}
}

func (m *NilMetrics) Each(f func(string, interface{})) {
	// do nothing
}

func (m *NilMetrics) UnregisterAll() {
	// do nothing
}
