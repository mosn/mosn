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

// DownstreamType represents downstream  metrics type
const DownstreamType = "downstream"

// metrics key in listener/proxy
const (
	DownstreamConnectionTotal    = "downstream_connection_total"
	DownstreamConnectionDestroy  = "downstream_connection_destroy"
	DownstreamConnectionActive   = "downstream_connection_active"
	DownstreamBytesReadTotal     = "downstream_bytes_read_total"
	DownstreamBytesReadBuffered  = "downstream_bytes_read_buffered"
	DownstreamBytesWriteTotal    = "downstream_bytes_write_total"
	DownstreamBytesWriteBuffered = "downstream_bytes_write_buffered"
	DownstreamRequestTotal       = "downstream_request_total"
	DownstreamRequestActive      = "downstream_request_active"
	DownstreamRequestReset       = "downstream_request_reset"
	DownstreamRequestTime        = "downstream_request_time"
)

// NewProxyStats returns a stats with namespace prefix proxy
func NewProxyStats(proxyName string) types.Metrics {
	namespace := fmt.Sprintf("proxy.%s", proxyName)
	return NewStats(DownstreamType, namespace)
}

// NewListenerStats returns a stats with namespace prefix listsener
func NewListenerStats(listenerName string) types.Metrics {
	namespace := fmt.Sprintf("listener.%s", listenerName)
	return NewStats(DownstreamType, namespace)
}
