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

import (
	"context"
	"errors"

	"mosn.io/api"
	"mosn.io/pkg/variable"
)

// [Proxy]: the identification of a request info's content
const (
	VarStartTime                      string = "start_time"
	VarRequestReceivedDuration        string = "request_received_duration"
	VarResponseReceivedDuration       string = "response_received_duration"
	VarRequestFinishedDuration        string = "request_finished_duration"
	VarProcessTimeDuration            string = "process_time_duration"
	VarBytesSent                      string = "bytes_sent"
	VarBytesReceived                  string = "bytes_received"
	VarProtocol                       string = "protocol"
	VarResponseCode                   string = "response_code"
	VarDuration                       string = "duration"
	VarResponseFlag                   string = "response_flag"
	VarResponseFlags                  string = "response_flags"
	VarUpstreamLocalAddress           string = "upstream_local_address"
	VarDownstreamLocalAddress         string = "downstream_local_address"
	VarDownstreamRemoteAddress        string = "downstream_remote_address"
	VarUpstreamHost                   string = "upstream_host"
	VarUpstreamTransportFailureReason string = "upstream_transport_failure_reason"
	VarUpstreamCluster                string = "upstream_cluster"
	VarRequestedServerName            string = "requested_server_name"
	VarRouteName                      string = "route_name"
	VarProtocolConfig                 string = "protocol_config"

	// ReqHeaderPrefix is the prefix of request header's formatter
	VarPrefixReqHeader string = "request_header_"
	// RespHeaderPrefix is the prefix of response header's formatter
	VarPrefixRespHeader string = "response_header_"
)

// [Proxy]: internal communication
const (
	VarProxyTryTimeout       string = "proxy_try_timeout"
	VarProxyGlobalTimeout    string = "proxy_global_timeout"
	VarProxyHijackStatus     string = "proxy_hijack_status"
	VarProxyGzipSwitch       string = "proxy_gzip_switch"
	VarProxyIsDirectResponse string = "proxy_direct_response"
	VarProxyDisableRetry     string = "proxy_disable_retry"
	VarDirection             string = "x-mosn-direction"
	VarScheme                string = "x-mosn-scheme"
	VarHost                  string = "x-mosn-host"
	VarPath                  string = "x-mosn-path"
	VarPathOriginal          string = "x-mosn-path-original"
	VarQueryString           string = "x-mosn-querystring"
	VarMethod                string = "x-mosn-method"
	VarIstioHeaderHost       string = "authority"
	VarHeaderStatus          string = "x-mosn-status"
	VarHeaderRPCService      string = "x-mosn-rpc-service"
	VarHeaderRPCMethod       string = "x-mosn-rpc-method"

	// notice: read-only!!! do not modify the raw data!!!
	VarRequestRawData string = "x-mosn-req-raw-data"
	// notice: read-only!!! do not modify the raw data!!!
	VarResponseRawData string = "x-mosn-resp-raw-data"
)

// [server]: common
const (
	VarListenerMatchFallbackIP string = "listener_match_fallback_ip"
)

// [Route]: internal
const (
	VarRouterMeta string = "x-mosn-router-meta"
)

// [Protocol]: common
const (
	VarProtocolRequestScheme       = "request_scheme"
	VarProtocolRequestMethod       = "request_method"
	VarProtocolRequestLength       = "request_length"
	VarProtocolRequestHeader       = "request_header_"
	VarProtocolCookie              = "cookie_"
	VarProtocolRequestPath         = "request_path"
	VarProtocolRequestPathOriginal = "request_path_original"
	VarProtocolRequestArgPrefix    = "request_arg_"
	VarProtocolRequestArg          = "request_arg"
	VarProtocolRequestUri          = "request_uri"
	VarProtocolRequestUseStream    = "request_use_stream"
	VarProtocolResponseUseStream   = "response_use_stream"
)

