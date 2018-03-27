package codec

import (
	"encoding/binary"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"time"
	"reflect"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
)

// types.Encoder & types.Decoder

var (
	BoltV2PropertyHeaders = make(map[string]reflect.Kind, 14)
)

func init() {
	BoltV2PropertyHeaders["protocol"] = reflect.Uint8
	BoltV2PropertyHeaders["cmdtype"] = reflect.Uint8
	BoltV2PropertyHeaders["cmdcode"] = reflect.Int16
	BoltV2PropertyHeaders["version"] = reflect.Uint8
	BoltV2PropertyHeaders["requestid"] = reflect.Uint32
	BoltV2PropertyHeaders["codec"] = reflect.Uint8
	BoltV2PropertyHeaders["classlength"] = reflect.Int16
	BoltV2PropertyHeaders["headerlength"] = reflect.Int16
	BoltV2PropertyHeaders["contentlength"] = reflect.Int
	BoltV2PropertyHeaders["timeout"] = reflect.Int
	BoltV2PropertyHeaders["responsestatus"] = reflect.Int16
	BoltV2PropertyHeaders["responsetimemills"] = reflect.Int64

	BoltV2PropertyHeaders["ver1"] = reflect.Uint8
	BoltV2PropertyHeaders["switchcode"] = reflect.Uint8

}

type boltV2Codec struct {}

func (c *boltV2Codec) EncodeHeaders(headers map[string]string) (uint32, types.IoBuffer) {
	cmd := c.mapToCmd(headers)

	switch cmd.(type) {
	case *boltRequestCommand:
		return c.encodeRequestCommand(cmd.(*boltRequestCommand))
	case *boltResponseCommand:
		return c.encodeResponseCommand(cmd.(*boltResponseCommand))
	default:
		log.DefaultLogger.Println("[BOLTV2 Decode] Invalid Input Type")
		return 0, nil
	}
}
func (c *boltV2Codec) EncodeData(data types.IoBuffer) types.IoBuffer {
	return data
}
func (c *boltV2Codec) EncodeTrailers(trailers map[string]string) types.IoBuffer {
	return nil
}


func (c *boltV2Codec) encodeRequestCommand(rpcCmd *boltRequestCommand) (uint32, types.IoBuffer) {

	log.DefaultLogger.Println("[BOLTV2]start to encode rpc REQUEST headers,=%+v", rpcCmd.cmdType)

	var result []byte

	result = append(result, rpcCmd.protocol, rpcCmd.ver1,rpcCmd.cmdType) //PAY ATTENTION TO THE DIFFERENCE WITH BOLT V1

	cmdCodeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(cmdCodeBytes, uint16(rpcCmd.cmdCode))
	result = append(result, cmdCodeBytes...)
	result = append(result, rpcCmd.version)


	requestIdBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(requestIdBytes, uint32(rpcCmd.id))
	result = append(result, requestIdBytes...)
	result = append(result, rpcCmd.codec)

	result = append(result, rpcCmd.switchCode) //PAY ATTENTION TO THE DIFFERENCE WITH BOLT V1

	timeoutBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(timeoutBytes, uint32(rpcCmd.timeout))
	result = append(result, timeoutBytes...)

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

	log.DefaultLogger.Println("[BOLTV2]rpc headers encode finished,bytes=%d", result)

	return rpcCmd.id, buffer.NewIoBufferBytes(result)
}

func (c *boltV2Codec) encodeResponseCommand(rpcCmd *boltResponseCommand) (uint32, types.IoBuffer) {
	log.DefaultLogger.Println("[BOLTV2]start to encode rpc RESPONSE headers,=%+v", rpcCmd.cmdType)

	var result []byte

	result = append(result, rpcCmd.protocol, rpcCmd.ver1,rpcCmd.cmdType) //PAY ATTENTION TO THE DIFFERENCE WITH BOLT V1


	cmdCodeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(cmdCodeBytes, uint16(rpcCmd.cmdCode))
	result = append(result, cmdCodeBytes...)
	result = append(result, rpcCmd.version)

	requestIdBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(requestIdBytes, uint32(rpcCmd.id))
	result = append(result, requestIdBytes...)
	result = append(result, rpcCmd.codec)

	result = append(result, rpcCmd.switchCode) ////PAY ATTENTION TO THE DIFFERENCE WITH BOLT V1

	respStatusBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(respStatusBytes, uint16(rpcCmd.responseStatus))
	result = append(result, respStatusBytes...)

	result = append(result, rpcCmd.switchCode)
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

	log.DefaultLogger.Println("[BOLTV2]rpc headers encode finished,bytes=%d", result)

	return rpcCmd.id, buffer.NewIoBufferBytes(result)
}

func (c *boltV2Codec) mapToCmd(headers map[string]string) interface{} {
	if len(headers) < 12 {
		return nil
	}

	protocolCode := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "protocol")
	cmdType := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "cmdtype")
	cmdCode := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "cmdcode")
	version := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "version")
	requestID := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "requestid")
	codec := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "codec")
	classLength := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "classlength")
	headerLength := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "headerlength")
	contentLength := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "contentlength")

	ver1 := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "ver1")
	switchcode := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "switchcode")


	//class
	className := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "classname")
	class, _ := serialize.Instance.Serialize(className)

	//RPC Request
	if cmdCode == sofarpc.RPC_REQUEST {
		timeout := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "timeout")

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
			ver1:ver1.(byte),
			switchCode:switchcode.(byte),
		}

		return request
	} else if cmdCode == sofarpc.RPC_RESPONSE {

		responseStatus := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "responsestatus")
		responseTime := sofarpc.GetPropertyValue(BoltV2PropertyHeaders,headers, "responsetimemills")

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
			ver1:ver1.(byte),
			switchCode:switchcode.(byte),
			responseStatus:     responseStatus.(int16),
			responseTimeMillis: responseTime.(int64),
		}

		return response
	} else {
		// todo RPC_HB
	}

	return nil
}


func (c *boltV2Codec) Decode(data types.IoBuffer) (int, interface{}) {
	readableBytes := data.Len()
	read := 0
	var cmd interface{}

	if readableBytes >= sofarpc.LESS_LEN_V2 {
		bytes := data.Bytes()

		ver1 := bytes[1]

		dataType := bytes[2]

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
					log.DefaultLogger.Println("[BOLTV2 Decoder]no enough data for fully decode")
					return 0,nil
				}

				request := &boltRequestCommand{
					boltCommand: boltCommand{
						sofarpc.PROTOCOL_CODE_V2,
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
					ver1:ver1,
					switchCode:switchCode,
					timeout:    int(timeout),
					arriveTime: time.Now().UnixNano() / int64(time.Millisecond),
				}

				log.DefaultLogger.Printf("[Decoder]bolt v2 decode request:%+v\n", request)

				cmd = request
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
					log.DefaultLogger.Println("[BOLTBV2 Decoder]no enough data for fully decode")
					return 0,nil
				}

				response := &boltResponseCommand{
					boltCommand: boltCommand{
						sofarpc.PROTOCOL_CODE_V2,
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
					ver1:ver1,
					switchCode:switchCode,
					responseStatus:     int16(status),
					responseTimeMillis: time.Now().UnixNano() / int64(time.Millisecond),
				}

				log.DefaultLogger.Printf("[Decoder]bolt v2 decode response:%+v\n", response)
				cmd = response
			}
		}
	}

	return read,cmd
}