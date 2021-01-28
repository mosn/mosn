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

// [Proxy]: the identification of a request info's content
const (
	VarStartTime                      = types.VarStartTime
	VarRequestReceivedDuration        = types.VarRequestReceivedDuration
	VarResponseReceivedDuration       = types.VarResponseReceivedDuration
	VarRequestFinishedDuration        = types.VarRequestFinishedDuration
	VarBytesSent                      = types.VarBytesSent
	VarBytesReceived                  = types.VarBytesReceived
	VarProtocol                       = types.VarProtocol
	VarResponseCode                   = types.VarResponseCode
	VarDuration                       = types.VarDuration
	VarResponseFlag                   = types.VarResponseFlag
	VarResponseFlags                  = types.VarResponseFlags
	VarUpstreamLocalAddress           = types.VarUpstreamLocalAddress
	VarDownstreamLocalAddress         = types.VarDownstreamLocalAddress
	VarDownstreamRemoteAddress        = types.VarDownstreamRemoteAddress
	VarUpstreamHost                   = types.VarUpstreamHost
	VarUpstreamTransportFailureReason = types.VarUpstreamTransportFailureReason
	VarUpstreamCluster                = types.VarUpstreamCluster
	VarRequestedServerName            = types.VarRequestedServerName
	VarRouteName                      = types.VarRouteName

	// ReqHeaderPrefix is the prefix of request header's formatter
	VarPrefixReqHeader = types.VarPrefixReqHeader
	// RespHeaderPrefix is the prefix of response header's formatter
	VarPrefixRespHeader = types.VarPrefixRespHeader
)

// [Proxy]: internal communication
const (
	VarProxyTryTimeout       = types.VarProxyTryTimeout
	VarProxyGlobalTimeout    = types.VarProxyGlobalTimeout
	VarProxyHijackStatus     = types.VarProxyHijackStatus
	VarProxyGzipSwitch       = types.VarProxyGzipSwitch
	VarProxyIsDirectResponse = types.VarProxyIsDirectResponse
	VarDirection             = types.VarDirection
	VarScheme                = types.VarScheme
	VarHost                  = types.VarHost
	VarPath                  = types.VarPath
	VarQueryString           = types.VarQueryString
	VarMethod                = types.VarMethod
	VarIstioHeaderHost       = types.VarIstioHeaderHost
	VarHeaderStatus          = types.VarHeaderStatus
	VarHeaderRPCService      = types.VarHeaderRPCService
	VarHeaderRPCMethod       = types.VarHeaderRPCMethod
)

// [Protocol]: common
const (
	VarProtocolRequestScheme    = types.VarProtocolRequestScheme
	VarProtocolRequestMethod    = types.VarProtocolRequestMethod
	VarProtocolRequestLength    = types.VarProtocolRequestLength
	VarProtocolRequestHeader    = types.VarProtocolRequestHeader
	VarProtocolCookie           = types.VarProtocolCookie
	VarProtocolRequestPath      = types.VarProtocolRequestPath
	VarProtocolRequestArgPrefix = types.VarProtocolRequestArgPrefix
	VarProtocolRequestArg       = types.VarProtocolRequestArg
	VarProtocolRequestUri       = types.VarProtocolRequestUri
)

// [Protocol]: http1
const (
	// the httpProtocolName value is protocol.HTTP1
	//httpProtocolName     = "Http1"
	VarHttpRequestScheme = types.VarHttpRequestScheme
	VarHttpRequestMethod = types.VarHttpRequestMethod
	VarHttpRequestLength = types.VarHttpRequestLength
	VarHttpRequestUri    = types.VarHttpRequestUri
	VarHttpRequestPath   = types.VarHttpRequestPath
	VarHttpRequestArg    = types.VarHttpRequestArg
	VarPrefixHttpHeader  = types.VarPrefixHttpHeader
	VarPrefixHttpArg     = types.VarPrefixHttpArg
	VarPrefixHttpCookie  = types.VarPrefixHttpCookie
)

// [Protocol]: http2
const (
	// the http2ProtocolName value is protocol.HTTP2
	//http2ProtocolName     = "Http2"
	VarHttp2RequestScheme = types.VarHttp2RequestScheme
	VarHttp2RequestMethod = types.VarHttp2RequestMethod
	VarHttp2RequestLength = types.VarHttp2RequestLength
	VarHttp2RequestUri    = types.VarHttp2RequestUri
	VarHttp2RequestPath   = types.VarHttp2RequestPath
	VarHttp2RequestArg    = types.VarHttp2RequestArg
	VarPrefixHttp2Header  = types.VarPrefixHttp2Header
	VarPrefixHttp2Arg     = types.VarPrefixHttp2Arg
	VarPrefixHttp2Cookie  = types.VarPrefixHttp2Cookie
)
