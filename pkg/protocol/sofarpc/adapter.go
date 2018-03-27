package sofarpc

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func SofaPropertyHeader(name string) string {
	return fmt.Sprintf("%s%s", SofaRpcPropertyHeaderPrefix, strings.ToLower(name))
}

func  GetPropertyValue(properHeaders map[string]reflect.Kind,headers map[string]string, name string) interface{} {
	name = strings.ToLower(name)
	propertyHeaderName := SofaPropertyHeader(name)

	if value, ok := headers[propertyHeaderName]; ok {
		delete(headers, propertyHeaderName)

		return ConvertPropertyValue(value, properHeaders[name])
	} else {
		if value, ok := headers[name]; ok {

			return ConvertPropertyValue(value, properHeaders[name])
		}
	}

	return nil
}

func ConvertPropertyValue(strValue string, kind reflect.Kind) interface{} {
	switch kind {
	case reflect.Uint8:
		value, _ := strconv.ParseUint(strValue, 10, 8)
		return byte(value)
	case reflect.Uint16:
		value, _ := strconv.ParseUint(strValue, 10, 8)
		return uint16(value)
	case reflect.Uint32:
		value, _ := strconv.ParseUint(strValue, 10, 32)
		return uint32(value)
	case reflect.Uint64:
		value, _ := strconv.ParseUint(strValue, 10, 64)
		return uint64(value)
	case reflect.Int8:
		value, _ := strconv.ParseInt(strValue, 10, 8)
		return int8(value)
	case reflect.Int16:
		value, _ := strconv.ParseInt(strValue, 10, 16)
		return int16(value)
	case reflect.Int:
		value, _ := strconv.ParseInt(strValue, 10, 32)
		return int(value)
	case reflect.Int64:
		value, _ := strconv.ParseInt(strValue, 10, 64)
		return int64(value)
	default:
		return strValue
	}
}
