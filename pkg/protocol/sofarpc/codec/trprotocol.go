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
package codec

import (
	_ "gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/handler"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func init() {
	sofarpc.RegisterProtocol(sofarpc.PROTOCOL_CODE_TR, Tr)
}

var Tr = &TrProtocol{
	sofarpc.PROTOCOL_CODE_TR,
	&trCodec{},
	&trCodec{},
	handler.NewTrCommandHandler(),
}

type TrProtocol struct {
	protocolCode byte
	encoder      types.Encoder
	decoder      types.Decoder
	//heartbeatTrigger			protocol.HeartbeatTrigger todo
	commandHandler sofarpc.CommandHandler
}

func (t *TrProtocol) GetEncoder() types.Encoder {
	return t.encoder
}

func (t *TrProtocol) GetDecoder() types.Decoder {
	return t.decoder
}

func (t *TrProtocol) GetCommandHandler() sofarpc.CommandHandler {
	return t.commandHandler
}

/*
func NewTrHeartbeat(requestId uint32) *sofarpc.TrRequestCommand{
	return &sofarpc.TrRequestCommand{
		TrCommand:sofarpc.TrCommand{
			sofarpc.PROTOCOL_CODE,
			sofarpc.HEADER_REQUEST,
			sofarpc.HESSIAN2_SERIALIZE,
			sofarpc.HEADER_TWOWAY,
			0,
			0,
			byte(len(sofarpc.TR_HEARTBEART_CLASS)),
			0,
			nil,
			sofarpc.TR_HEARTBEART_CLASS,
			nil,
		},
	}
}

func NewTrHeartbeatAck(requestId uint32) *sofarpc.TrResponseCommand{
	return &sofarpc.TrResponseCommand{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.RESPONSE,
		CmdCode:  sofarpc.HEARTBEAT,
		Version: 1,
		ReqId: requestId,
		CodecPro: sofarpc.HESSIAN2_SERIALIZE,//todo: read default codec from config
		ResponseStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
	}
}
*/
