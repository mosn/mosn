package codec

import (
	"context"
	"encoding/binary"
	"errors"
	"reflect"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	sf "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// types.Encoder & types.Decoder
type trCodec struct{}

var (
	TrPropertyHeaders = make(map[string]reflect.Kind, 8)
)

func init() {
	TrPropertyHeaders[sf.HeaderProtocolCode] = reflect.Uint8
	TrPropertyHeaders[sf.HeaderReqFlag] = reflect.Uint8
	TrPropertyHeaders[sf.HeaderSeriProtocol] = reflect.Uint8 //
	TrPropertyHeaders[sf.HeaderDirection] = reflect.Uint8
	TrPropertyHeaders[sf.HeaderReserved] = reflect.Uint8
	TrPropertyHeaders[sf.HeaderAppclassnamelen] = reflect.Uint8
	TrPropertyHeaders[sf.HeaderConnrequestlen] = reflect.Uint32
	TrPropertyHeaders[sf.HeaderAppclasscontentlen] = reflect.Uint32
	TrPropertyHeaders[sf.HeaderCmdCode] = reflect.Int16
	TrPropertyHeaders[sf.HeaderReqID] = reflect.Int64
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

func (c *trCodec) EncodeHeaders(context context.Context, headers interface{}) (error, types.IoBuffer) {
	if headerMap, ok := headers.(map[string]string); ok {
		cmd := c.mapToCmd(headerMap)
		return c.encodeHeaders(context, cmd)
	}

	return c.encodeHeaders(context, headers)
}

func (c *trCodec) encodeHeaders(context context.Context, headers interface{}) (error, types.IoBuffer) {
	switch headers.(type) {
	case *sf.TrRequestCommand:
		return c.encodeRequestCommand(context, headers.(*sf.TrRequestCommand))
	case *sf.TrResponseCommand:
		return c.encodeResponseCommand(context, headers.(*sf.TrResponseCommand))
	default:
		errMsg := sf.InvalidCommandType
		err := errors.New(errMsg)
		log.ByContext(context).Errorf(errMsg)

		return err, nil
	}
}

func (c *trCodec) encodeRequestCommand(context context.Context, rpcCmd *sf.TrRequestCommand) (error, types.IoBuffer) {
	log.ByContext(context).Debugf("TR encode start to encode rpc headers,protocol code=%+v", rpcCmd.Protocol)

	var result []byte
	result = append(result, rpcCmd.Protocol, rpcCmd.RequestFlag)
	result = append(result, rpcCmd.SerializeProtocol, rpcCmd.Direction, rpcCmd.Reserved)

	connRequestLen := make([]byte, 4)
	binary.BigEndian.PutUint32(connRequestLen, rpcCmd.ConnRequestLen)
	result = append(result, connRequestLen...)

	result = append(result, rpcCmd.AppClassNameLen)

	appContentLen := make([]byte, 4)
	binary.BigEndian.PutUint32(appContentLen, rpcCmd.AppClassContentLen)
	result = append(result, appContentLen...)

	return nil, buffer.NewIoBufferBytes(result)
}

func (c *trCodec) encodeResponseCommand(context context.Context, rpcCmd *sf.TrResponseCommand) (error, types.IoBuffer) {
	log.ByContext(context).Debugf("[TR]start to encode rpc headers,=%+v", rpcCmd.Protocol)

	var result []byte
	result = append(result, rpcCmd.Protocol, rpcCmd.RequestFlag)
	result = append(result, rpcCmd.SerializeProtocol, rpcCmd.Direction, rpcCmd.Reserved)

	connRequestLen := make([]byte, 4)
	binary.BigEndian.PutUint32(connRequestLen, rpcCmd.ConnRequestLen)
	result = append(result, connRequestLen...)

	result = append(result, rpcCmd.AppClassNameLen)

	appContentLen := make([]byte, 4)
	binary.BigEndian.PutUint32(appContentLen, rpcCmd.AppClassContentLen)
	result = append(result, appContentLen...)

	return nil, buffer.NewIoBufferBytes(result)
}

func (c *trCodec) EncodeData(context context.Context, data types.IoBuffer) types.IoBuffer {
	return data
}

func (c *trCodec) EncodeTrailers(context context.Context, trailers map[string]string) types.IoBuffer {
	return nil
}

func (c *trCodec) mapToCmd(headers_ interface{}) interface{} {

	headers, _ := headers_.(map[string]string)
	if len(headers) < 8 {
		return nil
	}

	protocol := sf.GetPropertyValue(TrPropertyHeaders, headers, sf.HeaderProtocolCode)
	requestFlag := sf.GetPropertyValue(TrPropertyHeaders, headers, sf.HeaderReqFlag)

	serializeProtocol := sf.GetPropertyValue(TrPropertyHeaders, headers, sf.HeaderSeriProtocol)
	direction := sf.GetPropertyValue(TrPropertyHeaders, headers, sf.HeaderDirection)
	reserved := sf.GetPropertyValue(TrPropertyHeaders, headers, sf.HeaderReserved)

	appClassNameLen := sf.GetPropertyValue(TrPropertyHeaders, headers, sf.HeaderAppclassnamelen)
	connRequestLen := sf.GetPropertyValue(TrPropertyHeaders, headers, sf.HeaderConnrequestlen)
	appClassContentLen := sf.GetPropertyValue(TrPropertyHeaders, headers, sf.HeaderAppclasscontentlen)

	cmdcode := sf.GetPropertyValue(TrPropertyHeaders, headers, sf.HeaderCmdCode)
	requestID := sf.GetPropertyValue(TrPropertyHeaders, headers, sf.HeaderReqID)

	if requestFlag == sf.HEADER_REQUEST {

		request := &sf.TrRequestCommand{
			TrCommand: sf.TrCommand{
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
			CmdCode:   cmdcode.(int16),
			RequestID: requestID.(int64),
		}
		return request
	} else if requestFlag == sf.HEADER_RESPONSE {
		response := &sf.TrResponseCommand{
			TrCommand: sf.TrCommand{
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
			CmdCode:   cmdcode.(int16),
			RequestID: requestID.(int64),
		}
		return response

	} else {
		//HB
	}

	return nil
}

func (decoder *trCodec) Decode(context context.Context, data types.IoBuffer) (int, interface{}) {
	readableBytes := data.Len()
	read := 0
	var cmd interface{}
	logger := log.ByContext(context)

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

		if uint32(readableBytes) < sf.PROTOCOL_HEADER_LENGTH+connRequestLen+
			uint32(appClassNameLen)+appClassContentLen {
			//not enough data
			logger.Debugf("Decoderno enough data for fully decode")
			return 0, nil
		}

		connRequestEnd := 14 + connRequestLen
		connRequestContent := bytes[14:connRequestEnd]
		appClassNameEnd := connRequestEnd + uint32(appClassNameLen)
		appClassNameContent := bytes[connRequestEnd:appClassNameEnd]
		appClassName := string(appClassNameContent)
		appClassContent := bytes[appClassNameEnd : appClassNameEnd+appClassContentLen]
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
				CmdCode:        cmdCode,
				RequestContent: bytes[14 : 14+connRequestLen+uint32(appClassNameLen)+appClassContentLen],
			}
			logger.Debugf("DecoderTR decode request:%+v", request)
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
				CmdCode:         cmdCode,
				ResponseContent: bytes[14 : 14+connRequestLen+uint32(appClassNameLen)+appClassContentLen],
			}
			cmd = response
			read = int(totalLength)
		}
	}
	return read, cmd
}
