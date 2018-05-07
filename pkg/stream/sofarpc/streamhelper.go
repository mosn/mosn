package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"reflect"
	"strconv"
)

func (s *stream) encodeSterilize(headers interface{}) interface{} {
	if headerMaps, ok := headers.(map[string]string); ok {
		if s.direction == InStream {
			headerMaps[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = s.requestId
		}

		// remove proxy header before codec encode
		delete(headerMaps, types.HeaderStreamID)
		delete(headerMaps, types.HeaderGlobalTimeout)
		delete(headerMaps, types.HeaderTryTimeout)

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
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context ,headerMaps, sofarpc.RESPONSE_STATUS_CODEC_EXCEPTION)
				case types.DeserialExceptionCode:
					//Hessian Exception
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context ,headerMaps, sofarpc.RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION)
				case types.TimeoutExceptionCode:
					//Response Timeout
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context ,headerMaps, sofarpc.RESPONSE_STATUS_TIMEOUT)
				default:
					respHeaders, err = sofarpc.BuildSofaRespMsg(s.context, headerMaps, sofarpc.RESPONSE_STATUS_UNKNOWN)
				}

				if err == nil {
					headers = respHeaders
				} else {
					s.connection.logger.Errorf(err.Error())
				}
			}
		}

		headers = headerMaps
	}

	return headers
}

//added by @boqin: return value represents whether the request is HearBeat or not
//if request is heartbeat msg, then it only has request header, so return true as endStream
func decodeSterilize(streamId string, headers map[string]string) bool{
	headers[types.HeaderStreamID] = streamId

	if v, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderTimeout)]; ok {
		headers[types.HeaderTryTimeout] = v
	}

	if cmdCodeStr, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)]; ok {
		cmdCode := sofarpc.ConvertPropertyValue(cmdCodeStr, reflect.Int16)
		if cmdCode == sofarpc.HEARTBEAT {
			return true
		}
	}
	return false
}
