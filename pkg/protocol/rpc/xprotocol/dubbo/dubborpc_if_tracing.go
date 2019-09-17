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

package dubbo

import (
	"fmt"
	"strconv"

	"github.com/AlexStocks/dubbogo/codec/hessian"
	"regexp"
)

// regular
const (
	JAVA_IDENT_REGEX = "(?:[_$a-zA-Z][_$a-zA-Z0-9]*)"
	CLASS_DESC       = "(?:L" + JAVA_IDENT_REGEX + "(?:\\/" + JAVA_IDENT_REGEX + ")*;)"
	ARRAY_DESC       = "(?:\\[+(?:(?:[VZBCDFIJS])|" + CLASS_DESC + "))"
	DESC_REGEX       = "(?:(?:[VZBCDFIJS])|" + CLASS_DESC + "|" + ARRAY_DESC + ")"

	RESPONSE_WITH_EXCEPTION                  int32 = 0
	RESPONSE_WITH_EXCEPTION_WITH_ATTACHMENTS int32 = 3
)

// DescRegex ...
var DescRegex, _ = regexp.Compile(DESC_REGEX)

func init() {
	serviceNameFunc = dubboGetServiceName
	methodNameFunc = dubboGetMethodName
	metaFunc = dubboGetMeta
}

func getSerializeId(flag byte) int {
	return int(flag & 0x1f)
}

func getEventPing(flag byte) bool {
	return (flag & (1 << 5)) != 0
}

func isReqFrame(flag byte) bool {
	return (flag & (1 << 7)) != 0
}

type dubboAttr struct {
	serviceName  string
	methodName   string
	path         string
	version      string
	dubboVersion string
	attachments  map[string]string
}

func unSerialize(serializeId int, data []byte, parseAttachments bool) *dubboAttr {
	if serializeId != 2 {
		// not hessian, do not support
		fmt.Printf("unSerialize: id=%d is not hessian\n", serializeId)
		return nil
	}
	attr := &dubboAttr{}
	decoder := hessian.NewDecoder(data[:])
	var field interface{}
	var err error
	var ok bool
	var str string
	var attachments map[string]string

	// xprotocol version + path + version + method

	field, err = decoder.Decode()
	if err != nil {
		fmt.Printf("unSerialize: Decode dubbo_version fail, err=%v\n", err)
		return nil
	}
	str, ok = field.(string)
	if !ok {
		fmt.Printf("unSerialize: Decode dubbo_version fail, illegal type\n")
		return nil
	}
	attr.dubboVersion = str

	field, err = decoder.Decode()
	if err != nil {
		fmt.Printf("unSerialize: Decode path fail, err=%v\n", err)
		return nil
	}
	str, ok = field.(string)
	if !ok {
		fmt.Printf("unSerialize: Decode path fail, illegal type\n")
		return nil
	}
	attr.serviceName = str
	attr.path = str

	field, err = decoder.Decode()
	if err != nil {
		fmt.Printf("unSerialize: Decode version fail, err=%v\n", err)
		return nil
	}
	str, ok = field.(string)
	if !ok {
		fmt.Printf("unSerialize: Decode version fail, illegal type\n")
		return nil
	}
	attr.version = str

	field, err = decoder.Decode()
	if err != nil {
		fmt.Printf("unSerialize: Decode method fail, err=%v\n", err)
		return nil
	}
	str, ok = field.(string)
	if !ok {
		fmt.Printf("unSerialize: Decode method fail, illegal type\n")
		return nil
	}
	attr.methodName = str

	if !parseAttachments {
		return attr
	}

	field, err = decoder.Decode()
	if err != nil {
		fmt.Printf("unSerialize: Decode argsTypes fail, err=%v\n", err)
		return nil
	}

	ats := DescRegex.FindAllString(field.(string), -1)
	for i := 0; i < len(ats); i++ {
		_, err = decoder.Decode()
		if err != nil {
			fmt.Printf("unSerialize: Decode argsTypes item fail, err=%v\n", err)
			return nil
		}
	}

	field, err = decoder.Decode()
	if err != nil {
		fmt.Printf("unSerialize: Decode attachments fail, err=%v\n", err)
		return nil
	}
	if v, ok := field.(map[interface{}]interface{}); ok {
		attachments = ToMapStringString(v)
		attr.attachments = attachments
	}

	return attr
}

