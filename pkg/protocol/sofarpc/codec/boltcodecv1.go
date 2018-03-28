package codec

import (
	"time"
	"reflect"
	"encoding/binary"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
)

var (
	BoltV1PropertyHeaders = make(map[string]reflect.Kind, 11)
)

func init() {
	BoltV1PropertyHeaders["protocol"] = reflect.Uint8
	BoltV1PropertyHeaders["cmdtype"] = reflect.Uint8
	BoltV1PropertyHeaders["cmdcode"] = reflect.Int16
	BoltV1PropertyHeaders["version"] = reflect.Uint8
	BoltV1PropertyHeaders["requestid"] = reflect.Uint32
	BoltV1PropertyHeaders["codec"] = reflect.Uint8
	BoltV1PropertyHeaders["classlength"] = reflect.Int16
	BoltV1PropertyHeaders["headerlength"] = reflect.Int16
	BoltV1PropertyHeaders["contentlength"] = reflect.Int
	BoltV1PropertyHeaders["timeout"] = reflect.Int
	BoltV1PropertyHeaders["responsestatus"] = reflect.Int16
	BoltV1PropertyHeaders["responsetimemills"] = reflect.Int64
}

// types.Encoder & types.Decoder
type boltV1Codec struct{}

func (c *boltV1Codec) EncodeHeaders(headers interface{}) (uint32, types.IoBuffer) {
	switch headers.(type) {
	case sofarpc.BoltRequestCommandV2:
		headerReq := headers.(sofarpc.BoltRequestCommandV2)
		return c.encodeRequestCommandV2(headerReq)
	case sofarpc.BoltResponseCommand:
		// todo
		return 0, nil
	case map[string]string:
		return c.EncodeHeadersMap(headers.(map[string]string))
	default:
		return 0, nil
	}
}

func (c *boltV1Codec) EncodeHeadersMap(headers map[string]string) (uint32, types.IoBuffer) {
	cmd := c.mapToCmd(headers)

	switch cmd.(type) {
	case *boltRequestCommand:
		return c.encodeRequestCommand(cmd.(*boltRequestCommand))
	case *boltResponseCommand:
		return c.encodeResponseCommand(cmd.(*boltResponseCommand))
	default:
		log.DefaultLogger.Println("[Decode] Invalid Input Type")
		return 0, nil
	}
}

func (c *boltV1Codec) EncodeData(data types.IoBuffer) types.IoBuffer {
	return data
}

func (c *boltV1Codec) EncodeTrailers(trailers map[string]string) types.IoBuffer {
	return nil
}

func (c *boltV1Codec) encodeRequestCommandV2(cmd sofarpc.BoltRequestCommandV2) (uint32, types.IoBuffer) {
	var result []byte

	result = append(result, cmd.ProtocolCode, cmd.CmdType)

	// todo

	return cmd.Id, buffer.NewIoBufferBytes(result)
}

func (c *boltV1Codec) encodeRequestCommand(rpcCmd *boltRequestCommand) (uint32, types.IoBuffer) {
	log.DefaultLogger.Println("start to encode rpc headers,=%+v", rpcCmd.cmdType)

	var result []byte

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

	log.DefaultLogger.Println("rpc headers encode finished,bytes=%d", result)

	return rpcCmd.id, buffer.NewIoBufferBytes(result)
}

func (c *boltV1Codec) encodeResponseCommand(rpcCmd *boltResponseCommand) (uint32, types.IoBuffer) {
	log.DefaultLogger.Println("start to encode rpc headers,=%+v", rpcCmd.cmdType)

	var result []byte

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

	log.DefaultLogger.Println("rpc headers encode finished,bytes=%d", result)

	return rpcCmd.id, buffer.NewIoBufferBytes(result)
}

func (c *boltV1Codec) mapToCmd(headers map[string]string) interface{} {
	if len(headers) < 10 {
		return nil
	}

	protocolCode := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "protocol")
	cmdType := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "cmdtype")
	cmdCode := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "cmdcode")
	version := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "version")
	requestID := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "requestid")
	codec := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "codec")
	classLength := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "classlength")
	headerLength := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "headerlength")
	contentLength := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "contentlength")

	//class
	className := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "classname")
	class, _ := serialize.Instance.Serialize(className)

	//RPC Request
	if cmdCode == sofarpc.RPC_REQUEST {
		timeout := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "timeout")

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
		responseStatus := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "responsestatus")
		responseTime := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "responsetimemills")

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

func (c *boltV1Codec) Decode(data types.IoBuffer) (int, interface{}) {
	readableBytes := data.Len()
	read := 0
	var cmd interface{}

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

					data.Set(read)
				} else { // not enough data
					log.DefaultLogger.Println("[Decoder]no enough data for fully decode")
					return 0, nil
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
					timeout:    int(timeout),
					arriveTime: time.Now().UnixNano() / int64(time.Millisecond),
				}

				log.DefaultLogger.Printf("[Decoder]bolt v1 decode request:%+v\n", request)

				cmd = request
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

					data.Set(read)
				} else {
					// not enough data
					log.DefaultLogger.Println("[Decoder]no enough data for fully decode")

					return 0, nil
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
					responseStatus:     int16(status),
					responseTimeMillis: time.Now().UnixNano() / int64(time.Millisecond),
				}

				log.DefaultLogger.Printf("[Decoder]bolt v1 decode response:%+v\n", response)

				cmd = response
			}
		}
	}

	return read, cmd
}
