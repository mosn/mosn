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

package sofa

// Const of tracing key
const (
	// old key
	TARGET_SERVICE_KEY = "sofa_head_target_service"
	// new key
	SERVICE_KEY             = "service"
	TARGET_METHOD_KEY       = "sofa_head_method_name"
	RPC_ID_KEY              = "rpc_trace_context.sofaRpcId"
	TRACER_ID_KEY           = "rpc_trace_context.sofaTraceId"
	CALLER_IP_KEY           = "rpc_trace_context.sofaCallerIp"
	CALLER_ZONE_KEY         = "rpc_trace_context.sofaCallerZone"
	APP_NAME_KEY            = "rpc_trace_context.sofaCallerApp"
	SOFA_TRACE_BAGGAGE_DATA = "rpc_trace_context.sofaPenAttrs"
	// http key
	HTTP_RPC_ID_KEY    = "SOFA-RpcId"
	HTTP_TRACER_ID_KEY = "SOFA-TraceId"
)

const (
	//0-30 for  rpc

	TRACE_ID = iota
	SPAN_ID
	PARENT_SPAN_ID
	SERVICE_NAME
	METHOD_NAME
	PROTOCOL
	RESULT_STATUS
	REQUEST_SIZE
	RESPONSE_SIZE
	UPSTREAM_HOST_ADDRESS
	DOWNSTEAM_HOST_ADDRESS
	CALLER_APP_NAME //caller
	TARGET_APP_NAME //remote app
	SPAN_TYPE
	BAGGAGE_DATA
	REQUEST_URL
	TARGET_CELL
	TARGET_IDC
	TARGET_CITY
	ROUTE_RECORD
	CALLER_CELL
	// 30-60 for other extends

	// 60-70 for mosn common

	TRACE_END = 70
)

const (
	MOSN_PROCESS_TIME = 60 + iota
	MOSN_TLS_STATE
	TLSCipherSuite
	MOSN_PROCESS_FAIL
)
