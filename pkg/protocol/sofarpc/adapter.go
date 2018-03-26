package sofarpc

import (
	"fmt"
	"encoding/binary"
	"reflect"
)

//ADAPT MAP[STRING]STRING TO COMMAND

func UintToString(value interface{}, len int) string {

	var result string
	if len == 16 {
		v := value.(uint16)
		vBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(vBytes, v)
		result = string(vBytes[:])

	} else if len == 32 {

		v := value.(uint32)
		vBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(vBytes, v)
		result = string(vBytes[:])
	} else if len == 64 {
		v := value.(uint64)
		vBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(vBytes, v)
		result = string(vBytes[:])

	} else {
		fmt.Println("Error input for converting")
	}
	return result
}

func String2Uint(in string, len int) interface{} {

	inByte := []byte(in)

	if len == 16 {

		resultUint16 := binary.BigEndian.Uint16(inByte)
		return resultUint16

	} else if len == 32 {
		resultUint32 := binary.BigEndian.Uint32(inByte)
		return resultUint32
	} else if len == 64 {

		resultUint64 := binary.BigEndian.Uint64(inByte)
		return resultUint64
	} else {
		fmt.Println("Error input for converting")
	}
	return nil
}

func SofaPropertyHeader(name string) string {
	return fmt.Sprintf("%s%s", SofaRpcPropertyHeaderPrefix, name)
}

func ConvertPropertyValue(strValue string, kind reflect.Kind) interface{} {
	switch kind {
	case reflect.Uint8:
		return []byte(strValue)[0]
	case reflect.Int16:
		return int16(String2Uint(strValue, 16).(uint16))
	case reflect.Uint32:
		return String2Uint(strValue, 32).(uint32)
	case reflect.Int:
		return int(String2Uint(strValue, 32).(uint32))
	case reflect.Int64:
		return int64(String2Uint(strValue, 64).(uint64))
	default:
		return strValue
	}
}
