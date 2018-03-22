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

/**
 * 新版报文组成:
 *   Header(1B): 报文版本
 *   Header(1B): 请求/响应
 *   Header(1B): 报文协议(HESSIAN/JAVA)
 *   Header(1B): 单向/双向(响应报文中不使用这个字段)
 *   Header(1B): Reserved
 *   Header(4B): 通信层对象长度
 *   Header(1B): 应用层对象类名长度
 *   Header(4B): 应用层对象长度
 *   Body:       通信层对象
 *   Body:       应用层对象类名
 *   Body:       应用层对象
 */
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
