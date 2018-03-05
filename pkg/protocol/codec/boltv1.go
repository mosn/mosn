package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"time"
)

type BoltEncoderV1 struct {
}

type BoltDecoderV1 struct {
}

func (encoder *BoltEncoderV1) Encode(value interface{}, data *bytes.Buffer) {

}

func (decoder *BoltDecoderV1) Decode(ctx interface{}, data *bytes.Buffer, out interface{}) {
	readableBytes := data.Len()
	if readableBytes >= protocol.LESS_LEN_V1-1 {
		//get type
		t, err := data.ReadByte()
		if err != nil {
			fmt.Println("decode type error :", err)
		}

		//1. request
		if t == protocol.REQUEST || t == protocol.REQUEST_ONEWAY {
			if readableBytes >= protocol.REQUEST_HEADER_LEN_V1 - 2 {
				buf := data.Next(protocol.REQUEST_HEADER_LEN_V1 - 2)

				cmdCode := binary.BigEndian.Uint16(buf[:2])
				ver2 := buf[2]
				requestId := binary.BigEndian.Uint32(buf[3:7])
				codec := buf[7]
				timeout := binary.BigEndian.Uint32(buf[8:12])
				classLen := binary.BigEndian.Uint16(buf[12:14])
				headerLen := binary.BigEndian.Uint16(buf[14:16])
				contentLen := binary.BigEndian.Uint32(buf[16:20])
				var class, header, content []byte

				//TODO because of no "mark & reset", buf.off is not recoverable
				if readableBytes >= int(classLen)+int(headerLen)+int(contentLen)+protocol.REQUEST_HEADER_LEN_V1-2 {
					buf := data.Next(int(classLen) + int(headerLen) + int(contentLen))
					if classLen > 0 {
						class = buf[0:classLen]
					}
					if headerLen > 0 {
						header = buf[classLen : classLen+headerLen]
					}
					if contentLen > 0 {
						content = buf[classLen+headerLen:]
					}
				} else { // not enough data
					fmt.Println("no enough data, and cannot reset buf's position")
					return
				}

				request := &BoltRequestCommand{
					BoltCommand: BoltCommand{
						int16(cmdCode),
						ver2,
						t,
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

				fmt.Println("bolt v1 decode request:", request)
				//out <- request

				//head code test
				//handler.NewBoltCommandHandler().HandleCommand(ctx, request)
			}
		} else {
			//2. resposne
			if readableBytes > protocol.RESPONSE_HEADER_LEN_V1 - 2 {
				buf := data.Next(protocol.RESPONSE_HEADER_LEN_V1 - 2)

				cmdCode := binary.BigEndian.Uint16(buf[:2])
				ver2 := buf[2]
				requestId := binary.BigEndian.Uint32(buf[3:7])
				codec := buf[7]
				status := binary.BigEndian.Uint16(buf[8:10])
				classLen := binary.BigEndian.Uint16(buf[10:12])
				headerLen := binary.BigEndian.Uint16(buf[12:14])
				contentLen := binary.BigEndian.Uint32(buf[14:18])
				var class, header, content []byte

				//TODO because of no "mark & reset", buf.off is not recoverable
				if readableBytes >= int(classLen)+int(headerLen)+int(contentLen)+protocol.RESPONSE_HEADER_LEN_V1-2 {
					buf := data.Next(int(classLen) + int(headerLen) + int(contentLen))
					if classLen > 0 {
						class = buf[0:classLen]
					}
					if headerLen > 0 {
						header = buf[classLen : classLen+headerLen]
					}
					if contentLen > 0 {
						content = buf[classLen+headerLen:]
					}
				} else { // not enough data
					fmt.Println("no enough data, and cannot reset buf's position")
					return
				}

				response := &BoltResponseCommand{
					BoltCommand: BoltCommand{
						int16(cmdCode),
						ver2,
						t,
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

				//out <- response
				fmt.Println("bolt v1 decode response:", response)
			}
		}
	}
}
