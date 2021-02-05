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

package dubbothrift

import (
	"github.com/apache/thrift/lib/go/thrift"
	"mosn.io/api"
)

const (
	ProtocolName = "dubbo-thrift"
)

// thrift protocol
const (
	MessageLenSize       = 4
	MagicLen             = 2
	MessageLenIdx        = 2
	MessageHeaderLenIdx  = 6
	MessageHeaderLenSize = 2
	HeaderIdx            = 9
	HeaderLen            = 9
	IdLen                = 8
)

// req/resp type
const (
	UnKnownCmdType string = "unknown cmd type"
)

const (
	EventRequest  int = 1 // request
	EventResponse int = 2 // response
)

const (
	ServiceNameHeader     string = "service"
	MethodNameHeader      string = "method"
	SeqIdNameHeader       string = "seqId"
	MessageTypeNameHeader string = "messageType"
)

const (
	ResponseStatusSuccess = uint16(thrift.REPLY)
)

type dThriftStatusInfo struct {
	Status int32
	Msg    string
}

var (
	dubboMosnStatusMap = map[int]dThriftStatusInfo{
		api.CodecExceptionCode:    {Status: thrift.PROTOCOL_ERROR, Msg: "0|codec exception"},
		api.UnknownCode:           {Status: thrift.UNKNOWN_APPLICATION_EXCEPTION, Msg: "2|unknown"},
		api.DeserialExceptionCode: {Status: thrift.PROTOCOL_ERROR, Msg: "3|deserial exception"},
		api.SuccessCode:           {Status: 0, Msg: "200|success"},
		api.PermissionDeniedCode:  {Status: thrift.INTERNAL_ERROR, Msg: "403|permission denied"},
		api.RouterUnavailableCode: {Status: thrift.UNKNOWN_METHOD, Msg: "404|router unavailable"},
		api.InternalErrorCode:     {Status: thrift.INTERNAL_ERROR, Msg: "500|internal error"},
		api.NoHealthUpstreamCode:  {Status: thrift.UNKNOWN_APPLICATION_EXCEPTION, Msg: "502|no health upstream"},
		api.UpstreamOverFlowCode:  {Status: thrift.UNKNOWN_APPLICATION_EXCEPTION, Msg: "503|upstream overflow"},
		api.TimeoutExceptionCode:  {Status: thrift.INTERNAL_ERROR, Msg: "504|timeout"},
		api.LimitExceededCode:     {Status: thrift.UNKNOWN_APPLICATION_EXCEPTION, Msg: "509|limit exceeded"},
	}
)
