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

package proxy

import (
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/rcrowley/go-metrics"
)

type proxyStats struct {
	stats types.Metrics
}

func newProxyStats(proxyname string) *proxyStats {
	return &proxyStats{
		stats: stats.NewProxyStats(proxyname),
	}
}

func (s *proxyStats) DownstreamConnectionTotal() metrics.Counter {
	return s.stats.Counter(stats.DownstreamConnectionTotal)
}

func (s *proxyStats) DownstreamConnectionDestroy() metrics.Counter {
	return s.stats.Counter(stats.DownstreamConnectionDestroy)
}

func (s *proxyStats) DownstreamConnectionActive() metrics.Counter {
	return s.stats.Counter(stats.DownstreamConnectionActive)
}

func (s *proxyStats) DownstreamBytesRead() metrics.Counter {
	return s.stats.Counter(stats.DownstreamBytesRead)
}

func (s *proxyStats) DownstreamBytesReadCurrent() metrics.Gauge {
	return s.stats.Gauge(stats.DownstreamBytesReadCurrent)
}

func (s *proxyStats) DownstreamBytesWrite() metrics.Counter {
	return s.stats.Counter(stats.DownstreamBytesWrite)
}

func (s *proxyStats) DownstreamBytesWriteCurrent() metrics.Gauge {
	return s.stats.Gauge(stats.DownstreamBytesWriteCurrent)
}

func (s *proxyStats) DownstreamRequestTotal() metrics.Counter {
	return s.stats.Counter(stats.DownstreamRequestTotal)
}

func (s *proxyStats) DownstreamRequestActive() metrics.Counter {
	return s.stats.Counter(stats.DownstreamRequestActive)
}

func (s *proxyStats) DownstreamRequestReset() metrics.Counter {
	return s.stats.Counter(stats.DownstreamRequestReset)
}

func (s *proxyStats) DownstreamRequestTime() metrics.Histogram {
	return s.stats.Histogram(stats.DownstreamRequestTime)
}

type listenerStats struct {
	stats types.Metrics
}

func newListenerStats(listenername string) *listenerStats {
	return &listenerStats{
		stats: stats.NewListenerStats(listenername),
	}
}

func (s *listenerStats) DownstreamRequestTotal() metrics.Counter {
	return s.stats.Counter(stats.DownstreamRequestTotal)
}

func (s *listenerStats) DownstreamRequestActive() metrics.Counter {
	return s.stats.Counter(stats.DownstreamRequestActive)
}

func (s *listenerStats) DownstreamRequestReset() metrics.Counter {
	return s.stats.Counter(stats.DownstreamRequestReset)
}

func (s *listenerStats) DownstreamRequestTime() metrics.Histogram {
	return s.stats.Histogram(stats.DownstreamRequestTime)
}
