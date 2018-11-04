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

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func (s *stream) encodeSterilize(headers types.HeaderMap) types.HeaderMap {
	if cmd, ok := headers.(sofarpc.ProtoBasicCmd); ok {
		switch s.direction {
		case ClientStream:
			cmd.SetReqID(protocol.RequestIDConv(s.streamID))
		case ServerStream:
			cmd.SetReqID(protocol.RequestIDConv(s.requestID))
		}

		// hijack scene
		// TODO: distinguish proxy response / hijack replay by origin response judgement
		if status, ok := headers.Get(types.HeaderStatus); ok {
			headers.Del(types.HeaderStatus)
			statusCode, _ := strconv.Atoi(status)

			if statusCode != types.SuccessCode {
				var err error
				var resp sofarpc.ProtoBasicCmd

				//Build Router Unavailable Response Msg
				switch statusCode {
				case types.RouterUnavailableCode:
					//No available path
					resp, err = sofarpc.BuildSofaRespMsg(s.context, cmd, sofarpc.RESPONSE_STATUS_CLIENT_SEND_ERROR)
				case types.NoHealthUpstreamCode:
					resp, err = sofarpc.BuildSofaRespMsg(s.context, cmd, sofarpc.RESPONSE_STATUS_CONNECTION_CLOSED)
				case types.UpstreamOverFlowCode:
					resp, err = sofarpc.BuildSofaRespMsg(s.context, cmd, sofarpc.RESPONSE_STATUS_SERVER_THREADPOOL_BUSY)
				case types.CodecExceptionCode:
					//Decode or Encode Error
					resp, err = sofarpc.BuildSofaRespMsg(s.context, cmd, sofarpc.RESPONSE_STATUS_CODEC_EXCEPTION)
				case types.DeserialExceptionCode:
					//Hessian Exception
					resp, err = sofarpc.BuildSofaRespMsg(s.context, cmd, sofarpc.RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION)
				case types.TimeoutExceptionCode:
					//Response Timeout
					resp, err = sofarpc.BuildSofaRespMsg(s.context, cmd, sofarpc.RESPONSE_STATUS_TIMEOUT)
				default:
					resp, err = sofarpc.BuildSofaRespMsg(s.context, cmd, sofarpc.RESPONSE_STATUS_UNKNOWN)
				}

				if err == nil {
					headers = resp
				} else {
					s.connection.logger.Errorf(err.Error())
				}
			}
		}
	}
	return headers
}
