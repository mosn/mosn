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

package subprotocol

import (
	"fmt"

	"github.com/AlexStocks/dubbogo/codec/hessian"
)

func init() {
	serviceNameFunc = dubboGetServiceName
	methodNameFunc = dubboGetMethodName
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
	serviceName string
	methodName  string
}

func unSerialize(serializeId int, data []byte) *dubboAttr {
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

	// dubbo version + path + version + method

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
	ret := unSerialize(serializeId, data[DUBBO_HEADER_LEN:])
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
	ret := unSerialize(serializeId, data[DUBBO_HEADER_LEN:])
	methodName := ""
	if ret != nil {
		methodName = ret.methodName
	}
	return methodName
}
