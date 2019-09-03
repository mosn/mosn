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
	gometrics "github.com/rcrowley/go-metrics"
	"sofastack.io/sofa-mosn/pkg/metrics"
	"sofastack.io/sofa-mosn/pkg/types"
)

type Stats struct {
	DownstreamConnectionTotal   gometrics.Counter
	DownstreamConnectionDestroy gometrics.Counter
	DownstreamConnectionActive  gometrics.Counter
	DownstreamBytesReadTotal    gometrics.Counter
	DownstreamBytesWriteTotal   gometrics.Counter
	DownstreamRequestTotal      gometrics.Counter
	DownstreamRequestActive     gometrics.Counter
	DownstreamRequestReset      gometrics.Counter
	DownstreamRequestTime       gometrics.Histogram
	DownstreamRequestTimeTotal  gometrics.Counter
	DownstreamProcessTime       gometrics.Histogram
	DownstreamProcessTimeTotal  gometrics.Counter
	DownstreamRequestConcurrent *Concurrency
}

func newListenerStats(listenerName string) *Stats {
	s := metrics.NewListenerStats(listenerName)
	return newStats(s)
}
func newProxyStats(proxyName string) *Stats {
	s := metrics.NewProxyStats(proxyName)
	return newStats(s)
}

func newStats(s types.Metrics) *Stats {
	return &Stats{
		DownstreamConnectionTotal:   s.Counter(metrics.DownstreamConnectionTotal),
		DownstreamConnectionDestroy: s.Counter(metrics.DownstreamConnectionDestroy),
		DownstreamConnectionActive:  s.Counter(metrics.DownstreamConnectionActive),
		DownstreamBytesReadTotal:    s.Counter(metrics.DownstreamBytesReadTotal),
		DownstreamBytesWriteTotal:   s.Counter(metrics.DownstreamBytesWriteTotal),
		DownstreamRequestTotal:      s.Counter(metrics.DownstreamRequestTotal),
		DownstreamRequestActive:     s.Counter(metrics.DownstreamRequestActive),
		DownstreamRequestReset:      s.Counter(metrics.DownstreamRequestReset),
		DownstreamRequestTime:       s.Histogram(metrics.DownstreamRequestTime),
		DownstreamRequestTimeTotal:  s.Counter(metrics.DownstreamRequestTimeTotal),
		DownstreamProcessTime:       s.Histogram(metrics.DownstreamProcessTime),
		DownstreamProcessTimeTotal:  s.Counter(metrics.DownstreamProcessTimeTotal),
		DownstreamRequestConcurrent: getOrNewConcurrency(metrics.DownstreamRequestConcurrency, s),
	}
}
