/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sofarpc

import (
	"reflect"
	"strconv"
)

func SofaPropertyHeader(name string) string {
	return name
}

func GetPropertyValue(properHeaders map[string]reflect.Kind, headers map[string]string, name string) interface{} {
	propertyHeaderName := SofaPropertyHeader(name)

	if value, ok := headers[propertyHeaderName]; ok {
		delete(headers, propertyHeaderName)

		return ConvertPropertyValue(value, properHeaders[name])
	}

	if value, ok := headers[name]; ok {

		return ConvertPropertyValue(value, properHeaders[name])
	}

	return nil
}

func ConvertPropertyValue(strValue string, kind reflect.Kind) interface{} {
	switch kind {
	case reflect.Uint8:
		value, _ := strconv.ParseUint(strValue, 10, 8)
		return byte(value)
	case reflect.Uint16:
		value, _ := strconv.ParseUint(strValue, 10, 16)
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

func GetPropertyValue1(properHeaders map[string]reflect.Kind, headers map[string]string, name string) string {
	propertyHeaderName := SofaPropertyHeader(name)

	if value, ok := headers[propertyHeaderName]; ok {
		delete(headers, propertyHeaderName)

		return value
	}

	if value, ok := headers[name]; ok {

		return value
	}

	return ""
}

func ConvertPropertyValueUint8(strValue string) byte {
	value, _ := strconv.ParseUint(strValue, 10, 8)
	return byte(value)
}

func ConvertPropertyValueUint16(strValue string) uint16 {
	value, _ := strconv.ParseUint(strValue, 10, 16)
	return uint16(value)
}

func ConvertPropertyValueUint32(strValue string) uint32 {
	value, _ := strconv.ParseUint(strValue, 10, 32)
	return uint32(value)
}

func ConvertPropertyValueUint64(strValue string) uint64 {
	value, _ := strconv.ParseUint(strValue, 10, 64)
	return uint64(value)
}

func ConvertPropertyValueInt8(strValue string) int8 {
	value, _ := strconv.ParseInt(strValue, 10, 8)
	return int8(value)
}

func ConvertPropertyValueInt16(strValue string) int16 {
	value, _ := strconv.ParseInt(strValue, 10, 16)
	return int16(value)
}

func ConvertPropertyValueInt(strValue string) int {
	value, _ := strconv.ParseInt(strValue, 10, 32)
	return int(value)
}

func ConvertPropertyValueInt64(strValue string) int64 {
	value, _ := strconv.ParseInt(strValue, 10, 64)
	return int64(value)
}
