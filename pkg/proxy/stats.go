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
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/types"
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
	DownstreamRequestTimeTotal  gometrics.Counter
	DownstreamProcessTimeTotal  gometrics.Counter
	DownstreamRequestFailed     gometrics.Counter
	DownstreamRequest200Total   gometrics.Counter
	DownstreamRequest206Total   gometrics.Counter
	DownstreamRequest302Total   gometrics.Counter
	DownstreamRequest304Total   gometrics.Counter
	DownstreamRequest400Total   gometrics.Counter
	DownstreamRequest403Total   gometrics.Counter
	DownstreamRequest404Total   gometrics.Counter
	DownstreamRequest416Total   gometrics.Counter
	DownstreamRequest499Total   gometrics.Counter
	DownstreamRequest500Total   gometrics.Counter
	DownstreamRequest502Total   gometrics.Counter
	DownstreamRequest503Total   gometrics.Counter
	DownstreamRequest504Total   gometrics.Counter
	DownstreamRequestOtherTotal gometrics.Counter
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
		DownstreamRequestTimeTotal:  s.Counter(metrics.DownstreamRequestTimeTotal),
		DownstreamProcessTimeTotal:  s.Counter(metrics.DownstreamProcessTimeTotal),
		DownstreamRequestFailed:     s.Counter(metrics.DownstreamRequestFailed),
		DownstreamRequest200Total:   s.Counter(metrics.DownstreamRequest200Total),
		DownstreamRequest206Total:   s.Counter(metrics.DownstreamRequest206Total),
		DownstreamRequest302Total:   s.Counter(metrics.DownstreamRequest302Total),
		DownstreamRequest304Total:   s.Counter(metrics.DownstreamRequest304Total),
		DownstreamRequest400Total:   s.Counter(metrics.DownstreamRequest400Total),
		DownstreamRequest403Total:   s.Counter(metrics.DownstreamRequest403Total),
		DownstreamRequest404Total:   s.Counter(metrics.DownstreamRequest404Total),
		DownstreamRequest416Total:   s.Counter(metrics.DownstreamRequest416Total),
		DownstreamRequest499Total:   s.Counter(metrics.DownstreamRequest499Total),
		DownstreamRequest500Total:   s.Counter(metrics.DownstreamRequest500Total),
		DownstreamRequest502Total:   s.Counter(metrics.DownstreamRequest502Total),
		DownstreamRequest503Total:   s.Counter(metrics.DownstreamRequest503Total),
		DownstreamRequest504Total:   s.Counter(metrics.DownstreamRequest504Total),
		DownstreamRequestOtherTotal: s.Counter(metrics.DownstreamRequestOtherTotal),
	}
}

func (s *Stats) DownstreamUpdateRequestCode(returnCode int) {
	switch returnCode {
	case 200:
		s.DownstreamRequest200Total.Inc(1)
	case 206:
		s.DownstreamRequest206Total.Inc(1)
	case 302:
		s.DownstreamRequest302Total.Inc(1)
	case 304:
		s.DownstreamRequest304Total.Inc(1)
	case 400:
		s.DownstreamRequest400Total.Inc(1)
	case 403:
		s.DownstreamRequest403Total.Inc(1)
	case 404:
		s.DownstreamRequest404Total.Inc(1)
	case 416:
		s.DownstreamRequest416Total.Inc(1)
	case 499:
		s.DownstreamRequest499Total.Inc(1)
	case 500:
		s.DownstreamRequest500Total.Inc(1)
	case 502:
		s.DownstreamRequest502Total.Inc(1)
	case 503:
		s.DownstreamRequest503Total.Inc(1)
	case 504:
		s.DownstreamRequest504Total.Inc(1)
	default:
		s.DownstreamRequestOtherTotal.Inc(1)
	}
}
