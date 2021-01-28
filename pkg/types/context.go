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

package types

import "mosn.io/api/types"

// ContextKey type
type ContextKey = types.ContextKey

// Context key types(built-in)
const (
	ContextKeyStreamID                    = types.ContextKeyStreamID
	ContextKeyConnection                  = types.ContextKeyConnection
	ContextKeyConnectionID                = types.ContextKeyConnectionID
	ContextKeyConnectionPoolIndex         = types.ContextKeyConnectionPoolIndex
	ContextKeyListenerPort                = types.ContextKeyListenerPort
	ContextKeyListenerName                = types.ContextKeyListenerName
	ContextKeyListenerType                = types.ContextKeyListenerType
	ContextKeyListenerStatsNameSpace      = types.ContextKeyListenerStatsNameSpace
	ContextKeyNetworkFilterChainFactories = types.ContextKeyNetworkFilterChainFactories
	ContextKeyBufferPoolCtx               = types.ContextKeyBufferPoolCtx
	ContextKeyAccessLogs                  = types.ContextKeyAccessLogs
	ContextOriRemoteAddr                  = types.ContextOriRemoteAddr
	ContextKeyAcceptChan                  = types.ContextKeyAcceptChan
	ContextKeyAcceptBuffer                = types.ContextKeyAcceptBuffer
	ContextKeyConnectionFd                = types.ContextKeyConnectionFd
	ContextSubProtocol                    = types.ContextSubProtocol
	ContextKeyTraceSpanKey                = types.ContextKeyTraceSpanKey
	ContextKeyActiveSpan                  = types.ContextKeyActiveSpan
	ContextKeyTraceId                     = types.ContextKeyTraceId
	ContextKeyVariables                   = types.ContextKeyVariables
	ContextKeyProxyGeneralConfig          = types.ContextKeyProxyGeneralConfig
	ContextKeyDownStreamProtocol          = types.ContextKeyDownStreamProtocol
	ContextKeyConfigDownStreamProtocol    = types.ContextKeyConfigDownStreamProtocol
	ContextKeyConfigUpStreamProtocol      = types.ContextKeyConfigUpStreamProtocol
	ContextKeyDownStreamHeaders           = types.ContextKeyDownStreamHeaders
	ContextKeyDownStreamRespHeaders       = types.ContextKeyDownStreamRespHeaders
	ContextKeyEnd                         = types.ContextKeyEnd
)

// GlobalProxyName represents proxy name for metrics
const (
	GlobalProxyName       = types.GlobalProxyName
	GlobalShutdownTimeout = types.GlobalShutdownTimeout
)
