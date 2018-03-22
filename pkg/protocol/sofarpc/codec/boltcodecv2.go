package codec

import (
	"encoding/binary"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"time"
)

// types.Encoder & types.Decoder
type boltV2Codec struct {
}

func (encoder *boltV2Codec) Encode(value interface{}, data types.IoBuffer) {

	if rpcCmd, ok := value.(*boltCommand); !ok {

		log.DefaultLogger.Println("[Decode] Invalid Input Type")
		return
	} else {

		requestType := ""

		if rpcCmd.cmdCode == sofarpc.RPC_REQUEST {
			requestType = "request"

		} else {
			requestType = "response"

		}
		log.DefaultLogger.Println("prepare to encode rpcCommand,type=%s,command=%+v", requestType, rpcCmd)

		var result []byte

		result = append(result, rpcCmd.protocol) //encode protocol type: bolt v1

		//set ver1
		if requestType == "request" { //timeout  int32 ,  for request

			requestCmd := value.(*boltRequestCommand)
			result = append(result, requestCmd.GetVer1())

		} else { //respStatus  int16  , for response
			responseCmd := value.(*boltResponseCommand)
			result = append(result, responseCmd.GetVer1())

		}

		result = append(result, rpcCmd.cmdType)

		cmdCodeBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(cmdCodeBytes, uint16(rpcCmd.cmdCode))
		result = append(result, cmdCodeBytes...)

		result = append(result, rpcCmd.ver2)

		requestIdBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(requestIdBytes, uint32(rpcCmd.id))
		result = append(result, requestIdBytes...)

		result = append(result, rpcCmd.codec)

		if requestType == "request" { //timeout  int32 ,  for request

			requestCmd := value.(*boltRequestCommand)
			result = append(result, requestCmd.GetSwitch())

			timeoutBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(timeoutBytes, uint32(requestCmd.timeout))
			result = append(result, timeoutBytes...)
		} else { //respStatus  int16  , for response
			responseCmd := value.(*boltResponseCommand)
			result = append(result, responseCmd.GetSwitch())

			respStatusBytes := make([]byte, 2)
			binary.BigEndian.PutUint16(respStatusBytes, uint16(responseCmd.responseStatus))
			result = append(result, respStatusBytes...)
		}

		clazzLengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(clazzLengthBytes, uint16(rpcCmd.classLength))
		result = append(result, clazzLengthBytes...)

		headerLengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(headerLengthBytes, uint16(rpcCmd.headerLength))
		result = append(result, headerLengthBytes...)

		contentLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(contentLenBytes, uint32(rpcCmd.contentLength))
		result = append(result, contentLenBytes...)

		if rpcCmd.classLength > 0 {
			result = append(result, rpcCmd.class...)
		}

		if rpcCmd.headerLength > 0 {
			result = append(result, rpcCmd.header...)
		}

		if rpcCmd.contentLength > 0 {
			result = append(result, rpcCmd.content...)
		}

		log.DefaultLogger.Println("encode command finished,type=%s,bytes=%d", requestType, result)

		//append to data
		data.Reset()
		data.Append(result)
	}
}

func (decoder *boltV2Codec) Decode(ctx interface{}, data types.IoBuffer, out interface{}) int {
	fmt.Println("bolt v2 decode:", data.Bytes())

	readableBytes := data.Len()
	read := 0

	if readableBytes >= sofarpc.LESS_LEN_V2 {
		bytes := data.Bytes()

		protocolCode := bytes[0]
		ver1 := bytes[1]

		dataType := bytes[2] // request/response/request oneway

		//1. request
		if dataType == sofarpc.REQUEST || dataType == sofarpc.REQUEST_ONEWAY {
			if readableBytes >= sofarpc.REQUEST_HEADER_LEN_V2 {

				cmdCode := binary.BigEndian.Uint16(bytes[3:5])
				ver2 := bytes[5]
				requestId := binary.BigEndian.Uint32(bytes[6:10])
				codec := bytes[10]

				switchCode := bytes[11]
				timeout := binary.BigEndian.Uint32(bytes[12:16])
				classLen := binary.BigEndian.Uint16(bytes[16:18])
				headerLen := binary.BigEndian.Uint16(bytes[18:20])
				contentLen := binary.BigEndian.Uint32(bytes[20:24])

				read = sofarpc.REQUEST_HEADER_LEN_V2
				var class, header, content []byte

				//TODO because of no "mark & reset", bytes.off is not recoverable
				if readableBytes >= read+int(classLen)+int(headerLen)+int(contentLen) {
					if classLen > 0 {
						class = bytes[read : read+int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytes[read : read+int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytes[read : read+int(contentLen)]
						read += int(contentLen)
					}
					//TODO mark buffer's off
				} else { // not enough data
					log.DefaultLogger.Println("[Decoder]no enough data for fully decode")
					return
				}

				request := &boltRequestCommand{
					boltCommand: boltCommand{
						protocolCode,
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
				request.SetVer1(ver1)
				request.SetSwitch(switchCode)

				request.SetTimeout(int(timeout))
				request.SetArriveTime(time.Now().UnixNano() / int64(time.Millisecond))

				log.DefaultLogger.Printf("[Decoder]bolt v2 decode request:%+v\n", request)

				//pass decode result to command handler
				if list, ok := out.(*[]sofarpc.RpcCommand); ok {
					*list = append(*list, request)
				}
			}
		} else {
			//2. resposne
			if readableBytes > sofarpc.RESPONSE_HEADER_LEN_V2 {

				cmdCode := binary.BigEndian.Uint16(bytes[3:5])
				ver2 := bytes[5]
				requestId := binary.BigEndian.Uint32(bytes[6:10])
				codec := bytes[10]
				switchCode := bytes[11]

				status := binary.BigEndian.Uint16(bytes[12:14])
				classLen := binary.BigEndian.Uint16(bytes[14:16])
				headerLen := binary.BigEndian.Uint16(bytes[16:18])
				contentLen := binary.BigEndian.Uint32(bytes[18:22])

				read = sofarpc.RESPONSE_HEADER_LEN_V2
				var class, header, content []byte

				if readableBytes >= read+int(classLen)+int(headerLen)+int(contentLen) {
					if classLen > 0 {
						class = bytes[read : read+int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytes[read : read+int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytes[read : read+int(contentLen)]
						read += int(contentLen)
					}
				} else { // not enough data
					log.DefaultLogger.Println("[Decoder]no enough data for fully decode")
					return
				}

				response := &boltResponseCommand{
					boltCommand: boltCommand{
						protocolCode,
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
				response.SetVer1(ver1)
				response.SetSwitch(switchCode)
				response.SetResponseStatus(int16(status))
				response.SetResponseTimeMillis(time.Now().UnixNano() / int64(time.Millisecond))

				log.DefaultLogger.Printf("[Decoder]bolt v2 decode response:%+v\n", response)

				//pass decode result to command handler
				if list, ok := out.(*[]sofarpc.RpcCommand); ok {
					*list = append(*list, response)
				}
			}
		}
	}
	return read
}
