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
	case sofarpc.BoltRequestCommand:
		headerReq := headers.(*sofarpc.BoltRequestCommand)
		return c.encodeRequestCommand(headerReq)

	case sofarpc.BoltResponseCommand:
		headerRsp := headers.(*sofarpc.BoltResponseCommand)
		return c.encodeResponseCommand(headerRsp)

	case map[string]string:
		return c.EncodeHeadersMap(headers.(map[string]string))

	default:
		return 0, nil
	}
}

func (c *boltV1Codec) EncodeHeadersMap(headers map[string]string) (uint32, types.IoBuffer) {
	cmd := c.mapToCmd(headers)

	switch cmd.(type) {
	case *sofarpc.BoltRequestCommand:
		return c.encodeRequestCommand(cmd.(*sofarpc.BoltRequestCommand))
	case *sofarpc.BoltResponseCommand:
		return c.encodeResponseCommand(cmd.(*sofarpc.BoltResponseCommand))
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

func (c *boltV1Codec) encodeRequestCommand(cmd *sofarpc.BoltRequestCommand) (uint32, types.IoBuffer) {
	var result []byte

	result = append(result, cmd.Protocol, cmd.CmdType)
	cmdCodeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(cmdCodeBytes, uint16(cmd.CmdCode))
	result = append(result, cmdCodeBytes...)
	result = append(result, cmd.Version)

	requestIdBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(requestIdBytes, uint32(cmd.ReqId))
	result = append(result, requestIdBytes...)
	result = append(result, cmd.CodecPro)

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

	log.DefaultLogger.Println("rpc headers encode finished,bytes=%d", result)

	return cmd.ReqId, buffer.NewIoBufferBytes(result)
}

func (c *boltV1Codec) encodeResponseCommand(cmd *sofarpc.BoltResponseCommand) (uint32, types.IoBuffer) {

	var result []byte

	result = append(result, cmd.Protocol, cmd.CmdType)
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

		request := &sofarpc.BoltRequestCommand{
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
		}

		return request
	} else if cmdCode == sofarpc.RPC_RESPONSE {
		//todo : review
		responseStatus := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "responsestatus")
		responseTime := sofarpc.GetPropertyValue(BoltV1PropertyHeaders, headers, "responsetimemills")

		//serialize header
		header, _ := serialize.Instance.Serialize(headers)
		response := &sofarpc.BoltResponseCommand{
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
					log.DefaultLogger.Println("[Decoder]no enough data for fully decode")
					return 0, nil
				}

				request := &sofarpc.BoltRequestCommand{

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
				} else {
					// not enough data
					log.DefaultLogger.Println("[Decoder]no enough data for fully decode")

					return 0, nil
				}

				response := &sofarpc.BoltResponseCommand{

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
				}

				log.DefaultLogger.Printf("[Decoder]bolt v1 decode response:%+v\n", response)

				cmd = response
			}
		}
	}
	return read, cmd
}
