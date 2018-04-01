package codec

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/handler"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
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
