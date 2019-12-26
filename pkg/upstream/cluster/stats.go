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

package cluster

import (
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/types"
)

func newHostStats(clustername string, addr string) types.HostStats {
	s := metrics.NewHostStats(clustername, addr)

	return types.HostStats{
		UpstreamConnectionTotal:                        s.Counter(metrics.UpstreamConnectionTotal),
		UpstreamConnectionClose:                        s.Counter(metrics.UpstreamConnectionClose),
		UpstreamConnectionActive:                       s.Counter(metrics.UpstreamConnectionActive),
		UpstreamConnectionConFail:                      s.Counter(metrics.UpstreamConnectionConFail),
		UpstreamConnectionLocalClose:                   s.Counter(metrics.UpstreamConnectionLocalClose),
		UpstreamConnectionRemoteClose:                  s.Counter(metrics.UpstreamConnectionRemoteClose),
		UpstreamConnectionLocalCloseWithActiveRequest:  s.Counter(metrics.UpstreamConnectionLocalCloseWithActiveRequest),
		UpstreamConnectionRemoteCloseWithActiveRequest: s.Counter(metrics.UpstreamConnectionRemoteCloseWithActiveRequest),
		UpstreamConnectionCloseNotify:                  s.Counter(metrics.UpstreamConnectionCloseNotify),
		UpstreamRequestTotal:                           s.Counter(metrics.UpstreamRequestTotal),
		UpstreamRequestActive:                          s.Counter(metrics.UpstreamRequestActive),
		UpstreamRequestLocalReset:                      s.Counter(metrics.UpstreamRequestLocalReset),
		UpstreamRequestRemoteReset:                     s.Counter(metrics.UpstreamRequestRemoteReset),
		UpstreamRequestTimeout:                         s.Counter(metrics.UpstreamRequestTimeout),
		UpstreamRequestFailureEject:                    s.Counter(metrics.UpstreamRequestFailureEject),
		UpstreamRequestPendingOverflow:                 s.Counter(metrics.UpstreamRequestPendingOverflow),
		UpstreamRequestDuration:                        s.Histogram(metrics.UpstreamRequestDuration),
		UpstreamRequestDurationTotal:                   s.Counter(metrics.UpstreamRequestDurationTotal),
		UpstreamResponseSuccess:                        s.Counter(metrics.UpstreamResponseSuccess),
		UpstreamResponseFailed:                         s.Counter(metrics.UpstreamResponseFailed),
	}
}

func newClusterStats(clustername string) types.ClusterStats {
	s := metrics.NewClusterStats(clustername)
	return types.ClusterStats{
		UpstreamConnectionTotal:                        s.Counter(metrics.UpstreamConnectionTotal),
		UpstreamConnectionClose:                        s.Counter(metrics.UpstreamConnectionClose),
		UpstreamConnectionActive:                       s.Counter(metrics.UpstreamConnectionActive),
		UpstreamConnectionConFail:                      s.Counter(metrics.UpstreamConnectionConFail),
		UpstreamConnectionRetry:                        s.Counter(metrics.UpstreamConnectionRetry),
		UpstreamConnectionLocalClose:                   s.Counter(metrics.UpstreamConnectionLocalClose),
		UpstreamConnectionRemoteClose:                  s.Counter(metrics.UpstreamConnectionRemoteClose),
		UpstreamConnectionLocalCloseWithActiveRequest:  s.Counter(metrics.UpstreamConnectionLocalCloseWithActiveRequest),
		UpstreamConnectionRemoteCloseWithActiveRequest: s.Counter(metrics.UpstreamConnectionRemoteCloseWithActiveRequest),
		UpstreamConnectionCloseNotify:                  s.Counter(metrics.UpstreamConnectionCloseNotify),
		UpstreamBytesReadTotal:                         s.Counter(metrics.UpstreamBytesReadTotal),
		UpstreamBytesWriteTotal:                        s.Counter(metrics.UpstreamBytesWriteTotal),
		UpstreamRequestTotal:                           s.Counter(metrics.UpstreamRequestTotal),
		UpstreamRequestActive:                          s.Counter(metrics.UpstreamRequestActive),
		UpstreamRequestLocalReset:                      s.Counter(metrics.UpstreamRequestLocalReset),
		UpstreamRequestRemoteReset:                     s.Counter(metrics.UpstreamRequestRemoteReset),
		UpstreamRequestRetry:                           s.Counter(metrics.UpstreamRequestRetry),
		UpstreamRequestRetryOverflow:                   s.Counter(metrics.UpstreamRequestRetryOverflow),
		UpstreamRequestTimeout:                         s.Counter(metrics.UpstreamRequestTimeout),
		UpstreamRequestFailureEject:                    s.Counter(metrics.UpstreamRequestFailureEject),
		UpstreamRequestPendingOverflow:                 s.Counter(metrics.UpstreamRequestPendingOverflow),
		UpstreamRequestDuration:                        s.Histogram(metrics.UpstreamRequestDuration),
		UpstreamRequestDurationTotal:                   s.Counter(metrics.UpstreamRequestDurationTotal),
		UpstreamResponseSuccess:                        s.Counter(metrics.UpstreamResponseSuccess),
		UpstreamResponseFailed:                         s.Counter(metrics.UpstreamResponseFailed),
		LBSubSetsFallBack:                              s.Counter(metrics.UpstreamLBSubSetsFallBack),
		LBSubsetsCreated:                               s.Gauge(metrics.UpstreamLBSubsetsCreated),
	}
}
