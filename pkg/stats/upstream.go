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

package stats

import (
	"fmt"

	"github.com/alipay/sofa-mosn/pkg/types"
)

// UpstreamType represents upstream metrics type
const UpstreamType = "upstream"

//  key in cluster/host
const (
	UpstreamConnectionTotal                        = "upstream_connection_total"
	UpstreamConnectionClose                        = "upstream_connection_close"
	UpstreamConnectionActive                       = "upstream_connection_active"
	UpstreamConnectionConFail                      = "upstream_connection_con_fail"
	UpstreamConnectionRetry                        = "upstream_connection_retry"
	UpstreamConnectionLocalClose                   = "upstream_connection_local_close"
	UpstreamConnectionRemoteClose                  = "upstream_connection_remote_close"
	UpstreamConnectionLocalCloseWithActiveRequest  = "upstream_connection_local_close_with_active_request"
	UpstreamConnectionRemoteCloseWithActiveRequest = "upstream_connection_remote_close_with_active_request"
	UpstreamConnectionCloseNotify                  = "upstream_connection_close_notify"
	UpstreamRequestTotal                           = "upstream_request_request_total"
	UpstreamRequestActive                          = "upstream_request_request_active"
	UpstreamRequestLocalReset                      = "upstream_request_request_local_reset"
	UpstreamRequestRemoteReset                     = "upstream_request_request_remote_reset"
	UpstreamRequestTimeout                         = "upstream_request_request_timeout"
	UpstreamRequestFailureEject                    = "upstream_request_failure_eject"
	UpstreamRequestPendingOverflow                 = "upstream_request_pending_overflow"
)

//  key in cluster
const (
	UpstreamRequestRetry         = "upstream_request_retry"
	UpstreamRequestRetryOverflow = "upstream_request_retry_overfolw"
	UpstreamLBSubSetsFallBack    = "upstream_lb_subsets_fallback"
	UpstreamLBSubSetsActive      = "upstream_lb_subsets_active"
	UpstreamLBSubsetsCreated     = "upstream_lb_subsets_created"
	UpstreamLBSubsetsRemoved     = "upstream_lb_subsets_removed"
	UpstreamBytesReadTotal       = "upstream_connection_bytes_read_total"
	UpstreamBytesReadBuffered    = "upstream_connection_bytes_read_buffered"
	UpstreamBytesWriteTotal      = "upstream_connection_bytes_write"
	UpstreamBytesWriteBuffered   = "upstream_connection_bytes_write_buffered"
)

// NewHostStats returns a stats that namespace contains cluster and host address
func NewHostStats(clusterName string, addr string) types.Metrics {
	namespace := fmt.Sprintf("cluster.%s.host.%s", clusterName, addr)
	return NewStats(UpstreamType, namespace)
}

// NewClusterStats returns a stats with namespace prefix cluster
func NewClusterStats(clusterName string) types.Metrics {
	namespace := fmt.Sprintf("cluster.%s", clusterName)
	return NewStats(UpstreamType, namespace)
}
