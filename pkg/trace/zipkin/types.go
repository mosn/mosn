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

package zipkin

import "strconv"

const (
	ServiceName SpanTag = iota
	MethodName
	PROTOCOL
	ResultStatus
	RequestSize
	ResponseSize
	DownsteamHostAddress
	UpstreamHostAddress
	CallerAppName //caller
	CalleeAppName //remote app
)

type SpanTag uint64

func (s SpanTag) String() string {
	switch s {
	case ServiceName:
		return "service_name"
	case MethodName:
		return "method_name"
	case PROTOCOL:
		return "protocol"
	case ResultStatus:
		return "result_status"
	case RequestSize:
		return "request_size"
	case ResponseSize:
		return "response_size"
	case DownsteamHostAddress:
		return "downstream_host_address"
	case UpstreamHostAddress:
		return "upstream_host_address"
	case CallerAppName:
		return "caller_app_name"
	case CalleeAppName:
		return "callee_app_name"
	default:
		return strconv.Itoa(int(s))
	}
}
