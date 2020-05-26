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

// [Proxy]: the identification of a request info's content
const (
	VarStartTime                string = "start_time"
	VarRequestReceivedDuration  string = "request_received_duration"
	VarResponseReceivedDuration string = "response_received_duration"
	VarRequestFinishedDuration  string = "request_finished_duration"
	VarBytesSent                string = "bytes_sent"
	VarBytesReceived            string = "bytes_received"
	VarProtocol                 string = "protocol"
	VarResponseCode             string = "response_code"
	VarDuration                 string = "duration"
	VarResponseFlag             string = "response_flag"
	VarUpstreamLocalAddress     string = "upstream_local_address"
	VarDownstreamLocalAddress   string = "downstream_local_address"
	VarDownstreamRemoteAddress  string = "downstream_remote_address"
	VarUpstreamHost             string = "upstream_host"

	// ReqHeaderPrefix is the prefix of request header's formatter
	VarPrefixReqHeader string = "request_header_"
	// RespHeaderPrefix is the prefix of response header's formatter
	VarPrefixRespHeader string = "response_header_"
)

// [Proxy]: internal communication
const (
	VarProxyTryTimeout    string = "proxy_try_timeout"
	VarProxyGlobalTimeout string = "proxy_global_timeout"
	VarProxyHijackStatus  string = "proxy_hijack_status"
)

// [Protocol]: http
const (
	VarHttpRequestMethod = "http_request_method"
	VarHttpRequestLength = "http_request_length"
	VarHttpRequestUri    = "http_request_uri"
	VarHttpRequestPath   = "http_request_path"
	VarHttpRequestArg    = "http_request_arg"

	VarPrefixHttpHeader = "http_header_"
	VarPrefixHttpArg    = "http_arg_"
	VarPrefixHttpCookie = "http_cookie_"
)
