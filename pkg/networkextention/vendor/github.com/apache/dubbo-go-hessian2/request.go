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

package hessian

import (
	"encoding/binary"
	"reflect"
	"strconv"
	"strings"
	"time"
)

import (
	perrors "github.com/pkg/errors"
)

/////////////////////////////////////////
// dubbo
/////////////////////////////////////////

func getArgType(v interface{}) string {
	if v == nil {
		return "V"
	}

	switch v.(type) {
	// Serialized tags for base types
	case nil:
		return "V"
	case bool:
		return "Z"
	case []bool:
		return "[Z"
	case byte:
		return "B"
	case []byte:
		return "[B"
	case int8:
		return "B"
	case []int8:
		return "[B"
	case int16:
		return "S"
	case []int16:
		return "[S"
	case uint16: // Equivalent to Char of Java
		return "C"
	case []uint16:
		return "[C"
	// case rune:
	//	return "C"
	case int:
		return "J"
	case []int:
		return "[J"
	case int32:
		return "I"
	case []int32:
		return "[I"
	case int64:
		return "J"
	case []int64:
		return "[J"
	case time.Time:
		return "java.util.Date"
	case []time.Time:
		return "[Ljava.util.Date"
	case float32:
		return "F"
	case []float32:
		return "[F"
	case float64:
		return "D"
	case []float64:
		return "[D"
	case string:
		return "java.lang.String"
	case []string:
		return "[Ljava.lang.String;"
	case []Object:
		return "[Ljava.lang.Object;"
	case map[interface{}]interface{}:
		// return  "java.util.HashMap"
		return "java.util.Map"

	//  Serialized tags for complex types
	default:
		t := reflect.TypeOf(v)
		if reflect.Ptr == t.Kind() {
			t = reflect.TypeOf(reflect.ValueOf(v).Elem())
		}
		switch t.Kind() {
		case reflect.Struct:
			return "java.lang.Object"
		case reflect.Slice, reflect.Array:
			if t.Elem().Kind() == reflect.Struct {
				return "[Ljava.lang.Object;"
			}
			// return "java.util.ArrayList"
			return "java.util.List"
		case reflect.Map: // Enter here, map may be map[string]int
			return "java.util.Map"
		default:
			return ""
		}
	}

	// unreachable
	// return "java.lang.RuntimeException"
}

func getArgsTypeList(args []interface{}) (string, error) {
	var (
		typ   string
		types string
	)

	for i := range args {
		typ = getArgType(args[i])
		if typ == "" {
			return types, perrors.Errorf("cat not get arg %#v type", args[i])
		}
		if !strings.Contains(typ, ".") {
			types += typ
		} else if strings.Index(typ, "[") == 0 {
			types += strings.Replace(typ, ".", "/", -1)
		} else {
			// java.util.List -> Ljava/util/List;
			types += "L" + strings.Replace(typ, ".", "/", -1) + ";"
		}
	}

	return types, nil
}

type Request struct {
	Params      interface{}
	Attachments map[string]string
}

// NewRequest create a new Request
func NewRequest(params interface{}, atta map[string]string) *Request {
	if atta == nil {
		atta = make(map[string]string)
	}
	return &Request{
		Params:      params,
		Attachments: atta,
	}
}

func EnsureRequest(body interface{}) *Request {
	if req, ok := body.(*Request); ok {
		return req
	}
	return NewRequest(body, nil)
}

