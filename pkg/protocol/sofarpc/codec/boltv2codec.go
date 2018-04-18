package codec

import (
	"encoding/binary"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/utility"
	"reflect"
	"time"
)

// types.Encoder & types.Decoder

var (
	BoltV2PropertyHeaders = make(map[string]reflect.Kind, 14)
)

var boltV1 = &boltV1Codec{}

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

func (c *boltV2Codec) EncodeHeaders(headers interface{}) (string, types.IoBuffer) {

	if headerMap, ok := headers.(map[string]string); ok {

		cmd := c.mapToCmd(headerMap)
		return c.encodeHeaders(cmd)
	}
	return c.encodeHeaders(headers)

}

func (c *boltV2Codec) encodeHeaders(headers interface{}) (string, types.IoBuffer) {

	switch headers.(type) {
	case *sofarpc.BoltV2RequestCommand:
		return c.encodeRequestCommand(headers.(*sofarpc.BoltV2RequestCommand))
	case *sofarpc.BoltV2ResponseCommand:
		return c.encodeResponseCommand(headers.(*sofarpc.BoltV2ResponseCommand))
	default:
		err := "[BoltV2 Encode] Invalid Input Type"
		streamID := utility.GenerateExceptionStreamID(err)
		log.DefaultLogger.Println(err)
		return streamID, nil
	}
}

func (c *boltV2Codec) EncodeData(data types.IoBuffer) types.IoBuffer {
	return data
}

func (c *boltV2Codec) EncodeTrailers(trailers map[string]string) types.IoBuffer {
	return nil
}

func (c *boltV2Codec) encodeRequestCommand(cmd *sofarpc.BoltV2RequestCommand) (string, types.IoBuffer) {
	result := boltV1.doEncodeRequestCommand(&cmd.BoltRequestCommand)

	c.insertToBytes(result, 1, cmd.Version1)
	c.insertToBytes(result, 11, cmd.SwitchCode)

	log.DefaultLogger.Println("[BOLTV2]rpc headers encode finished,bytes=%d", result)

	return utility.StreamIDConvert(cmd.ReqId), buffer.NewIoBufferBytes(result)
}

func (c *boltV2Codec) encodeResponseCommand(cmd *sofarpc.BoltV2ResponseCommand) (string, types.IoBuffer) {

	result := boltV1.doEncodeResponseCommand(&cmd.BoltResponseCommand)

	c.insertToBytes(result, 1, cmd.Version1)
	c.insertToBytes(result, 11, cmd.SwitchCode)

	log.DefaultLogger.Println("rpc headers encode finished,bytes=%d", result)

	return utility.StreamIDConvert(cmd.ReqId), buffer.NewIoBufferBytes(result)

}

func (c *boltV2Codec) mapToCmd(headers map[string]string) interface{} {
	if len(headers) < 12 {
		return nil
	}

	cmdV1 := boltV1.mapToCmd(headers)

	ver1 := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "ver1")
	switchcode := sofarpc.GetPropertyValue(BoltV2PropertyHeaders, headers, "switchcode")

	if cmdV2req, ok := cmdV1.(sofarpc.BoltRequestCommand); ok {

		request := &sofarpc.BoltV2RequestCommand{
			BoltRequestCommand: cmdV2req,
			Version1:           ver1.(byte),
			SwitchCode:         switchcode.(byte),
		}
		return request

	} else if cmdV2res, ok := cmdV1.(sofarpc.BoltResponseCommand); ok {

		response := &sofarpc.BoltV2ResponseCommand{
			BoltResponseCommand: cmdV2res,
			Version1:            ver1.(byte),
			SwitchCode:          switchcode.(byte),
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
					data.Drain(read)
				} else { // not enough data
					log.DefaultLogger.Println("[BOLTV2 Decoder]no enough data for fully decode")
					return 0, nil
				}

				request := &sofarpc.BoltV2RequestCommand{
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

				response := &sofarpc.BoltV2ResponseCommand{
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

func (c *boltV2Codec) insertToBytes(slice []byte, idx int, b byte) {
	slice = append(slice, 0)
	copy(slice[idx+1:], slice[idx:])
	slice[idx] = b
}
