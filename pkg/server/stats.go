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
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/rcrowley/go-metrics"
)

// ListenerStats is stats for listener
type ListenerStats struct {
	stats types.Metrics
}

func newListenerStats(listenername string) *ListenerStats {
	return &ListenerStats{
		stats: stats.NewListenerStats(listenername),
	}
}

// DownstreamConnectionTotal return downstream connection total number
func (ls *ListenerStats) DownstreamConnectionTotal() metrics.Counter {
	return ls.stats.Counter(stats.DownstreamConnectionTotal)
}

// DownstreamConnectionDestroy return downstream connection destroy number
func (ls *ListenerStats) DownstreamConnectionDestroy() metrics.Counter {
	return ls.stats.Counter(stats.DownstreamConnectionDestroy)
}

// DownstreamConnectionActive return downstream connection active number
func (ls *ListenerStats) DownstreamConnectionActive() metrics.Counter {
	return ls.stats.Counter(stats.DownstreamConnectionActive)
}

// DownstreamBytesRead return downstream bytes read number
func (ls *ListenerStats) DownstreamBytesRead() metrics.Counter {
	return ls.stats.Counter(stats.DownstreamBytesRead)
}

// DownstreamBytesReadCurrent return downstream bytes read current number
func (ls *ListenerStats) DownstreamBytesReadCurrent() metrics.Gauge {
	return ls.stats.Gauge(stats.DownstreamBytesReadCurrent)
}

// DownstreamBytesWrite return downstream bytes write number
func (ls *ListenerStats) DownstreamBytesWrite() metrics.Counter {
	return ls.stats.Counter(stats.DownstreamBytesWrite)
}

// DownstreamBytesWriteCurrent return downstream bytes write number
func (ls *ListenerStats) DownstreamBytesWriteCurrent() metrics.Gauge {
	return ls.stats.Gauge(stats.DownstreamBytesWriteCurrent)
}
