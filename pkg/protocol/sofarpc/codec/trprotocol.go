package codec

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/handler"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
)

func init() {
	sofarpc.RegisterProtocol(sofarpc.PROTOCOL_CODE, Tr)
}

var Tr = &TrProtocol{
	sofarpc.PROTOCOL_CODE,
	&trCodec{},
	&trCodec{},
	handler.NewTrCommandHandler(),
}

type TrProtocol struct {
	protocolCode byte
	encoder types.Encoder
	decoder types.Decoder
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