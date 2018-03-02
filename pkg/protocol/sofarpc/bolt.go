package sofarpc

import "gitlab.alipay-inc.com/afe/mosn/pkg/types"
import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"bytes"
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



type bolt struct{

	PROTOCOL_CODE				byte
	REQUEST_HEADER_LEN			int32
	RESPONSE_HEADER_LEN			int32

	//for bolt v2

	PROTOCOL_VERSION_1			byte
	PROTOCOL_VERSION_2			byte




	encoder 					types.Encoder
	decoder						types.Decoder

	//heartbeatTrigger			protocol.HeartbeatTrigger todo
	//commandHandler			protocol.CommandHandler

}



func NewBoltV1()bolt{

	return bolt {
		PROTOCOL_CODE:byte(1),
		REQUEST_HEADER_LEN:int32(22),
		RESPONSE_HEADER_LEN:int32(20),
		//encoder:protocol.Protocol()   todo
		//decoder:
		//heartbeatTrigger
		//commandHandler
	}
}

func NewBoltV2()bolt{

	return bolt{

		PROTOCOL_CODE:byte(2),
		PROTOCOL_VERSION_1:byte(1),
		PROTOCOL_VERSION_2:byte(2),
		REQUEST_HEADER_LEN:int32(22+2),
		RESPONSE_HEADER_LEN:int32(20+2),
		//encoder:protocol.Protocol()   todo
		decoder:bolt.V1Decode
		//heartbeatTrigger
		//commandHandler
	}
}


func (b*bolt)getRequestHeaderLength()int32{

	return b.REQUEST_HEADER_LEN
}

func (b*bolt)getResponseHeaderLength()int32{

	return b.RESPONSE_HEADER_LEN
}

func (b*bolt)getEncoder()types.Encoder{

	return b.encoder
}

func (b*bolt)getDecoder()types.Decoder{
	return b.decoder
}


func (b*bolt)V1Decode(data bytes.Buffer){


}






