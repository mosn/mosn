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

package xprotocol

import (
	"errors"
	"net/http"
	"strconv"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

// Response related consts
const (
	Response_OK                byte = 20
	Response_CLIENT_TIMEOUT    byte = 30
	Response_SERVER_TIMEOUT    byte = 31
	Response_BAD_REQUEST       byte = 40
	Response_BAD_RESPONSE      byte = 50
	Response_SERVICE_NOT_FOUND byte = 60
	Response_SERVICE_ERROR     byte = 70
	Response_SERVER_ERROR      byte = 80
	Response_CLIENT_ERROR      byte = 90
)

func init() {
	protocol.RegisterMapping(protocol.Xprotocol, &xMapping{})
}

type xMapping struct{}

func (m *xMapping) MappingHeaderStatusCode(headers types.HeaderMap) (int, error) {

	subProtocol, ok := headers.Get(types.HeaderXprotocolSubProtocol)
	if !ok {
		return 0, errors.New("no sub protocol in headers")
	}

	if subProtocol == "dubbo" {
		status, ok := headers.Get(types.HeaderXprotocolRespStatus)
		if !ok {
			return http.StatusInternalServerError, nil
		}
		s, _ := strconv.Atoi(status)
		code := byte(s)
		// TODO: more accurate mapping
		switch code {
		case Response_OK:
			withException, ok := headers.Get(types.HeaderXprotocolRespIsException)
			if ok && withException == "true" {
				return http.StatusInternalServerError, nil
			}
			return http.StatusOK, nil
		case Response_SERVER_TIMEOUT:
			return http.StatusGatewayTimeout, nil
		default:
			return http.StatusInternalServerError, nil
		}
	}

	return 0, nil

}

//
////TODO use protocol.Mapping interface
//func MappingFromHttpStatus(code int) int16 {
//	switch code {
//	case http.StatusOK:
//		return RESPONSE_STATUS_SUCCESS
//	case types.RouterUnavailableCode:
//		return RESPONSE_STATUS_NO_PROCESSOR
//	case types.NoHealthUpstreamCode:
//		return RESPONSE_STATUS_CONNECTION_CLOSED
//	case types.UpstreamOverFlowCode:
//		return RESPONSE_STATUS_SERVER_THREADPOOL_BUSY
//	case types.CodecExceptionCode:
//		//Decode or Encode Error
//		return RESPONSE_STATUS_CODEC_EXCEPTION
//	case types.DeserialExceptionCode:
//		//Hessian Exception
//		return RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION
//	case types.TimeoutExceptionCode:
//		//Response Timeout
//		return RESPONSE_STATUS_TIMEOUT
//	default:
//		return RESPONSE_STATUS_UNKNOWN
//	}
//}
