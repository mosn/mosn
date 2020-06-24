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

package dubbo

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"mosn.io/mosn/pkg/types"
)

const (
	ProtocolName = "dubbo"
)

// dubbo protocol
const (
	HeaderLen   = 16
	IdLen       = 8
	MagicIdx    = 0
	FlagIdx     = 2
	StatusIdx   = 3
	IdIdx       = 4
	DataLenIdx  = 12
	DataLenSize = 4
)

// req/resp type
const (
	CmdTypeResponse      byte   = 0 // cmd type
	CmdTypeRequest       byte   = 1
	CmdTypeRequestOneway byte   = 2
	UnKnownCmdType       string = "unknown cmd type"
)

const (
	EventRequest  int = 1 // request
	EventResponse int = 0 // response
)

const (
	FrameworkVersionNameHeader string = "dubbo"
	ServiceNameHeader          string = "service"
	MethodNameHeader           string = "method"
	VersionNameHeader          string = "version"
	GroupNameHeader            string = "group"
	InterfaceNameHeader        string = "interface"
)

const (
	EgressDubbo  string = "egress_dubbo"
	IngressDubbo string = "ingress_dubbo"
)

const (
	ResponseStatusSuccess uint16 = 0x14 // 0x14 response status
)

var (
	dubboStatusMsg = map[int]string{
		types.CodecExceptionCode:    "0|codec exception",
		types.UnknownCode:           "2|unknown",
		types.DeserialExceptionCode: "3|deserial exception",
		types.SuccessCode:           "200|success",
		types.PermissionDeniedCode:  "403|permission denied",
		types.RouterUnavailableCode: "404|router unavailable",
		types.InternalErrorCode:     "500|internal error",
		types.NoHealthUpstreamCode:  "502|no health upstream",
		types.UpstreamOverFlowCode:  "503|upstream overflow",
		types.TimeoutExceptionCode:  "504|timeout",
		types.LimitExceededCode:     "509|limit exceeded",
	}

	dubboStatusMap = map[int]byte{
		types.CodecExceptionCode:    hessian.Response_SERVICE_ERROR,
		types.UnknownCode:           hessian.Response_SERVICE_ERROR,
		types.DeserialExceptionCode: hessian.Response_SERVICE_ERROR,
		types.SuccessCode:           hessian.Response_OK,
		types.PermissionDeniedCode:  hessian.Response_SERVER_ERROR,
		types.RouterUnavailableCode: hessian.Response_SERVICE_NOT_FOUND,
		types.InternalErrorCode:     hessian.Response_SERVICE_ERROR,
		types.NoHealthUpstreamCode:  hessian.Response_SERVICE_NOT_FOUND,
		types.UpstreamOverFlowCode:  hessian.Response_BAD_REQUEST,
		types.TimeoutExceptionCode:  hessian.Response_CLIENT_TIMEOUT,
		types.LimitExceededCode:     hessian.Response_BAD_REQUEST,
	}
)
