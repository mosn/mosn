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
	"github.com/alipay/sofa-mosn/pkg/protocol"
)

func (s *stream) encodeSterilize(headers types.HeaderMap) types.HeaderMap {
	switch h := headers.(type) {
	case protocol.CommonHeader:
		if s.direction == ServerStream {
			h[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = s.requestID
		}

		// remove proxy header before codec encode
		delete(h, types.HeaderStreamID)
		delete(h, types.HeaderGlobalTimeout)
		delete(h, types.HeaderTryTimeout)

		delete(h, types.HeaderStremEnd)

		if status, ok := h[types.HeaderStatus]; ok {
			delete(h, types.HeaderStatus)
			statusCode, _ := strconv.Atoi(status)

			if statusCode != types.SuccessCode {
				var err error
				var respHeaders types.HeaderMap

				//Build Router Unavailable Response Msg
				switch statusCode {
				case types.RouterUnavailableCode, types.NoHealthUpstreamCode, types.UpstreamOverFlowCode:
					//No available path
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, h, sofarpc.RESPONSE_STATUS_CLIENT_SEND_ERROR)
				case types.CodecExceptionCode:
					//Decode or Encode Error
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, h, sofarpc.RESPONSE_STATUS_CODEC_EXCEPTION)
				case types.DeserialExceptionCode:
					//Hessian Exception
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, h, sofarpc.RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION)
				case types.TimeoutExceptionCode:
					//Response Timeout
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, h, sofarpc.RESPONSE_STATUS_TIMEOUT)
				default:
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, h, sofarpc.RESPONSE_STATUS_UNKNOWN)
				}

				if err == nil {
					headers = respHeaders
				} else {
					s.connection.logger.Errorf(err.Error())
				}
			}
		}
		//else {
		//	headers = h
		//}
	case sofarpc.ProtoBasicCmd:
		if s.direction == ServerStream {
			reqId, _ := strconv.ParseUint(s.requestID, 10, 32)
			h.SetReqID(uint32(reqId))
		}

		if status := headers.Get(types.HeaderStatus); status != "" {
			headers.Del(types.HeaderStatus)
			statusCode, _ := strconv.Atoi(status)

			if statusCode != types.SuccessCode {
				var err error
				var respHeaders types.HeaderMap

				//Build Router Unavailable Response Msg
				switch statusCode {
				case types.RouterUnavailableCode, types.NoHealthUpstreamCode, types.UpstreamOverFlowCode:
					//No available path
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, headers, sofarpc.RESPONSE_STATUS_CLIENT_SEND_ERROR)
				case types.CodecExceptionCode:
					//Decode or Encode Error
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, headers, sofarpc.RESPONSE_STATUS_CODEC_EXCEPTION)
				case types.DeserialExceptionCode:
					//Hessian Exception
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, headers, sofarpc.RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION)
				case types.TimeoutExceptionCode:
					//Response Timeout
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, headers, sofarpc.RESPONSE_STATUS_TIMEOUT)
				default:
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, headers, sofarpc.RESPONSE_STATUS_UNKNOWN)
				}

				if err == nil {
					headers = respHeaders
				} else {
					s.connection.logger.Errorf(err.Error())
				}
			}
		}
		//else {
		//	headers = headers
		//}
	}
	return headers
}

//added by @boqin: return value represents whether the request is HearBeat or not
//if request is heartbeat msg, then it only has request header, so return true as endStream
func decodeSterilize(streamID string, headers types.HeaderMap) bool {
	switch h := headers.(type) {
	case protocol.CommonHeader:

		h[types.HeaderStreamID] = streamID

		if v, ok := h[sofarpc.SofaPropertyHeader(sofarpc.HeaderTimeout)]; ok {
			h[types.HeaderTryTimeout] = v
		}

		if cmdCodeStr, ok := h[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)]; ok {
			cmdCode := sofarpc.ConvertPropertyValueInt16(cmdCodeStr)
			if cmdCode == sofarpc.HEARTBEAT {
				return true
			}
		}

		if _, ok := h[types.HeaderStremEnd]; ok {
			return true
		}
	case sofarpc.ProtoBasicCmd:
		// streamId replace ignored

		// timeout is used for server-side fast-fail,so no need to pass it to proxy-level

		// TODO need to get content length
		return h.GetCmdCode() == sofarpc.HEARTBEAT
	}

	return false
}