func packRequest(service Service, header DubboHeader, req interface{}) ([]byte, error) {
	var (
		err       error
		types     string
		byteArray []byte
		pkgLen    int
	)

	request := EnsureRequest(req)

	args, ok := request.Params.([]interface{})
	if !ok {
		return nil, perrors.Errorf("@params is not of type: []interface{}")
	}

	hb := header.Type == PackageHeartbeat

	//////////////////////////////////////////
	// byteArray
	//////////////////////////////////////////
	// magic
	switch header.Type {
	case PackageHeartbeat:
		byteArray = append(byteArray, DubboRequestHeartbeatHeader[:]...)
	case PackageRequest_TwoWay:
		byteArray = append(byteArray, DubboRequestHeaderBytesTwoWay[:]...)
	default:
		byteArray = append(byteArray, DubboRequestHeaderBytes[:]...)
	}

	// serialization id, two way flag, event, request/response flag
	// SerialID is id of serialization approach in java dubbo
	byteArray[2] |= header.SerialID & SERIAL_MASK
	// request id
	binary.BigEndian.PutUint64(byteArray[4:], uint64(header.ID))

	encoder := NewEncoder()
	encoder.Append(byteArray[:HEADER_LENGTH])

	//////////////////////////////////////////
	// body
	//////////////////////////////////////////
	if hb {
		encoder.Encode(nil)
		goto END
	}

	// dubbo version + path + version + method
	encoder.Encode(DEFAULT_DUBBO_PROTOCOL_VERSION)
	encoder.Encode(service.Path)
	encoder.Encode(service.Version)
	encoder.Encode(service.Method)

	// args = args type list + args value list
	if types, err = getArgsTypeList(args); err != nil {
		return nil, perrors.Wrapf(err, " PackRequest(args:%+v)", args)
	}
	encoder.Encode(types)
	for _, v := range args {
		encoder.Encode(v)
	}

	request.Attachments[PATH_KEY] = service.Path
	request.Attachments[VERSION_KEY] = service.Version
	if len(service.Group) > 0 {
		request.Attachments[GROUP_KEY] = service.Group
	}
	if len(service.Interface) > 0 {
		request.Attachments[INTERFACE_KEY] = service.Interface
	}
	if service.Timeout != 0 {
		request.Attachments[TIMEOUT_KEY] = strconv.Itoa(int(service.Timeout / time.Millisecond))
	}

	encoder.Encode(request.Attachments)

END:
	byteArray = encoder.Buffer()
	pkgLen = len(byteArray)
	if pkgLen > int(DEFAULT_LEN) { // 8M
		return nil, perrors.Errorf("Data length %d too large, max payload %d", pkgLen, DEFAULT_LEN)
	}
	// byteArray{body length}
	binary.BigEndian.PutUint32(byteArray[12:], uint32(pkgLen-HEADER_LENGTH))
	return byteArray, nil
}

// hessian decode request body
func unpackRequestBody(decoder *Decoder, reqObj interface{}) error {

	if decoder == nil {
		return perrors.Errorf("@decoder is nil")
	}

	req, ok := reqObj.([]interface{})
	if !ok {
		return perrors.Errorf("@reqObj is not of type: []interface{}")
	}
	if len(req) < 7 {
		return perrors.New("length of @reqObj should  be 7")
	}

	var (
		err                                                     error
		dubboVersion, target, serviceVersion, method, argsTypes interface{}
		args                                                    []interface{}
	)

	dubboVersion, err = decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}
	req[0] = dubboVersion

	target, err = decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}
	req[1] = target

	serviceVersion, err = decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}
	req[2] = serviceVersion

	method, err = decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}
	req[3] = method

	argsTypes, err = decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}
	req[4] = argsTypes

	ats := DescRegex.FindAllString(argsTypes.(string), -1)
	var arg interface{}
	for i := 0; i < len(ats); i++ {
		arg, err = decoder.Decode()
		if err != nil {
			return perrors.WithStack(err)
		}
		args = append(args, arg)
	}
	req[5] = args

	attachments, err := decoder.Decode()
	if err != nil {
		return perrors.WithStack(err)
	}
	if v, ok := attachments.(map[interface{}]interface{}); ok {
		v[DUBBO_VERSION_KEY] = dubboVersion
		req[6] = ToMapStringString(v)
		return nil
	}

	return perrors.Errorf("get wrong attachments: %+v", attachments)
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
