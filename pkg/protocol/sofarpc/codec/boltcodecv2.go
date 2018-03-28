package codec

import (
	"encoding/binary"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"reflect"
	"time"
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

type boltV2Codec struct{}

func (c *boltV2Codec) EncodeHeaders(headers interface{}) (uint32, types.IoBuffer) {
	switch headers.(type) {
	case sofarpc.BoltRequestCommandV2:
		headerReq := headers.(*sofarpc.BoltRequestCommandV2)
		return c.encodeRequestCommand(headerReq)

	case sofarpc.BoltResponseCommandV2:
		headerRsp := headers.(*sofarpc.BoltResponseCommandV2)
		return c.encodeResponseCommand(headerRsp)

	case map[string]string:
		return c.EncodeHeadersMap(headers.(map[string]string))

	default:
		return 0, nil
	}
}
func (c *boltV2Codec) EncodeHeadersMap(headers map[string]string) (uint32, types.IoBuffer) {
	cmd := c.mapToCmd(headers)

	switch cmd.(type) {
	case *sofarpc.BoltRequestCommandV2:
		return c.encodeRequestCommand(cmd.(*sofarpc.BoltRequestCommandV2))
	case *sofarpc.BoltResponseCommandV2:
		return c.encodeResponseCommand(cmd.(*sofarpc.BoltResponseCommandV2))
	default:
		log.DefaultLogger.Println("[BoltV2 Decode] Invalid Input Type")
		return 0, nil
	}
}

func (c *boltV2Codec) EncodeData(data types.IoBuffer) types.IoBuffer {
	return data
}
func (c *boltV2Codec) EncodeTrailers(trailers map[string]string) types.IoBuffer {
	return nil
}
func (c *boltV2Codec) encodeRequestCommand(cmd *sofarpc.BoltRequestCommandV2) (uint32, types.IoBuffer) {
	var result []byte

	result = append(result, cmd.Protocol, cmd.Version1, cmd.CmdType) //PAY ATTENTION TO THE DIFFERENCE WITH BOLT V1
	cmdCodeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(cmdCodeBytes, uint16(cmd.CmdCode))
	result = append(result, cmdCodeBytes...)
	result = append(result, cmd.Version)

	requestIdBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(requestIdBytes, uint32(cmd.ReqId))
	result = append(result, requestIdBytes...)
	result = append(result, cmd.CodecPro)

	result = append(result, cmd.SwitchCode) //PAY ATTENTION TO THE DIFFERENCE WITH BOLT V1

	timeoutBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(timeoutBytes, uint32(cmd.Timeout))
	result = append(result, timeoutBytes...)

	clazzLengthBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(clazzLengthBytes, uint16(cmd.ClassLen))
	result = append(result, clazzLengthBytes...)

	headerLengthBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(headerLengthBytes, uint16(cmd.HeaderLen))
	result = append(result, headerLengthBytes...)

	contentLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(contentLenBytes, uint32(cmd.ContentLen))
	result = append(result, contentLenBytes...)

	if cmd.ClassLen > 0 {
		result = append(result, cmd.ClassName...)
	}

	if cmd.HeaderLen > 0 {
		result = append(result, cmd.HeaderMap...)
	}

	log.DefaultLogger.Println("[BOLTV2]rpc headers encode finished,bytes=%d", result)

	return cmd.ReqId, buffer.NewIoBufferBytes(result)
}
func (c *boltV2Codec) encodeResponseCommand(cmd *sofarpc.BoltResponseCommandV2) (uint32, types.IoBuffer) {

	var result []byte

	result = append(result, cmd.Protocol, cmd.Version1, cmd.CmdType)
	cmdCodeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(cmdCodeBytes, uint16(cmd.CmdCode))
	result = append(result, cmdCodeBytes...)
	result = append(result, cmd.Version)

	requestIdBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(requestIdBytes, uint32(cmd.ReqId))
	result = append(result, requestIdBytes...)
	result = append(result, cmd.CodecPro)

	respStatusBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(respStatusBytes, uint16(cmd.ResponseStatus))
	result = append(result, respStatusBytes...)

	clazzLengthBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(clazzLengthBytes, uint16(cmd.ClassLen))
	result = append(result, clazzLengthBytes...)

	headerLengthBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(headerLengthBytes, uint16(cmd.HeaderLen))
	result = append(result, headerLengthBytes...)

	contentLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(contentLenBytes, uint32(cmd.ContentLen))
	result = append(result, contentLenBytes...)

	if cmd.ClassLen > 0 {
		result = append(result, cmd.ClassName...)
	}

	if cmd.HeaderLen > 0 {
		result = append(result, cmd.HeaderMap...)
	}

	log.DefaultLogger.Println("rpc headers encode finished,bytes=%d", result)

	return cmd.ReqId, buffer.NewIoBufferBytes(result)

}

func (c *boltV2Codec) mapToCmd(headers map[string]string) interface{} {
	if len(headers) < 12 {
		return nil
	}

	protocolCode := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "protocol")
	cmdType := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "cmdtype")
	cmdCode := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "cmdcode")
	version := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "version")
	requestID := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "requestid")
	codec := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "codec")
	classLength := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "classlength")
	headerLength := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "headerlength")
	contentLength := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "contentlength")

	ver1 := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "ver1")
	switchcode := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "switchcode")

	//class
	className := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "classname")
	class, _ := serialize.Instance.Serialize(className)

	//RPC Request
	if cmdCode == sofarpc.RPC_REQUEST {
		timeout := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "timeout")

		//serialize header
		header, _ := serialize.Instance.Serialize(headers)

		request := &sofarpc.BoltRequestCommandV2{
			sofarpc.BoltRequestCommand{
				protocolCode.(byte),
				cmdType.(byte),
				cmdCode.(int16),
				version.(byte),
				requestID.(uint32),
				codec.(byte),
				timeout.(int),
				classLength.(int16),
				headerLength.(int16),
				contentLength.(int),
				class,
				header,
				nil,
				nil,
				nil,
			},
			ver1.(byte),
			switchcode.(byte),
		}
		return request
	} else if cmdCode == sofarpc.RPC_RESPONSE {

		responseStatus := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "responsestatus")
		responseTime := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "responsetimemills")

		//serialize header
		header, _ := serialize.Instance.Serialize(headers)

		response := &sofarpc.BoltResponseCommandV2{
			sofarpc.BoltResponseCommand{
				protocolCode.(byte),
				cmdType.(byte),
				cmdCode.(int16),
				version.(byte),
				requestID.(uint32),
				codec.(byte),
				responseStatus.(int16),
				classLength.(int16),
				headerLength.(int16),
				contentLength.(int),
				class,
				header,
				nil,
				nil,
				responseTime.(int64),
				nil,
			},
			ver1.(byte),
			switchcode.(byte),
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
					data.Set(read)
				} else { // not enough data
					log.DefaultLogger.Println("[BOLTV2 Decoder]no enough data for fully decode")
					return 0, nil
				}

				request := &sofarpc.BoltRequestCommandV2{
					sofarpc.BoltRequestCommand{
						sofarpc.PROTOCOL_CODE_V1,
						dataType,
						int16(cmdCode),
						ver2,
						requestId,
						codec,
						int(timeout),
						int16(classLen),
						int16(headerLen),
						int(contentLen),
						class,
						header,
						content,
						nil,
						nil,
					},
					ver1,
					switchCode,
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
					log.DefaultLogger.Println("[BOLTBV2 Decoder]no enough data for fully decode")
					return 0, nil
				}

				response := &sofarpc.BoltResponseCommandV2{
					sofarpc.BoltResponseCommand{

						sofarpc.PROTOCOL_CODE_V1,
						dataType,
						int16(cmdCode),
						ver2,
						requestId,
						codec,
						int16(status),
						int16(classLen),
						int16(headerLen),
						int(contentLen),
						class,
						header,
						content,
						nil,
						time.Now().UnixNano() / int64(time.Millisecond),
						nil,
					},
					ver1,
					switchCode,
				}

				log.DefaultLogger.Printf("[Decoder]bolt v2 decode response:%+v\n", response)
				cmd = response
			}
		}
	}

	return read, cmd
}
