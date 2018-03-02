package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/codec"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

const (
	PROTOCOL_CODE byte = 13

	PROTOCOL_HEADER_LEN int = 14
)

var Tr = &TrProtocol{
	PROTOCOL_CODE,
	&codec.TrEncoder{},
	&codec.TrDecoder{},
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
	//commandHandler			protocol.CommandHandler todo
}

func (t *TrProtocol) GetEncoder() types.Encoder {
	return t.encoder
}

func (t *TrProtocol) GetDecoder() types.Decoder {
	return t.decoder
}
