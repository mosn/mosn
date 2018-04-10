package codec

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"encoding/binary"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	sf"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"reflect"
)

// types.Encoder & types.Decoder
type trCodec struct {}

var (
	TrPropertyHeaders = make(map[string]reflect.Kind, 8)
)

func init() {
	TrPropertyHeaders["protocol"] = reflect.Uint8
	TrPropertyHeaders["requestflag"] = reflect.Uint8
	TrPropertyHeaders["serializeprotocol"] = reflect.Uint8 //
	TrPropertyHeaders["direction"] = reflect.Uint8
	TrPropertyHeaders["reserved"] = reflect.Uint8
	TrPropertyHeaders["appclassnamelen"] = reflect.Uint8

	TrPropertyHeaders["connrequestlen"] = reflect.Uint32
	TrPropertyHeaders["appclasscontentlen"] = reflect.Uint32

	TrPropertyHeaders["cmdcode"] = reflect.Int16

	TrPropertyHeaders["requestid"] = reflect.Int64
}
/**
 *   Header(1B): 报文版本
 *   Header(1B): 请求/响应
 *   Header(1B): 报文协议(HESSIAN/JAVA)
 *   Header(1B): 单向/双向(响应报文中不使用这个字段)
 *   Header(1B): Reserved
 *   Header(4B): 通信层对象长度
 *   Header(1B): 应用层对象类名长度
 *   Header(4B): 应用层对象长度
//以下使用HESSION序列化
 *   Body:       通信层对象
 *   Body:       应用层对象类名
 *   Body:       应用层对象
 */


func (c *trCodec) EncodeHeaders(headers interface{}) (uint32, types.IoBuffer) {

	if headerMap, ok := headers.(map[string]string); ok {

		cmd := c.mapToCmd(headerMap)
		return c.encodeHeaders(cmd)
	}
	return c.encodeHeaders(headers)
}

func (c *trCodec) encodeHeaders(headers interface{}) (uint32, types.IoBuffer) {

	switch headers.(type) {
	case *sf.TrRequestCommand:
		return c.encodeRequestCommand(headers.(*sf.TrRequestCommand))
	case *sf.TrResponseCommand:
		return c.encodeResponseCommand(headers.(*sf.TrResponseCommand))
	default:
		log.DefaultLogger.Println("[Decode] Invalid Input Type")
		return 0, nil
	}
}

func (c *trCodec) encodeRequestCommand(rpcCmd *sf.TrRequestCommand) (uint32, types.IoBuffer) {

	log.DefaultLogger.Println("[TR]start to encode rpc headers,protocol code=%+v", rpcCmd.Protocol)

	var result []byte
	result = append(result, rpcCmd.Protocol, rpcCmd.RequestFlag)
	result = append(result, rpcCmd.SerializeProtocol, rpcCmd.Direction,rpcCmd.Reserved)

	connRequestLen := make([]byte, 4)
	binary.BigEndian.PutUint32(connRequestLen, rpcCmd.ConnRequestLen)
	result = append(result,connRequestLen...)

	result = append(result,rpcCmd.AppClassNameLen)

	appContentLen := make([]byte, 4)
	binary.BigEndian.PutUint32(appContentLen, rpcCmd.AppClassContentLen)
	result = append(result,appContentLen...)

	//todo AS TR's req id is 64bit long, need adjust
	return uint32(rpcCmd.RequestID),buffer.NewIoBufferBytes(result)
}

func (c *trCodec) encodeResponseCommand(rpcCmd *sf.TrResponseCommand) (uint32, types.IoBuffer) {
	log.DefaultLogger.Println("[TR]start to encode rpc headers,=%+v", rpcCmd.Protocol)

	var result []byte
	result = append(result, rpcCmd.Protocol, rpcCmd.RequestFlag)
	result = append(result, rpcCmd.SerializeProtocol, rpcCmd.Direction,rpcCmd.Reserved)

	connRequestLen := make([]byte, 4)
	binary.BigEndian.PutUint32(connRequestLen, rpcCmd.ConnRequestLen)
	result = append(result,connRequestLen...)

	result = append(result,rpcCmd.AppClassNameLen)

	appContentLen := make([]byte, 4)
	binary.BigEndian.PutUint32(appContentLen, rpcCmd.AppClassContentLen)
	result = append(result,appContentLen...)

	//todo AS TR's req id is 64bit long, need adjust
	return uint32(rpcCmd.RequestID),buffer.NewIoBufferBytes(result)
}

func (c *trCodec) EncodeData(data types.IoBuffer) types.IoBuffer {
	return data
}

func (c *trCodec) EncodeTrailers(trailers map[string]string) types.IoBuffer {
	return nil
}

