package codec

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"encoding/binary"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

const (
	HEADER_REQUEST byte = 0

	HEADER_RESPONSE byte = 1

	HESSIAN_SERIALIZE byte = 1

	JAVA_SERIALIZE byte = 2

	TOP_SERIALIZE byte = 3

	HESSIAN2_SERIALIZE byte = 4

	HEADER_ONEWAY byte = 1

	HEADER_TWOWAY byte = 2

	TR_REQUEST int16 = 13

	TR_RESPONSE int16 = 14

	TR_HEARTBEAT int16 = 0

	PROCOCOL_VERSION byte = 13

	PROTOCOL_HEADER_LENGTH uint32 = 14
)

// types.Encoder & types.Decoder
type trCodec struct {
}

/**
encode 这个
 */
func (encoder *trCodec) Encode(value interface{}, data types.IoBuffer) {

	//pass decode result to command handler
	if trCommand, ok := value.(*trCommand); ok {
		data.Append(trCommand.originalBytes)
	} else {
		log.DefaultLogger.Println("[Decoder]no enough data for fully decode")
	}

	if trCommand, ok := value.(*trRequestCommand); ok {

		data.AppendByte(PROCOCOL_VERSION)
		data.AppendByte(HEADER_REQUEST)
		//TODO 这里的协议要允许扩展,不能写死,最好从外面传递过来
		data.AppendByte(HESSIAN2_SERIALIZE)
		data.AppendByte(trCommand.direction)
		data.AppendByte(0)
		connRequestLength := len(trCommand.connRequestContent)
		connRequestLengthByte := make([]byte, 4)
		binary.BigEndian.PutUint32(connRequestLengthByte, uint32(connRequestLength))

		data.Append(connRequestLengthByte)

		appClassNameLength := len([]byte(trCommand.appClassName))

		//appClassNameLength
		data.AppendByte(byte(appClassNameLength))

		appContentLengthByte := make([]byte, 4)

		appContentLength := len(trCommand.appClassContent)
		binary.BigEndian.PutUint32(appContentLengthByte, uint32(appContentLength))

		data.Append(appContentLengthByte)

		data.Append(trCommand.connRequestContent)
		data.Append([]byte(trCommand.appClassName))
		data.Append(trCommand.appClassContent)

	} else if trCommand, ok := value.(*trResponseCommand); ok {

		data.AppendByte(PROCOCOL_VERSION)
		data.AppendByte(HEADER_RESPONSE)
		//TODO 这里的协议要允许扩展,不能写死,最好从外面传递过来
		data.AppendByte(HESSIAN2_SERIALIZE)
		data.AppendByte(0)
		data.AppendByte(0)
		connRequestLength := len(trCommand.connRequestContent)
		connRequestLengthByte := make([]byte, 4)
		binary.BigEndian.PutUint32(connRequestLengthByte, uint32(connRequestLength))

		data.Append(connRequestLengthByte)

		appClassNameLength := len([]byte(trCommand.appClassName))

		//appClassNameLength
		data.AppendByte(byte(appClassNameLength))

		appContentLengthByte := make([]byte, 4)

		appContentLength := len(trCommand.appClassContent)
		binary.BigEndian.PutUint32(appContentLengthByte, uint32(appContentLength))

		data.Append(appContentLengthByte)

		data.Append(trCommand.connRequestContent)
		data.Append([]byte(trCommand.appClassName))
		data.Append(trCommand.appClassContent)

	} else {
		log.DefaultLogger.Println("[Decoder]no enough data for fully decode")
	}

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
 *   Body:       通信层对象
 *   Body:       应用层对象类名
 *   Body:       应用层对象
 */
func (decoder *trCodec) Decode(ctx interface{}, data types.IoBuffer, out interface{}) {
	fmt.Println("tr decode:", data.Bytes())

	bytes := data.Bytes()

	protocolVersion := bytes[0]

	requestFlag := bytes[1]

	serializeProtocol := bytes[2]

	direction := bytes[3]

	//跳过 reserved

	connRequestLength := binary.BigEndian.Uint32(bytes[5:9])

	appClassNameLength := uint32(bytes[9])

	appClassContentLength := binary.BigEndian.Uint32(bytes[10:14])

	connRequestEnd := 14 + connRequestLength
	connRequestContent := bytes[14:connRequestEnd]

	appClassNameEnd := connRequestEnd + appClassNameLength
	appClassNameContent := bytes[connRequestEnd:appClassNameEnd]
	appClassName := string(appClassNameContent)
	appClassContent := bytes[appClassNameEnd : appClassNameEnd+appClassContentLength]

	totalLength := PROTOCOL_HEADER_LENGTH + connRequestLength + appClassNameLength + appClassContentLength

	var cmdCode int16

	if requestFlag == HEADER_REQUEST {

		if appClassName == "com.taobao.remoting.impl.ConnectionHeartBeat" {

			direction = HEADER_TWOWAY
			cmdCode = TR_HEARTBEAT
		} else {
			cmdCode = TR_REQUEST
		}

		request := trRequestCommand{

			trCommand: trCommand{
				originalBytes:      bytes,
				protocolVersion:    protocolVersion,
				requestFlag:        requestFlag,
				protocol:           serializeProtocol,
				direction:          direction,
				cmdCode:            cmdCode,
				appClassName:       appClassName,
				connRequestContent: connRequestContent,
				appClassContent:    appClassContent,
				totalLength:        totalLength,
			},
		}
		//pass decode result to command handler
		if list, ok := out.(*[]trRequestCommand); ok {
			*list = append(*list, request)
		}

	} else if requestFlag == HEADER_RESPONSE {

		cmdCode = TR_RESPONSE

		response := trResponseCommand{

			trCommand: trCommand{
				originalBytes:      bytes,
				protocolVersion:    protocolVersion,
				requestFlag:        requestFlag,
				protocol:           serializeProtocol,
				direction:          direction,
				cmdCode:            cmdCode,
				appClassName:       appClassName,
				connRequestContent: connRequestContent,
				appClassContent:    appClassContent,
				totalLength:        totalLength,
			},
		}
		//pass decode result to command handler
		if list, ok := out.(*[]trResponseCommand); ok {
			*list = append(*list, response)
		}
	}

}
