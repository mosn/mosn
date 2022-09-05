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
	VariableStreamID                    = variable.NewVariable(keyStreamID, nil, nil, variable.DefaultSetter, 0)
	VariableConnection                  = variable.NewVariable(keyConnection, nil, nil, variable.DefaultSetter, 0)
	VariableConnectionID                = variable.NewVariable(keyConnectionID, nil, nil, variable.DefaultSetter, 0)
	VariableConnectionPoolIndex         = variable.NewVariable(keyConnectionPoolIndex, nil, nil, variable.DefaultSetter, 0)
	VariableListenerPort                = variable.NewVariable(keyListenerPort, nil, nil, variable.DefaultSetter, 0)
	VariableListenerName                = variable.NewVariable(keyListenerName, nil, nil, variable.DefaultSetter, 0)
	VariableListenerType                = variable.NewVariable(keyListenerType, nil, nil, variable.DefaultSetter, 0)
	VariableNetworkFilterChainFactories = variable.NewVariable(keyNetworkFilterChainFactories, nil, nil, variable.DefaultSetter, 0)
	VariableAccessLogs                  = variable.NewVariable(keyAccessLogs, nil, nil, variable.DefaultSetter, 0)
	VariableAcceptChan                  = variable.NewVariable(keyAcceptChan, nil, nil, variable.DefaultSetter, 0)
	VariableAcceptBuffer                = variable.NewVariable(keyAcceptBuffer, nil, nil, variable.DefaultSetter, 0)
	VariableConnectionFd                = variable.NewVariable(keyConnectionFd, nil, nil, variable.DefaultSetter, 0)
	VariableTraceSpankey                = variable.NewVariable(keyTraceSpankey, nil, nil, variable.DefaultSetter, 0)
	VariableTraceId                     = variable.NewVariable(keyTraceId, nil, nil, variable.DefaultSetter, 0)
	VariableProxyGeneralConfig          = variable.NewVariable(keyProxyGeneralConfig, nil, nil, variable.DefaultSetter, 0)
	VariableConnectionEventListeners    = variable.NewVariable(keyConnectionEventListeners, nil, nil, variable.DefaultSetter, 0)
	VariableUpstreamConnectionID        = variable.NewVariable(keyUpstreamConnectionID, nil, nil, variable.DefaultSetter, 0)
	VariableOriRemoteAddr               = variable.NewVariable(keyOriRemoteAddr, nil, nil, variable.DefaultSetter, 0)
	// alias from varaibles
	VariableDownStreamProtocol    = variable.VariableDownStreamProtocol
	VariableUpstreamProtocol      = variable.VariableUpstreamProtocol
	VariableDownStreamReqHeaders  = variable.VariableDownStreamReqHeaders
	VariableDownStreamRespHeaders = variable.VariableDownStreamRespHeaders
	VariableTraceSpan             = variable.VariableTraceSpan
)

func init() {
	builtinVariables := []variable.Variable{
		VariableStreamID, VariableConnection, VariableConnectionID, VariableConnectionPoolIndex,
		VariableListenerPort, VariableListenerName, VariableListenerType, VariableNetworkFilterChainFactories,
		VariableAccessLogs, VariableAcceptChan, VariableAcceptBuffer, VariableConnectionFd,
		VariableTraceSpankey, VariableTraceId, VariableProxyGeneralConfig, VariableConnectionEventListeners,
		VariableUpstreamConnectionID, VariableOriRemoteAddr,
	}
	for _, v := range builtinVariables {
		variable.Register(v)
	}
}
