package codec

import (
	"time"
	"reflect"
	"encoding/binary"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

var (
	BoltV1PropertyHeaderNames = make(map[string]reflect.Kind, 11)
)

func init() {
	BoltV1PropertyHeaderNames["protocol"] = reflect.Uint8
	BoltV1PropertyHeaderNames["cmdType"] = reflect.Uint8
	BoltV1PropertyHeaderNames["cmdCode"] = reflect.Int16
	BoltV1PropertyHeaderNames["version"] = reflect.Uint8
	BoltV1PropertyHeaderNames["requestId"] = reflect.Uint32
	BoltV1PropertyHeaderNames["codec"] = reflect.Uint8
	BoltV1PropertyHeaderNames["classLength"] = reflect.Int16
	BoltV1PropertyHeaderNames["headerLength"] = reflect.Int16
	BoltV1PropertyHeaderNames["contentLength"] = reflect.Int
	BoltV1PropertyHeaderNames["timeout"] = reflect.Int
	BoltV1PropertyHeaderNames["responseStatus"] = reflect.Int16
	BoltV1PropertyHeaderNames["responseTimeMills"] = reflect.Int64
}

// types.Encoder & types.Decoder
type boltV1Codec struct{}

func (c *boltV1Codec) Encode(value interface{}, data types.IoBuffer) uint32 {
	var cmd interface{}
	if valueMap, ok := value.(map[string]string); ok {
		cmd = c.mapToCmd(valueMap)
	}

	var reqId uint32
	var result []byte

	//REQUEST
	if rpcCmd, ok := cmd.(*boltRequestCommand); ok {
		log.DefaultLogger.Println("prepare to encode rpcCommand,=%+v", rpcCmd.cmdType)

		//COMMON
		result = append(result, rpcCmd.protocol, rpcCmd.cmdType)

		cmdCodeBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(cmdCodeBytes, uint16(rpcCmd.cmdCode))
		result = append(result, cmdCodeBytes...)
		result = append(result, rpcCmd.version)

		requestIdBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(requestIdBytes, uint32(rpcCmd.id))
		result = append(result, requestIdBytes...)
		result = append(result, rpcCmd.codec)

		//FOR REQUEST
		timeoutBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(timeoutBytes, uint32(rpcCmd.timeout))
		result = append(result, timeoutBytes...)

		//COMMON
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

		//CONTENT IS REGARDED AS BODY AND TRANSMIT DIRECTLY

		reqId = rpcCmd.id
		//RESPONSE
	} else if rpcCmd, ok := value.(*boltResponseCommand); ok {
		result = append(result, rpcCmd.protocol, rpcCmd.cmdType)

		cmdCodeBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(cmdCodeBytes, uint16(rpcCmd.cmdCode))
		result = append(result, cmdCodeBytes...)
		result = append(result, rpcCmd.version)

		requestIdBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(requestIdBytes, uint32(rpcCmd.id))
		result = append(result, requestIdBytes...)
		result = append(result, rpcCmd.codec)

		//FOR RESPONSE
		respStatusBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(respStatusBytes, uint16(rpcCmd.responseStatus))
		result = append(result, respStatusBytes...)

		//COMMON
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

		reqId = rpcCmd.id
	} else {
		log.DefaultLogger.Println("[Decode] Invalid Input Type")
		return 0
	}

	log.DefaultLogger.Println("encode command finished,bytes=%d", result)

	//GET BINARY BYTE FLOW
	data.Reset()
	data.Append(result)

	return reqId
}

func (c *boltV1Codec) mapToCmd(headers map[string]string) interface{} {
	protocolCode := c.getPropertyValue(headers, "protocol")
	cmdType := c.getPropertyValue(headers, "cmdType")
	cmdCode := c.getPropertyValue(headers, "cmdCode")
	version := c.getPropertyValue(headers, "version")
	requestID := c.getPropertyValue(headers, "requestId")
	codec := c.getPropertyValue(headers, "codec")
	classLength := c.getPropertyValue(headers, "classLength")
	headerLength := c.getPropertyValue(headers, "headerLength")
	contentLength := c.getPropertyValue(headers, "contentLength")

	//class
	className := c.getPropertyValue(headers, "className")
	class, _ := serialize.Instance.Serialize(className)

	//RPC Request
	if cmdCode == sofarpc.RPC_REQUEST {
		timeout := c.getPropertyValue(headers, "timeout")

		//serialize header
		header, _ := serialize.Instance.Serialize(headers)

		request := &boltRequestCommand{
			boltCommand: boltCommand{
				protocolCode.(byte),
				cmdType.(byte),
				cmdCode.(int16),
				version.(byte),
				requestID.(uint32),
				codec.(byte),
				classLength.(int16),
				headerLength.(int16),
				contentLength.(int),
				class,
				header,
				nil,
				nil,
			},
			timeout: timeout.(int),
		}

		return request
	} else if cmdCode == sofarpc.RPC_RESPONSE {
		//todo : review
		responseStatus := c.getPropertyValue(headers, "responseStatus")
		responseTime := c.getPropertyValue(headers, "responseTimeMills")

		//serialize header
		header, _ := serialize.Instance.Serialize(headers)

		response := &boltResponseCommand{
			boltCommand: boltCommand{
				protocolCode.(byte),
				cmdType.(byte),
				cmdCode.(int16),
				version.(byte),
				requestID.(uint32),
				codec.(byte),
				classLength.(int16),
				headerLength.(int16),
				contentLength.(int),
				class,
				header,
				nil,
				nil,
			},
			responseStatus:     responseStatus.(int16),
			responseTimeMillis: responseTime.(int64),
		}

		return response
	} else {
		// todo RPC_HB
	}

	return nil
}

func (c *boltV1Codec) getPropertyValue(headers map[string]string, name string) interface{} {
	propertyHeaderName := sofarpc.SofaPropertyHeader(name)

	if value, ok := headers[propertyHeaderName]; ok {
		delete(headers, propertyHeaderName)

		return sofarpc.ConvertPropertyValue(value, BoltV1PropertyHeaderNames[name])
	} else {
		if value, ok := headers[name]; ok {

			return sofarpc.ConvertPropertyValue(value, BoltV1PropertyHeaderNames[name])
		}
	}

	return nil
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
						class = bytes[read: read+int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytes[read: read+int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytes[read: read+int(contentLen)]
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
						dataType,
						int16(cmdCode),
						ver2,
						requestId,
						codec,
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
						class = bytes[read: read+int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytes[read: read+int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytes[read: read+int(contentLen)]
						read += int(contentLen)
					}
				} else { // not enough data
					log.DefaultLogger.Println("[Decoder]no enough data for fully decode")
					return 0
				}

				response := &boltResponseCommand{
					boltCommand: boltCommand{
						sofarpc.PROTOCOL_CODE_V1,
						dataType,
						int16(cmdCode),
						ver2,
						requestId,
						codec,
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
