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

package types

import (
	"io"

	"github.com/rcrowley/go-metrics"
)

// Metrics is a wrapper interface for go-metrics
// support Counter, Gauge Histogram
type Metrics interface {
	// Type returns metrics logical type, e.g. 'downstream'/'upstream', this is more like the Subsystem concept
	Type() string

	// Labels used to distinguish the metrics' owner for same metrics key set, like 'cluster: local_service'
	Labels() map[string]string

	// SortedLabels return keys and vals in stable order
	SortedLabels() (keys, vals []string)

	// Counter creates or returns a go-metrics counter by key
	// if the key is registered by other interface, it will be panic
	Counter(key string) metrics.Counter

	// Gauge creates or returns a go-metrics gauge by key
	// if the key is registered by other interface, it will be panic
	Gauge(key string) metrics.Gauge

	// Histogram creates or returns a go-metrics histogram by key
	// if the key is registered by other interface, it will be panic
	Histogram(key string) metrics.Histogram

	// Each call the given function for each registered metric.
	Each(func(string, interface{}))

	// UnregisterAll unregister all metrics.  (Mostly for testing.)
	UnregisterAll()
}

// MetricsSink flush metrics to backend storage
type MetricsSink interface {
	// Flush flush given metrics
	Flush(writer io.Writer, metrics []Metrics)
}
