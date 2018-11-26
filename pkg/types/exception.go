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

// Header key types
const (
	HeaderStatus        = "x-mosn-status"
	HeaderMethod        = "x-mosn-method"
	HeaderHost          = "x-mosn-host"
	HeaderPath          = "x-mosn-path"
	HeaderQueryString   = "x-mosn-querystring"
	HeaderStreamID      = "x-mosn-streamid"
	HeaderGlobalTimeout = "x-mosn-global-timeout"
	HeaderTryTimeout    = "x-mosn-try-timeout"
	HeaderException     = "x-mosn-exception"
	HeaderStremEnd      = "x-mosn-endstream"
	HeaderRPCService    = "x-mosn-rpc-service"
	HeaderRPCMethod     = "x-mosn-rpc-method"
)

// Error messages
const (
	UnSupportedProCode   string = "Protocol Code not supported"
	CodecException       string = "Codec exception occurs"
	DeserializeException string = "Deserial exception occurs"
)

// Error codes
const (
	CodecExceptionCode    int = 0
	UnknownCode           int = 2
	DeserialExceptionCode int = 3
	SuccessCode           int = 200
	RouterUnavailableCode int = 404
	NoHealthUpstreamCode  int = 502
	UpstreamOverFlowCode  int = 503
	TimeoutExceptionCode  int = 504
	LimitExceededCode     int = 509
)
