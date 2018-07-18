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
	"context"
	"reflect"
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/types"
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

func IsSofaRequest(headers map[string]string) bool {
	procode := ConvertPropertyValue(headers[SofaPropertyHeader(HeaderProtocolCode)], reflect.Uint8)

	if procode == PROTOCOL_CODE_V1 || procode == PROTOCOL_CODE_V2 {
		cmdtype := ConvertPropertyValue(headers[SofaPropertyHeader(HeaderCmdType)], reflect.Uint8)

		if cmdtype == REQUEST {
			return true
		}
	} else if procode == PROTOCOL_CODE_TR {
		requestFlage := ConvertPropertyValue(headers[SofaPropertyHeader(HeaderReqFlag)], reflect.Uint8)

		if requestFlage == HEADER_REQUEST {
			return true
		}
	}

	return false
}

func GetMap(context context.Context, defaultSize int) map[string]string {
	var amap map[string]string

	if context != nil && context.Value(types.ContextKeyConnectionCodecMapPool) != nil {
		pool := context.Value(types.ContextKeyConnectionCodecMapPool).(types.HeadersBufferPool)
		amap = pool.Take(defaultSize)
	}

	if amap == nil {
		amap = make(map[string]string, defaultSize)
	}

	return amap
}

func ReleaseMap(context context.Context, amap map[string]string) {
	if context != nil && context.Value(types.ContextKeyConnectionCodecMapPool) != nil {
		pool := context.Value(types.ContextKeyConnectionCodecMapPool).(types.HeadersBufferPool)
		pool.Give(amap)
	}
}