func dubboGetServiceName(data []byte) string {
	rslt, bodyLen := isValidDubboData(data)
	if rslt == false || bodyLen <= 0 {
		return ""
	}

	flag := data[DUBBO_FLAG_IDX]
	if getEventPing(flag) {
		// heart-beat frame, there is not service-name
		return ""
	}
	if isReqFrame(flag) != true {
		// response frame, there is not service-name
		return ""
	}
	serializeId := getSerializeId(flag)
	ret := unSerialize(serializeId, data[DUBBO_HEADER_LEN:], false)
	serviceName := ""
	if ret != nil {
		serviceName = ret.serviceName
	}
	return serviceName
}

func dubboGetMethodName(data []byte) string {
	//return "dubboMethod"
	rslt, bodyLen := isValidDubboData(data)
	if rslt == false || bodyLen <= 0 {
		return ""
	}

	flag := data[DUBBO_FLAG_IDX]
	if getEventPing(flag) {
		// heart-beat frame, there is not method-name
		return ""
	}
	if isReqFrame(flag) != true {
		// response frame, there is not method-name
		return ""
	}
	serializeId := getSerializeId(flag)
	ret := unSerialize(serializeId, data[DUBBO_HEADER_LEN:], false)
	methodName := ""
	if ret != nil {
		methodName = ret.methodName
	}
	return methodName
}

func dubboGetMeta(data []byte) map[string]string {
	//return "dubboMeta"
	retMap := make(map[string]string)
	rslt, bodyLen := isValidDubboData(data)
	if rslt == false || bodyLen <= 0 {
		return nil
	}

	flag := data[DUBBO_FLAG_IDX]
	if getEventPing(flag) {
		// heart-beat frame, there is not method-name
		retMap["x-protocol-heartbeat"] = XPROTOCOL_PLUGIN_DUBBO
		return retMap
	}
	if isReqFrame(flag) != true {
		status := data[DUBBO_STATUS_IDX]
		retMap["x-mosn-xprotocol-resp-status"] = strconv.Itoa(int(status))

		// TODO: support version under v2.7.1
		decoder := hessian.NewDecoder(data[DUBBO_HEADER_LEN:])
		field, err := decoder.Decode()
		if err != nil {
			fmt.Printf("Decode resWithException fail, err=%v\n", err)
		}
		resWithException, ok := field.(int32)
		if !ok {
			fmt.Printf("Decode resWithException fail, illegal type\n")
		} else {
			if resWithException == RESPONSE_WITH_EXCEPTION || resWithException == RESPONSE_WITH_EXCEPTION_WITH_ATTACHMENTS {
				retMap["x-mosn-xprotocol-resp-is-exception"] = "true"
			}
		}

		return retMap
	}
	serializeId := getSerializeId(flag)
	ret := unSerialize(serializeId, data[DUBBO_HEADER_LEN:], true)
	retMap["serviceName"] = ret.serviceName
	retMap["dubboVersion"] = ret.dubboVersion
	retMap["methodName"] = ret.methodName
	retMap["path"] = ret.path
	retMap["version"] = ret.version

	if ret.attachments != nil {
		for k, v := range ret.attachments {
			retMap[k] = v
		}
	}

	return retMap
}

func ToMapStringString(origin map[interface{}]interface{}) map[string]string {
	dest := make(map[string]string)
	for k, v := range origin {
		if kv, ok := k.(string); ok {
			if vv, ok := v.(string); ok {
				dest[kv] = vv
			}
		}
	}
	return dest
}
