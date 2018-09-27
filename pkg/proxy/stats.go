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
	metrics "github.com/rcrowley/go-metrics"
)

type Stats struct {
	DownstreamConnectionTotal   metrics.Counter
	DownstreamConnectionDestroy metrics.Counter
	DownstreamConnectionActive  metrics.Counter
	DownstreamBytesReadTotal    metrics.Counter
	DownstreamBytesWriteTotal   metrics.Counter
	DownstreamRequestTotal      metrics.Counter
	DownstreamRequestActive     metrics.Counter
	DownstreamRequestReset      metrics.Counter
	DownstreamRequestTime       metrics.Histogram
}

func newListenerStats(listenerName string) *Stats {
	s := stats.NewListenerStats(listenerName)
	return newStats(s)
}
func newProxyStats(proxyName string) *Stats {
	s := stats.NewProxyStats(proxyName)
	return newStats(s)
}

func newStats(s types.Metrics) *Stats {
	return &Stats{
		DownstreamConnectionTotal:   s.Counter(stats.DownstreamConnectionTotal),
		DownstreamConnectionDestroy: s.Counter(stats.DownstreamConnectionDestroy),
		DownstreamConnectionActive:  s.Counter(stats.DownstreamConnectionActive),
		DownstreamBytesReadTotal:    s.Counter(stats.DownstreamBytesReadTotal),
		DownstreamBytesWriteTotal:   s.Counter(stats.DownstreamBytesWriteTotal),
		DownstreamRequestTotal:      s.Counter(stats.DownstreamRequestTotal),
		DownstreamRequestActive:     s.Counter(stats.DownstreamRequestActive),
		DownstreamRequestReset:      s.Counter(stats.DownstreamRequestReset),
		DownstreamRequestTime:       s.Histogram(stats.DownstreamRequestTime),
	}
}