func (c *trCodec)mapToCmd(headers_ interface{})interface{}{

	headers,_ := headers_.(map[string]string)
	if len(headers) < 8{
		return nil
	}

	protocol := sf.GetPropertyValue(TrPropertyHeaders,headers, "protocol")
	requestFlag := sf.GetPropertyValue(TrPropertyHeaders,headers, "requestflag")

	serializeProtocol := sf.GetPropertyValue(TrPropertyHeaders,headers, "serializeprotocol")
	direction := sf.GetPropertyValue(TrPropertyHeaders,headers, "direction")
	reserved := sf.GetPropertyValue(TrPropertyHeaders,headers, "reserved")

	appClassNameLen := sf.GetPropertyValue(TrPropertyHeaders,headers, "appclassnameLen")
	connRequestLen := sf.GetPropertyValue(TrPropertyHeaders,headers, "connrequestLen")
	appClassContentLen := sf.GetPropertyValue(TrPropertyHeaders,headers, "appclasscontentLen")

	cmdcode := sf.GetPropertyValue(TrPropertyHeaders,headers, "cmdcode")
	requestID := sf.GetPropertyValue(TrPropertyHeaders,headers, "requestID")

	if requestFlag == sf.HEADER_REQUEST{

		request := &sf.TrRequestCommand{
			TrCommand:sf.TrCommand{
				protocol.(byte),
				requestFlag.(byte),
				serializeProtocol.(byte),
				direction.(byte),
				reserved.(byte),
				connRequestLen.(uint32),
				appClassNameLen.(byte),
				appClassContentLen.(uint32),
				nil,
				"",
				nil,
			},
			CmdCode:cmdcode.(int16),
			RequestID:requestID.(int64),
		}
		return request
	} else if requestFlag == sf.HEADER_RESPONSE{
		response := &sf.TrResponseCommand{
			TrCommand:sf.TrCommand{
				protocol.(byte),
				requestFlag.(byte),
				serializeProtocol.(byte),
				direction.(byte),
				reserved.(byte),
				connRequestLen.(uint32),
				appClassNameLen.(byte),
				appClassContentLen.(uint32),
				nil,
				"",
				nil,
			},
			CmdCode:cmdcode.(int16),
			RequestID:requestID.(int64),
		}
		return response

	} else{
		//HB
	}

	return nil
}

func (decoder *trCodec) Decode(data types.IoBuffer) (int,interface{}) {
	readableBytes := data.Len()
	read := 0
	var cmd interface{}

	if readableBytes >= int(sf.PROTOCOL_HEADER_LENGTH) {
		bytes := data.Bytes()
		protocol := bytes[0]
		requestFlag := bytes[1]
		serializeProtocol := bytes[2]
		direction := bytes[3]
		reserved := bytes[4]
		connRequestLen := binary.BigEndian.Uint32(bytes[5:9])
		appClassNameLen := bytes[9]
		appClassContentLen := binary.BigEndian.Uint32(bytes[10:14])

		if uint32(readableBytes) < sf.PROTOCOL_HEADER_LENGTH + connRequestLen +
			uint32(appClassNameLen) + appClassContentLen {
			//not enough data
			log.DefaultLogger.Println("[Decoder]no enough data for fully decode")
			return 0, nil
		}

		connRequestEnd := 14 + connRequestLen
		connRequestContent := bytes[14:connRequestEnd]
		appClassNameEnd := connRequestEnd + uint32(appClassNameLen)
		appClassNameContent := bytes[connRequestEnd:appClassNameEnd]
		appClassName := string(appClassNameContent)
		appClassContent := bytes[appClassNameEnd: appClassNameEnd+appClassContentLen]
		totalLength := sf.PROTOCOL_HEADER_LENGTH + connRequestLen + uint32(appClassNameLen) + appClassContentLen
		data.Drain(int(totalLength))
		var cmdCode int16

		if requestFlag == sf.HEADER_REQUEST {
			if appClassName == "com.taobao.remoting.impl.ConnectionHeartBeat" {
				cmdCode = sf.TR_HEARTBEAT
			} else {
				cmdCode = sf.TR_REQUEST
			}
			request := &sf.TrRequestCommand{
				TrCommand: sf.TrCommand{
					protocol,
					requestFlag,
					serializeProtocol,
					direction,
					reserved,
					connRequestLen,
					appClassNameLen,
					appClassContentLen,
					connRequestContent,
					appClassName,
					appClassContent,
				},
				CmdCode:cmdCode,
				RequestContent:bytes[14 : 14 + connRequestLen + uint32(appClassNameLen) + appClassContentLen],
			}
			log.DefaultLogger.Printf("[Decoder]TR decode request:%+v\n", request)
			cmd = request
			read = int(totalLength)

		} else if requestFlag == sf.HEADER_RESPONSE {
			cmdCode = sf.TR_RESPONSE
			response := sf.TrResponseCommand{
				TrCommand: sf.TrCommand{
					protocol,
					requestFlag,
					serializeProtocol,
					direction,
					reserved,
					connRequestLen,
					appClassNameLen,
					appClassContentLen,
					connRequestContent,
					appClassName,
					appClassContent,
				},
				CmdCode:cmdCode,
				ResponseContent:bytes[14 : 14 + connRequestLen + uint32(appClassNameLen) + appClassContentLen],

			}
			cmd = response
			read = int(totalLength)
		}
	}
	return read,cmd
}
