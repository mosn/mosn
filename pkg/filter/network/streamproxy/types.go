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

package streamproxy

import (
	"mosn.io/api"
	"time"
)

// Proxy
type Proxy interface {
	api.ReadFilter

	ReadDisableUpstream(disable bool)

	ReadDisableDownstream(disable bool)
}

// ProxyConfig
type ProxyConfig interface {
	GetRouteFromEntries(connection api.Connection) string

	GetIdleTimeout(network string) time.Duration

	GetReadTimeout(network string) time.Duration
}

// UpstreamCallbacks for upstream's callbacks
type UpstreamCallbacks interface {
	api.ReadFilter
	api.ConnectionEventListener
}

// DownstreamCallbacks for downstream's callbacks
type DownstreamCallbacks interface {
	api.ConnectionEventListener
}

// UpstreamFailureReason used to define Upstream Failure Reason
type UpstreamFailureReason string

// Upstream Failure Reason const
const (
	ConnectFailed         UpstreamFailureReason = "ConnectFailed"
	NoHealthyUpstream     UpstreamFailureReason = "NoHealthyUpstream"
	ResourceLimitExceeded UpstreamFailureReason = "ResourceLimitExceeded"
	NoRoute               UpstreamFailureReason = "NoRoute"
)
