package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/handler"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
)

/**
 * Request command protocol for v1
 * 0     1     2           4           6           8          10           12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |proto| type| cmdcode   |ver2 |   requestId           |codec|        timeout        |  classLen |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * |headerLen  | contentLen            |                             ... ...                       |
 * +-----------+-----------+-----------+                                                                                               +
 * |               className + header  + content  bytes                                            |
 * +                                                                                               +
 * |                               ... ...                                                         |
 * +-----------------------------------------------------------------------------------------------+
 *
 * proto: code for protocol
 * type: request/response/request oneway
 * cmdcode: code for remoting command
 * ver2:version for remoting command
 * requestId: id of request
 * codec: code for codec
 * headerLen: length of header
 * contentLen: length of content
 *
 * Response command protocol for v1
 * 0     1     2     3     4           6           8          10           12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |proto| type| cmdcode   |ver2 |   requestId           |codec|respstatus |  classLen |headerLen  |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * | contentLen            |                  ... ...                                              |
 * +-----------------------+                                                                       +
 * |                         className + header  + content  bytes                                  |
 * +                                                                                               +
 * |                               ... ...                                                         |
 * +-----------------------------------------------------------------------------------------------+
 * respstatus: response status
 */
var BoltV1 = &BoltProtocol{
	protocol.PROTOCOL_CODE_V1,
	protocol.REQUEST_HEADER_LEN_V1,
	protocol.RESPONSE_HEADER_LEN_V1,
	&codec.BoltEncoderV1{},
	&codec.BoltDecoderV1{},
	handler.NewBoltCommandHandler(),
}

/**
 * Request command protocol for v2
 * 0     1     2           4           6           8          10     11     12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+------+-----+-----+-----+-----+
 * |proto| ver1|type | cmdcode   |ver2 |   requestId           |codec|switch|   timeout             |
 * +-----------+-----------+-----------+-----------+-----------+------------+-----------+-----------+
 * |classLen   |headerLen  |contentLen             |           ...                                  |
 * +-----------+-----------+-----------+-----------+                                                +
 * |               className + header  + content  bytes                                             |
 * +                                                                                                +
 * |                               ... ...                                  | CRC32(optional)       |
 * +------------------------------------------------------------------------------------------------+
 *
 * proto: code for protocol
 * ver1: version for protocol
 * type: request/response/request oneway
 * cmdcode: code for remoting command
 * ver2:version for remoting command
 * requestId: id of request
 * codec: code for codec
 * switch: function switch for protocol
 * headerLen: length of header
 * contentLen: length of content
 * CRC32: CRC32 of the frame(Exists when ver1 > 1)
 *
 * Response command protocol for v2
 * 0     1     2     3     4           6           8          10     11    12          14          16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+------+-----+-----+-----+-----+
 * |proto| ver1| type| cmdcode   |ver2 |   requestId           |codec|switch|respstatus |  classLen |
 * +-----------+-----------+-----------+-----------+-----------+------------+-----------+-----------+
 * |headerLen  | contentLen            |                      ...                                   |
 * +-----------------------------------+                                                            +
 * |               className + header  + content  bytes                                             |
 * +                                                                                                +
 * |                               ... ...                                  | CRC32(optional)       |
 * +------------------------------------------------------------------------------------------------+
 * respstatus: response status
 */
var BoltV2 = &BoltProtocol{
	protocol.PROTOCOL_CODE_V2,
	protocol.REQUEST_HEADER_LEN_V2,
	protocol.RESPONSE_HEADER_LEN_V2,
	&codec.BoltEncoderV2{},
	&codec.BoltDecoderV2{},
	handler.NewBoltCommandHandler(),
}

type BoltProtocol struct {
	protocolCode      byte
	requestHeaderLen  int
	responseHeaderLen int

	encoder types.Encoder
	decoder types.Decoder
	//heartbeatTrigger			protocol.HeartbeatTrigger todo
	commandHandler			protocol.CommandHandler
}

func (b *BoltProtocol) GetRequestHeaderLength() int {
	return b.requestHeaderLen
}

func (b *BoltProtocol) GetResponseHeaderLength() int {
	return b.responseHeaderLen
}

func (b *BoltProtocol) GetEncoder() types.Encoder {
	return b.encoder
}

func (b *BoltProtocol) GetDecoder() types.Decoder {
	return b.decoder
}

func (b *BoltProtocol) GetCommandHandler() protocol.CommandHandler {
	return b.commandHandler
}

//TODO move this func to util
func GetLessLen(x, y int) int {
	if x < y {
		return x
	}
	return y
}
