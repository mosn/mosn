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

// DownstreamType represents downstream  metrics type
const DownstreamType = "downstream"

// metrics key in listener/proxy
const (
	DownstreamConnectionTotal    = "connection_total"
	DownstreamConnectionDestroy  = "connection_destroy"
	DownstreamConnectionActive   = "connection_active"
	DownstreamBytesReadTotal     = "bytes_read_total"
	DownstreamBytesReadBuffered  = "bytes_read_buffered"
	DownstreamBytesWriteTotal    = "bytes_write_total"
	DownstreamBytesWriteBuffered = "bytes_write_buffered"
	DownstreamRequestTotal       = "request_total"
	DownstreamRequestActive      = "request_active"
	DownstreamRequestReset       = "request_reset"
	DownstreamRequestTime        = "request_time"
	DownstreamRequestTimeTotal   = "request_time_total"
	DownstreamProcessTime        = "process_time"
	DownstreamProcessTimeTotal   = "process_time_total"
	DownstreamRequestFailed      = "request_failed"
	DownstreamRequest200Total    = "request_200_total"
	DownstreamRequest206Total    = "request_206_total"
	DownstreamRequest302Total    = "request_302_total"
	DownstreamRequest304Total    = "request_304_total"
	DownstreamRequest400Total    = "request_400_total"
	DownstreamRequest403Total    = "request_403_total"
	DownstreamRequest404Total    = "request_404_total"
	DownstreamRequest416Total    = "request_416_total"
	DownstreamRequest499Total    = "request_499_total"
	DownstreamRequest500Total    = "request_500_total"
	DownstreamRequest502Total    = "request_502_total"
	DownstreamRequest503Total    = "request_503_total"
	DownstreamRequest504Total    = "request_504_total"
	DownstreamRequestOtherTotal  = "request_other_code"
)

// NewProxyStats returns a stats with namespace prefix proxy
func NewProxyStats(proxyName string) types.Metrics {
	metrics, _ := NewMetrics(DownstreamType, map[string]string{"proxy": proxyName})
	return metrics
}

// NewListenerStats returns a stats with namespace prefix listsener
func NewListenerStats(listenerName string) types.Metrics {
	metrics, _ := NewMetrics(DownstreamType, map[string]string{"listener": listenerName})
	return metrics
}
