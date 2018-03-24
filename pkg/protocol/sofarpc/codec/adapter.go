package codec

import(
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"encoding/binary"
	"fmt"
)
/*
allField["XXX_protocol"] = string(requestCommand.GetProtocolCode())
allField["XXX_cmdType"]  = string(requestCommand.GetCmdType())
allField["XXX_cmdCode"] = uintToString(uint16(requestCommand.GetCmdCode()),16)
allField["XXX_version"] = string(requestCommand.GetVersion())
allField["XXX_requestId"] = uintToString(uint32(requestCommand.GetId()),32)
allField["XXX_codec"]  = string(requestCommand.GetCodec())

allField["XXX_timeout"] = uintToString(requestCommand.GetTimeout(),32)

allField["XXX_classLength"] = uintToString(uint16(requestCommand.GetClassLength()),16)
allField["XXX_headerLength"] = uintToString(uint16(requestCommand.GetHeaderLength()),16)
allField["XXX_contentLength"] = uintToString(uint32(requestCommand.GetCmdCode()),32)


//serialize class name
var className string
serialize.DeSerialize(requestCommand.GetClass(), &className)
fmt.Println("deSerialize class :", className)
allField["XXX_className"]  = className


//serialize header
var headerMap map[string]string
serialize.DeSerialize(requestCommand.GetHeader(), &headerMap)
fmt.Println("deSerialize  headerMap:", headerMap)
requestCommand.SetRequestHeader(headerMap)

for k,v := range headerMap {

allField[k] = v
}

return allField*/


//ADAPT MAP[STRING]STRING TO COMMAND
func EncodeAdapter(allField map[string]string)interface {}{

	keyList := []string{"XXX_protocol","XXX_cmdType","XXX_cmdCode","XXX_requestId","XXX_codec",
		"XXX_classLength", "XXX_headerLength","XXX_contentLength","XXX_className"}

	//COMMON FOR ALL
	protocolCode := []byte(allField["XXX_protocol"])[0]

	//BOLT V1
	if protocolCode == sofarpc.PROTOCOL_CODE_V1{

		cmdType := []byte(allField["XXX_cmdType"])[0]
		cmdCode := int16(String2Uint(allField["XXX_cmdCode"],16).(uint16))  //Interface{} need converted first
		version := []byte(allField["XXX_version"])[0]
		requestID := String2Uint(allField["XXX_requestId"],32).(uint32)
		codec := []byte(allField["XXX_codec"])[0]

		//RPC Request
		var timeout int
		if cmdCode == sofarpc.RPC_REQUEST {

			timeout =  int(String2Uint(allField["XXX_timeout"],32).(uint32))
			keyList = append(keyList,"XXX_timeout" )

		} else if cmdCode == sofarpc.RPC_RESPONSE {

			//todo RPC RESPONSE

		}else{
			// todo RPC_HB

		}

		classLength := int16(String2Uint(allField["XXX_classLength"],16).(uint16))
		headerLength := int16(String2Uint(allField["XXX_headerLength"],16).(uint16))
		contentLength :=  int(String2Uint(allField["XXX_contentLength"],32).(uint32))

		//class
		className := allField["XXX_className"]
		class,_ := serialize.Instance.Serialize(className)

		//header, reconstruct map，由于不知道KEY的内容，因而有点麻烦
		headerMap := make(map[string]string)

		for k,v := range allField {

			if keyInString(keyList,k){

				headerMap[k] = v
			}
		}

		//serialize header

		header,_:= serialize.Instance.Serialize(headerMap)


		request := &boltRequestCommand{

				boltCommand:boltCommand{
				protocolCode,
				cmdType,
				cmdCode,
				version,
				requestID,
				codec,
				classLength,
				headerLength,
				contentLength,
				class,
				header,
				nil,
				nil,
			},
		}
		request.SetTimeout(int(timeout))
		return request
	} else if protocolCode == sofarpc.PROTOCOL_CODE_V2 {

	}

	return nil
}



func UintToString(value interface{}, len int)string{

	var result string
	if len == 16 {
		v      := value.(uint16)
		vBytes := make([]byte,2)
		binary.BigEndian.PutUint16(vBytes,v)
		result= string(vBytes[:])

	}else if len == 32 {

		v      := value.(uint32)
		vBytes := make([]byte,4)
		binary.BigEndian.PutUint32(vBytes,v)
		result= string(vBytes[:])
	} else {
		fmt.Println("Error input for converting")
	}
	return result
}

func String2Uint(in string,len int) interface{}{

	inByte := []byte(in)

	if len == 16 {

		resultUint16 := binary.BigEndian.Uint16(inByte)
		return resultUint16

	} else if len == 32 {
		resultUint32 := binary.BigEndian.Uint32(inByte)
		return resultUint32

	}else {
		fmt.Println("Error input for converting")
	}
	return nil
}

func keyInString(str[]string,key string) bool{

	for _,v := range str{

		if v == key {
			return true
		}

	}
	return  false
}