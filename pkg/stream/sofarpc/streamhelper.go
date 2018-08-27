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

package sofarpc

import (
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func (s *stream) encodeSterilize(headers interface{}) interface{} {
	if headerMaps, ok := headers.(map[string]string); ok {
		if s.direction == ServerStream {
			headerMaps[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = s.requestID
		}

		// remove proxy header before codec encode
		delete(headerMaps, types.HeaderStreamID)
		delete(headerMaps, types.HeaderGlobalTimeout)
		delete(headerMaps, types.HeaderTryTimeout)

		delete(headerMaps, types.HeaderStremEnd)

		if status, ok := headerMaps[types.HeaderStatus]; ok {
			delete(headerMaps, types.HeaderStatus)
			statusCode, _ := strconv.Atoi(status)

			if statusCode != types.SuccessCode {
				var err error
				var respHeaders interface{}

				//Build Router Unavailable Response Msg
				switch statusCode {
				case types.RouterUnavailableCode, types.NoHealthUpstreamCode, types.UpstreamOverFlowCode:
					//No available path
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, headerMaps, sofarpc.RESPONSE_STATUS_CLIENT_SEND_ERROR)
				case types.CodecExceptionCode:
					//Decode or Encode Error
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, headerMaps, sofarpc.RESPONSE_STATUS_CODEC_EXCEPTION)
				case types.DeserialExceptionCode:
					//Hessian Exception
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, headerMaps, sofarpc.RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION)
				case types.TimeoutExceptionCode:
					//Response Timeout
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, headerMaps, sofarpc.RESPONSE_STATUS_TIMEOUT)
				default:
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, headerMaps, sofarpc.RESPONSE_STATUS_UNKNOWN)
				}

				if err == nil {
					headers = respHeaders
				} else {
					s.connection.logger.Errorf(err.Error())
				}
			}
		} else {
			headers = headerMaps
		}
	}

	return headers
}

//added by @boqin: return value represents whether the request is HearBeat or not
//if request is heartbeat msg, then it only has request header, so return true as endStream
func decodeSterilize(streamID string, headers map[string]string) bool {
	headers[types.HeaderStreamID] = streamID

	if v, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderTimeout)]; ok {
		headers[types.HeaderTryTimeout] = v
	}

	if cmdCodeStr, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)]; ok {
		cmdCode := sofarpc.ConvertPropertyValueInt16(cmdCodeStr)
		if cmdCode == sofarpc.HEARTBEAT {
			return true
		}
	}

	if _, ok := headers[types.HeaderStremEnd]; ok {
		return true
	}

	return false
}
