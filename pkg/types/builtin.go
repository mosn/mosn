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

import "mosn.io/pkg/variable"

// built-in variables name in mosn context
const (
	keyStreamID                    = "mosn_key_stream_id"
	keyConnection                  = "mosn_key_connection"
	keyConnectionID                = "mosn_key_connection_id"
	keyConnectionPoolIndex         = "mosn_key_connection_pool_index"
	keyListenerPort                = "mosn_key_listener_port"
	keyListenerName                = "mosn_key_listener_name"
	keyListenerType                = "mosn_key_listener_type"
	keyNetworkFilterChainFactories = "mosn_key_network_filterchain_factories"
	keyAccessLogs                  = "mosn_key_access_logs"
	keyAcceptChan                  = "mosn_key_accept_chan"
	keyAcceptBuffer                = "mosn_key_accept_buffer"
	keyConnectionFd                = "mosn_key_connection_fd"
	keyTraceSpankey                = "mosn_key_span_key"
	keyTraceId                     = "mosn_key_trace_id"
	keyProxyGeneralConfig          = "mosn_key_proxy_general_config"
	keyConnectionEventListeners    = "mosn_key_connection_event_listeners"
	keyUpstreamConnectionID        = "mosn_key_upstream_connection_id"
	keyOriRemoteAddr               = "mosn_key_ori_remote_addr"
)

// built-in variables used in mosn context
var (
	VarStreamID                    = variable.NewVariable(keyStreamID, nil, nil, variable.DefaultSetter, 0)
	VarConnection                  = variable.NewVariable(keyConnection, nil, nil, variable.DefaultSetter, 0)
	VarConnectionID                = variable.NewVariable(keyConnectionID, nil, nil, variable.DefaultSetter, 0)
	VarConnectionPoolIndex         = variable.NewVariable(keyConnectionPoolIndex, nil, nil, variable.DefaultSetter, 0)
	VarListenerPort                = variable.NewVariable(keyListenerPort, nil, nil, variable.DefaultSetter, 0)
	VarListenerName                = variable.NewVariable(keyListenerName, nil, nil, variable.DefaultSetter, 0)
	VarListenerType                = variable.NewVariable(keyListenerType, nil, nil, variable.DefaultSetter, 0)
	VarNetworkFilterChainFactories = variable.NewVariable(keyNetworkFilterChainFactories, nil, nil, variable.DefaultSetter, 0)
	VarAccessLogs                  = variable.NewVariable(keyAccessLogs, nil, nil, variable.DefaultSetter, 0)
	VarAcceptChan                  = variable.NewVariable(keyAcceptChan, nil, nil, variable.DefaultSetter, 0)
	VarAcceptBuffer                = variable.NewVariable(keyAcceptBuffer, nil, nil, variable.DefaultSetter, 0)
	VarConnectionFd                = variable.NewVariable(keyConnectionFd, nil, nil, variable.DefaultSetter, 0)
	VarTraceSpankey                = variable.NewVariable(keyTraceSpankey, nil, nil, variable.DefaultSetter, 0)
	VarTraceId                     = variable.NewVariable(keyTraceId, nil, nil, variable.DefaultSetter, 0)
	VarProxyGeneralConfig          = variable.NewVariable(keyProxyGeneralConfig, nil, nil, variable.DefaultSetter, 0)
	VarConnectionEventListeners    = variable.NewVariable(keyConnectionEventListeners, nil, nil, variable.DefaultSetter, 0)
	VarUpstreamConnectionID        = variable.NewVariable(keyUpstreamConnectionID, nil, nil, variable.DefaultSetter, 0)
	VarOriRemoteAddr               = variable.NewVariable(keyOriRemoteAddr, nil, nil, variable.DefaultSetter, 0)
	// alias from varaibles
	VarDownStreamProtocol    = variable.VarDownStreamProtocol
	VarUpstreamProtocol      = variable.VarUpstreamProtocol
	VarDownStreamReqHeaders  = variable.VarDownStreamReqHeaders
	VarDownStreamRespHeaders = variable.VarDownStreamRespHeaders
	VarTraceSpan             = variable.VarTraceSpan
)

func init() {
	builtinVariables := []variable.Variable{
		VarStreamID, VarConnection, VarConnectionID, VarConnectionPoolIndex,
		VarListenerPort, VarListenerName, VarListenerType, VarNetworkFilterChainFactories,
		VarAccessLogs, VarAcceptChan, VarAcceptBuffer, VarConnectionFd,
		VarTraceSpankey, VarTraceId, VarProxyGeneralConfig, VarConnectionEventListeners,
		VarUpstreamConnectionID, VarOriRemoteAddr,
	}
	for _, v := range builtinVariables {
		variable.Register(v)
	}
}
