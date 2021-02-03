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
	"mosn.io/mosn/pkg/types"
)

// UpstreamType represents upstream metrics type
const UpstreamType = "upstream"

//  key in cluster/host
const (
	UpstreamConnectionTotal                        = "connection_total"
	UpstreamConnectionClose                        = "connection_close"
	UpstreamConnectionActive                       = "connection_active"
	UpstreamConnectionConFail                      = "connection_con_fail"
	UpstreamConnectionRetry                        = "connection_retry"
	UpstreamConnectionLocalClose                   = "connection_local_close"
	UpstreamConnectionRemoteClose                  = "connection_remote_close"
	UpstreamConnectionLocalCloseWithActiveRequest  = "connection_local_close_with_active_request"
	UpstreamConnectionRemoteCloseWithActiveRequest = "connection_remote_close_with_active_request"
	UpstreamConnectionCloseNotify                  = "connection_close_notify"
	UpstreamRequestTotal                           = "request_total"
	UpstreamRequestActive                          = "request_active"
	UpstreamRequestLocalReset                      = "request_local_reset"
	UpstreamRequestRemoteReset                     = "request_remote_reset"
	UpstreamRequestTimeout                         = "request_timeout"
	UpstreamRequestFailureEject                    = "request_failure_eject"
	UpstreamRequestPendingOverflow                 = "request_pending_overflow"
	UpstreamRequestDuration                        = "request_duration_time"
	UpstreamRequestDurationTotal                   = "request_duration_time_total"
	UpstreamResponseSuccess                        = "response_success"
	UpstreamResponseFailed                         = "response_failed"
)

//  key in cluster
const (
	UpstreamRequestRetry         = "request_retry"
	UpstreamRequestRetryOverflow = "request_retry_overflow"
	UpstreamLBSubSetsFallBack    = "lb_subsets_fallback"
	UpstreamLBSubsetsCreated     = "lb_subsets_created"
	UpstreamBytesReadTotal       = "connection_bytes_read_total"
	UpstreamBytesReadBuffered    = "connection_bytes_read_buffered"
	UpstreamBytesWriteTotal      = "connection_bytes_write"
	UpstreamBytesWriteBuffered   = "connection_bytes_write_buffered"
)

// NewHostStats returns a stats that namespace contains cluster and host address
func NewHostStats(clusterName string, addr string) types.Metrics {
	metrics, _ := NewMetrics(UpstreamType, map[string]string{"cluster": clusterName, "host": addr})
	return metrics
}

// NewClusterStats returns a stats with namespace prefix cluster
func NewClusterStats(clusterName string) types.Metrics {
	metrics, _ := NewMetrics(UpstreamType, map[string]string{"cluster": clusterName})
	return metrics
}
