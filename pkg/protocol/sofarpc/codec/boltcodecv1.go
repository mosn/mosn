package codec

import (
	"encoding/binary"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"time"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

type BoltEncoderV1 struct {
}

type BoltDecoderV1 struct {
}

func (encoder *BoltEncoderV1) Encode(value interface{}, data types.IoBuffer) {

}

func (decoder *BoltDecoderV1) Decode(ctx interface{}, data types.IoBuffer, out interface{}) {
	readableBytes := data.Len()
	read := 0
	if readableBytes >= protocol.LESS_LEN_V1 {
		bytes := data.Bytes()

		//protocolCode := bytes[0]
		dataType := bytes[1]

		//1. request
		if dataType == protocol.REQUEST || dataType == protocol.REQUEST_ONEWAY {
			if readableBytes >= protocol.REQUEST_HEADER_LEN_V1 {

				cmdCode := binary.BigEndian.Uint16(bytes[2:4])
				ver2 := bytes[4]
				requestId := binary.BigEndian.Uint32(bytes[5:9])
				codec := bytes[9]
				timeout := binary.BigEndian.Uint32(bytes[10:14])
				classLen := binary.BigEndian.Uint16(bytes[14:16])
				headerLen := binary.BigEndian.Uint16(bytes[16:18])
				contentLen := binary.BigEndian.Uint32(bytes[18:22])

				read = protocol.REQUEST_HEADER_LEN_V1
				var class, header, content []byte

				//TODO because of no "mark & reset", bytes.off is not recoverable
				if readableBytes >= read + int(classLen) + int(headerLen) + int(contentLen) {
					if classLen > 0 {
						class = bytes[read: read +int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytes[read : read +int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytes[read: read +int(contentLen)]
						read += int(contentLen)
					}
					//TODO mark buffer's off
				} else { // not enough data
					log.DefaultLogger.Println("[Decoder]no enough data for fully decode")
					return
				}

				request := &BoltRequestCommand{
					BoltCommand: BoltCommand{
						int16(cmdCode),
						ver2,
						dataType,
						codec,

						int(requestId),
						int16(classLen),
						int16(headerLen),
						int(contentLen),
						class,
						header,
						content,
						nil,
					},
				}
				request.SetTimeout(int(timeout))
				request.SetArriveTime(time.Now().UnixNano() / int64(time.Millisecond))

				log.DefaultLogger.Printf("[Decoder]bolt v1 decode request:%+v\n", request)

				//pass decode result to command handler
				if list, ok := out.(*[]protocol.RpcCommand); ok {
					*list = append(*list, request)
				}
			}
		} else {
			//2. resposne
			if readableBytes > protocol.RESPONSE_HEADER_LEN_V1 {

				cmdCode := binary.BigEndian.Uint16(bytes[2:4])
				ver2 := bytes[4]
				requestId := binary.BigEndian.Uint32(bytes[5:9])
				codec := bytes[9]
				status := binary.BigEndian.Uint16(bytes[10:12])
				classLen := binary.BigEndian.Uint16(bytes[12:14])
				headerLen := binary.BigEndian.Uint16(bytes[14:16])
				contentLen := binary.BigEndian.Uint32(bytes[16:20])

				read = protocol.RESPONSE_HEADER_LEN_V1
				var class, header, content []byte

				if readableBytes >= read + int(classLen)+int(headerLen)+int(contentLen) {
					if classLen > 0 {
						class = bytes[read: read +int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytes[read : read+int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytes[read:read+ int(contentLen)]
						read += int(contentLen)
					}
				} else { // not enough data
					log.DefaultLogger.Println("[Decoder]no enough data for fully decode")
					return
				}

				response := &BoltResponseCommand{
					BoltCommand: BoltCommand{
						int16(cmdCode),
						ver2,
						dataType,
						codec,

						int(requestId),
						int16(classLen),
						int16(headerLen),
						int(contentLen),
						class,
						header,
						content,
						nil,
					},
				}
				response.SetResponseStatus(int16(status))
				response.SetResponseTimeMillis(time.Now().UnixNano() / int64(time.Millisecond))

				log.DefaultLogger.Printf("[Decoder]bolt v1 decode response:%+v\n", response)

				//pass decode result to command handler
				if list, ok := out.(*[]protocol.RpcCommand); ok {
					*list = append(*list, response)
				}
			}
		}
	}
}
