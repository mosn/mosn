package sofarpc

import(

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
	}else if len == 64 {
		v      := value.(uint64)
		vBytes := make([]byte,8)
		binary.BigEndian.PutUint64(vBytes,v)
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
	}else if len == 64 {

		resultUint64 := binary.BigEndian.Uint64(inByte)
		return resultUint64
	}else {
		fmt.Println("Error input for converting")
	}
	return nil
}

func KeyInString(str[]string,key string) bool{

	for _,v := range str{

		if v == key {
			return true
		}

	}
	return  false
}