// [Protocol]: http1
const (
	// the httpProtocolName value is protocol.HTTP1
	httpProtocolName           = "Http1"
	VarHttpRequestScheme       = httpProtocolName + "_" + VarProtocolRequestScheme
	VarHttpRequestMethod       = httpProtocolName + "_" + VarProtocolRequestMethod
	VarHttpRequestLength       = httpProtocolName + "_" + VarProtocolRequestLength
	VarHttpRequestUri          = httpProtocolName + "_" + VarProtocolRequestUri
	VarHttpRequestPath         = httpProtocolName + "_" + VarProtocolRequestPath
	VarHttpRequestPathOriginal = httpProtocolName + "_" + VarProtocolRequestPathOriginal
	VarHttpRequestArg          = httpProtocolName + "_" + VarProtocolRequestArg
	VarPrefixHttpHeader        = httpProtocolName + "_" + VarProtocolRequestHeader
	VarPrefixHttpArg           = httpProtocolName + "_" + VarProtocolRequestArgPrefix
	VarPrefixHttpCookie        = httpProtocolName + "_" + VarProtocolCookie
)

// [Protocol]: http2
const (
	// the http2ProtocolName value is protocol.HTTP2
	http2ProtocolName           = "Http2"
	VarHttp2RequestScheme       = http2ProtocolName + "_" + VarProtocolRequestScheme
	VarHttp2RequestMethod       = http2ProtocolName + "_" + VarProtocolRequestMethod
	VarHttp2RequestLength       = http2ProtocolName + "_" + VarProtocolRequestLength
	VarHttp2RequestUri          = http2ProtocolName + "_" + VarProtocolRequestUri
	VarHttp2RequestPath         = http2ProtocolName + "_" + VarProtocolRequestPath
	VarHttp2RequestPathOriginal = http2ProtocolName + "_" + VarProtocolRequestPathOriginal
	VarHttp2RequestArg          = http2ProtocolName + "_" + VarProtocolRequestArg
	VarHttp2RequestUseStream    = http2ProtocolName + "_" + VarProtocolRequestUseStream
	VarHttp2ResponseUseStream   = http2ProtocolName + "_" + VarProtocolResponseUseStream
	VarPrefixHttp2Header        = http2ProtocolName + "_" + VarProtocolRequestHeader
	VarPrefixHttp2Arg           = http2ProtocolName + "_" + VarProtocolRequestArgPrefix
	VarPrefixHttp2Cookie        = http2ProtocolName + "_" + VarProtocolCookie
)

// [MOSN]: mosn built-invariables name
const (
	VarStreamID                    = "stream_id"
	VarConnection                  = "connection"
	VarConnectionID                = "connection_id"
	VarConnectionPoolIndex         = "connection_pool_index"
	VarListenerPort                = "listener_port"
	VarListenerName                = "listener_name"
	VarListenerType                = "listener_type"
	VarConnDefaultReadBufferSize   = "conn_default_read_buffer_size"
	VarNetworkFilterChainFactories = "network_filterchain_factories"
	VarAccessLogs                  = "access_logs"
	VarAcceptChan                  = "accept_chan"
	VarAcceptBuffer                = "accept_buffer"
	VarConnectionFd                = "connection_fd"
	VarTraceSpanKey                = "span_key"
	VarTraceId                     = "trace_id"
	VarProxyGeneralConfig          = "proxy_general_config"
	VarConnectionEventListeners    = "connection_event_listeners"
	VarUpstreamConnectionID        = "upstream_connection_id"
	VarOriRemoteAddr               = "ori_remote_addr"
	VarDownStreamProtocol          = "downstream_protocol"
	VarUpStreamProtocol            = "upstream_protocol"
	VarDownStreamReqHeaders        = "downstream_req_headers"
	VarDownStreamRespHeaders       = "downstream_resp_headers"
	VarTraceSpan                   = "trace_span"
)

