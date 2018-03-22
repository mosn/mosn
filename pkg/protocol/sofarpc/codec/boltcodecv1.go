package codec

import (
	"encoding/binary"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"time"
)

// types.Encoder & types.Decoder
type boltV1Codec struct{}

func (encoder *boltV1Codec) Encode(value interface{}, data types.IoBuffer) {
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

		result = append(result, rpcCmd.cmdType)

		cmdCodeBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(cmdCodeBytes, uint16(rpcCmd.cmdCode))
		result = append(result, cmdCodeBytes...)

		result = append(result, rpcCmd.version)

		requestIdBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(requestIdBytes, uint32(rpcCmd.id))
		result = append(result, requestIdBytes...)

		result = append(result, rpcCmd.codec)

		if requestType == "request" { //timeout  int32 ,  for request

			requestCmd := value.(*boltRequestCommand)
			timeoutBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(timeoutBytes, uint32(requestCmd.timeout))
			result = append(result, timeoutBytes...)
		} else { //respStatus  int16  , for response
			responseCmd := value.(*boltResponseCommand)
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

func (decoder *boltV1Codec) Decode(ctx interface{}, data types.IoBuffer, out interface{}) int {
	readableBytes := data.Len()
	read := 0

	if readableBytes >= sofarpc.LESS_LEN_V1 {
		bytes := data.Bytes()

		//protocolCode := bytes[0]
		dataType := bytes[1]

		//1. request
		if dataType == sofarpc.REQUEST || dataType == sofarpc.REQUEST_ONEWAY {
			if readableBytes >= sofarpc.REQUEST_HEADER_LEN_V1 {

				cmdCode := binary.BigEndian.Uint16(bytes[2:4])
				ver2 := bytes[4]
				requestId := binary.BigEndian.Uint32(bytes[5:9])
				codec := bytes[9]
				timeout := binary.BigEndian.Uint32(bytes[10:14])
				classLen := binary.BigEndian.Uint16(bytes[14:16])
				headerLen := binary.BigEndian.Uint16(bytes[16:18])
				contentLen := binary.BigEndian.Uint32(bytes[18:22])

				read = sofarpc.REQUEST_HEADER_LEN_V1
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
					return 0
				}

				request := &boltRequestCommand{
					boltCommand: boltCommand{
						sofarpc.PROTOCOL_CODE_V1,
						int16(cmdCode),
						ver2,
						dataType,
						codec,

						requestId,
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
				if list, ok := out.(*[]sofarpc.RpcCommand); ok {
					*list = append(*list, request)
				}
			}
		} else {
			//2. resposne
			if readableBytes > sofarpc.RESPONSE_HEADER_LEN_V1 {

				cmdCode := binary.BigEndian.Uint16(bytes[2:4])
				ver2 := bytes[4]
				requestId := binary.BigEndian.Uint32(bytes[5:9])
				codec := bytes[9]
				status := binary.BigEndian.Uint16(bytes[10:12])
				classLen := binary.BigEndian.Uint16(bytes[12:14])
				headerLen := binary.BigEndian.Uint16(bytes[14:16])
				contentLen := binary.BigEndian.Uint32(bytes[16:20])

				read = sofarpc.RESPONSE_HEADER_LEN_V1
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
					return 0
				}

				response := &boltResponseCommand{
					boltCommand: boltCommand{
						sofarpc.PROTOCOL_CODE_V1,
						int16(cmdCode),
						ver2,
						dataType,
						codec,
						requestId,
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
				if list, ok := out.(*[]sofarpc.RpcCommand); ok {
					*list = append(*list, response)
				}
			}
		}
	}

	return read
}

func (encoder *boltV1Codec) AddDecodeFilter(filter types.DecodeFilter) {

}
