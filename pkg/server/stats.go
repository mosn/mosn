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

package server

import (
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/rcrowley/go-metrics"
)

const (
	// ~~~ stats names
	DownstreamConnectionTotal   = "downstream_connection_total"
	DownstreamConnectionDestroy = "downstream_connection_destroy"
	DownstreamConnectionActive  = "downstream_connection_active"
	DownstreamBytesRead         = "downstream_bytes_read"
	DownstreamBytesReadCurrent  = "downstream_bytes_read_current"
	DownstreamBytesWrite        = "downstream_bytes_write"
	DownstreamBytesWriteCurrent = "downstream_bytes_write_current"
)

type ListenerStats struct {
	stats *stats.Stats
}

func newListenerStats(namespace string) *ListenerStats {
	return &ListenerStats{
		stats: initStats(namespace),
	}
}

func initStats(namespace string) *stats.Stats {
	return stats.NewStats(namespace).AddCounter(DownstreamConnectionTotal).AddCounter(DownstreamConnectionDestroy).
		AddCounter(DownstreamConnectionActive).AddCounter(DownstreamBytesRead).
		AddGauge(DownstreamBytesReadCurrent).AddCounter(DownstreamBytesWrite).
		AddGauge(DownstreamBytesWriteCurrent)
}

func (ls *ListenerStats) DownstreamConnectionTotal() metrics.Counter {
	return ls.stats.Counter(DownstreamConnectionTotal)
}

func (ls *ListenerStats) DownstreamConnectionDestroy() metrics.Counter {
	return ls.stats.Counter(DownstreamConnectionDestroy)
}

func (ls *ListenerStats) DownstreamConnectionActive() metrics.Counter {
	return ls.stats.Counter(DownstreamConnectionActive)
}

func (ls *ListenerStats) DownstreamBytesRead() metrics.Counter {
	return ls.stats.Counter(DownstreamBytesRead)
}

func (ls *ListenerStats) DownstreamBytesReadCurrent() metrics.Gauge {
	return ls.stats.Gauge(DownstreamBytesReadCurrent)
}

func (ls *ListenerStats) DownstreamBytesWrite() metrics.Counter {
	return ls.stats.Counter(DownstreamBytesWrite)
}

func (ls *ListenerStats) DownstreamBytesWriteCurrent() metrics.Gauge {
	return ls.stats.Gauge(DownstreamBytesWriteCurrent)
}

func (ls *ListenerStats) String() string {
	return ls.stats.String()
}