var (
	VariableStreamID                    = variable.NewVariable(VarStreamID, nil, nil, variable.DefaultSetter, 0)
	VariableConnection                  = variable.NewVariable(VarConnection, nil, nil, variable.DefaultSetter, 0)
	VariableConnectionID                = variable.NewVariable(VarConnectionID, nil, nil, variable.DefaultSetter, 0)
	VariableConnectionPoolIndex         = variable.NewVariable(VarConnectionPoolIndex, nil, nil, variable.DefaultSetter, 0)
	VariableListenerPort                = variable.NewVariable(VarListenerPort, nil, nil, variable.DefaultSetter, 0)
	VariableListenerName                = variable.NewVariable(VarListenerName, nil, nil, variable.DefaultSetter, 0)
	VariableListenerType                = variable.NewVariable(VarListenerType, nil, nil, variable.DefaultSetter, 0)
	VariableConnDefaultReadBufferSize   = variable.NewVariable(VarConnDefaultReadBufferSize, nil, nil, variable.DefaultSetter, 0)
	VariableNetworkFilterChainFactories = variable.NewVariable(VarNetworkFilterChainFactories, nil, nil, variable.DefaultSetter, 0)
	VariableAccessLogs                  = variable.NewVariable(VarAccessLogs, nil, nil, variable.DefaultSetter, 0)
	VariableAcceptChan                  = variable.NewVariable(VarAcceptChan, nil, nil, variable.DefaultSetter, 0)
	VariableAcceptBuffer                = variable.NewVariable(VarAcceptBuffer, nil, nil, variable.DefaultSetter, 0)
	VariableConnectionFd                = variable.NewVariable(VarConnectionFd, nil, nil, variable.DefaultSetter, 0)
	VariableTraceId                     = variable.NewVariable(VarTraceId, nil, nil, variable.DefaultSetter, 0)
	VariableProxyGeneralConfig          = variable.NewVariable(VarProxyGeneralConfig, nil, nil, variable.DefaultSetter, 0)
	VariableConnectionEventListeners    = variable.NewVariable(VarConnectionEventListeners, nil, nil, variable.DefaultSetter, 0)
	VariableUpstreamConnectionID        = variable.NewVariable(VarUpstreamConnectionID, nil, nil, variable.DefaultSetter, 0)
	VariableOriRemoteAddr               = variable.NewVariable(VarOriRemoteAddr, nil, nil, variable.DefaultSetter, 0)
	VariableTraceSpankey                = variable.NewVariable(VarTraceSpanKey, nil, nil, variable.DefaultSetter, 0)
	VariableDownStreamProtocol          = variable.NewVariable(VarDownStreamProtocol, nil, nil, variable.DefaultSetter, 0)
	VariableUpstreamProtocol            = variable.NewVariable(VarUpStreamProtocol, nil, nil, variable.DefaultSetter, 0)
	VariableDownStreamReqHeaders        = variable.NewVariable(VarDownStreamReqHeaders, nil, nil, variable.DefaultSetter, 0)
	VariableDownStreamRespHeaders       = variable.NewVariable(VarDownStreamRespHeaders, nil, nil, variable.DefaultSetter, 0)
	VariableTraceSpan                   = variable.NewVariable(VarTraceSpan, nil, nil, variable.DefaultSetter, 0)
)

func init() {
	builtinVariables := []variable.Variable{
		VariableStreamID, VariableConnection, VariableConnectionID, VariableConnectionPoolIndex,
		VariableListenerPort, VariableListenerName, VariableListenerType, VariableConnDefaultReadBufferSize, VariableNetworkFilterChainFactories,
		VariableAccessLogs, VariableAcceptChan, VariableAcceptBuffer, VariableConnectionFd,
		VariableTraceSpankey, VariableTraceId, VariableProxyGeneralConfig, VariableConnectionEventListeners,
		VariableUpstreamConnectionID, VariableOriRemoteAddr,
		VariableDownStreamProtocol, VariableUpstreamProtocol, VariableDownStreamReqHeaders, VariableDownStreamRespHeaders, VariableTraceSpan,
	}
	for _, v := range builtinVariables {
		variable.Register(v)
	}
	// register protocol resource
	variable.GetProtocol = func(ctx context.Context) (api.ProtocolName, error) {
		v, err := variable.Get(ctx, VariableDownStreamProtocol)
		if err != nil {
			return api.ProtocolName("-"), err
		}
		if proto, ok := v.(api.ProtocolName); ok {
			return proto, nil
		}
		return api.ProtocolName("-"), errors.New("invalid protocol name")
	}
}
