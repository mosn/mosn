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
	"time"

	"github.com/alipay/sofa-mosn/pkg/types"
)

// Proxy
type Proxy interface {
	types.ReadFilter

	ReadDisableUpstream(disable bool)

	ReadDisableDownstream(disable bool)
}

// UpstreamCallbacks
// callback invoked when upstream event happened
type UpstreamCallbacks interface {
	types.ReadFilter
	types.ConnectionEventListener
}

// Timeout
type Timeout struct {
	GlobalTimeout time.Duration
	TryTimeout    time.Duration
}

// UpstreamFailureReason
type UpstreamFailureReason string

// Group pf some Upstream Failure Reason
const (
	ConnectFailed         UpstreamFailureReason = "ConnectFailed"
	NoHealthyUpstream     UpstreamFailureReason = "NoHealthyUpstream"
	ResourceLimitExceeded UpstreamFailureReason = "ResourceLimitExceeded"
	NoRoute               UpstreamFailureReason = "NoRoute"
)

// UpstreamResetType
type UpstreamResetType string

// Group of Upstream Reset Type
const (
	UpstreamReset         UpstreamResetType = "UpstreamReset"
	UpstreamGlobalTimeout UpstreamResetType = "UpstreamGlobalTimeout"
	UpstreamPerTryTimeout UpstreamResetType = "UpstreamPerTryTimeout"
)

func init() {
	ConnNewPoolFactories = make(map[types.Protocol]connNewPool)
}

type connNewPool func(host types.Host) types.ConnectionPool

var ConnNewPoolFactories map[types.Protocol]connNewPool

func RegisterNewPoolFactory(protocol types.Protocol, factory connNewPool) {
	//other
	ConnNewPoolFactories[protocol] = factory
}
