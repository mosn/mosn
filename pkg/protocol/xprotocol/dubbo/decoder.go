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
	"context"
	"encoding/binary"
	"fmt"
	"time"

	hessian "github.com/apache/dubbo-go-hessian2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func decodeFrame(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	// convert data to duboo frame
	dataBytes := data.Bytes()
	frame := &Frame{
		Header: Header{
			CommonHeader: protocol.CommonHeader{},
		},
	}
	// decode magic
	frame.Magic = dataBytes[MagicIdx:FlagIdx]
	// decode flag
	frame.Flag = dataBytes[FlagIdx]
	// decode status
	frame.Status = dataBytes[StatusIdx]
	// decode request id
	reqIdRaw := dataBytes[IdIdx:(IdIdx + IdLen)]
	frame.Id = binary.BigEndian.Uint64(reqIdRaw)
	// decode data length
	frame.DataLen = binary.BigEndian.Uint32(dataBytes[DataLenIdx:(DataLenIdx + DataLenSize)])

	// decode event
	eventBool := frame.Flag & (1 << 5)
	if eventBool != 0 {
		frame.Event = 1
	} else {
		frame.Event = 0
	}
	// decode twoway
	twoWayBool := frame.Flag & (1 << 6)
	if twoWayBool != 0 {
		frame.TwoWay = 1
	} else {
		frame.TwoWay = 0
	}
	// decode direction
	directionBool := frame.Flag & (1 << 7)
	if directionBool != 0 {
		frame.Direction = EventRequest
	} else {
		frame.Direction = EventResponse
	}
	// decode serializationId
	frame.SerializationId = int(frame.Flag & 0x1f)

	// decode payload
	payload := make([]byte, frame.DataLen)
	copy(payload, dataBytes[HeaderLen:HeaderLen+frame.DataLen])
	frame.payload = payload
	frame.content = buffer.NewIoBufferBytes(frame.payload)

	// not heartbeat & is request
	if frame.Event != 1 && frame.Direction == 1 {
		// service aware
		meta, attachment, err := getServiceAwareMeta(frame)
		if err != nil {
			return nil, err
		}
		for k, v := range meta {
			frame.Set(k, v)
		}
		frame.attachment = attachment
		for k, v := range attachment {
			frame.Set(k, v)
		}
	}

	frameLen := HeaderLen + int(frame.DataLen)
	frame.rawData = dataBytes[:frameLen]
	frame.data = buffer.NewIoBufferBytes(frame.rawData)
	data.Drain(frameLen)
	return frame, nil
}

func getServiceAwareMeta(frame *Frame) (map[string]string, map[string]string, error) {
	meta := make(map[string]string)
	if frame.SerializationId != 2 {
		// not hessian , do not support
		return meta, nil, fmt.Errorf("[xprotocol][dubbo] not hessian,do not support")
	}
	decoder := hessian.NewDecoderWithSkip(frame.payload[:])

	var (
		err                                                          error
		ok                                                           bool
		str                                                          string
		dubboVersion, servicePath, serviceVersion, method, argsTypes interface{}
		args                                                         []interface{}
	)

	// dubbo version + path + version + method
	// get service name
	dubboVersion, err = decoder.Decode()
	if err != nil {
		return meta, nil, fmt.Errorf("[xprotocol][dubbo] decode version fail: %s", err)
	}
	str, ok = dubboVersion.(string)
	if !ok {
		return meta, nil, fmt.Errorf("[xprotocol][dubbo] service name version type error")
	}

	servicePath, err = decoder.Decode()
	if err != nil {
		return meta, nil, fmt.Errorf("[xprotocol][dubbo] decode service fail: %s", err)
	}
	str, ok = servicePath.(string)
	if !ok {
		return meta, nil, fmt.Errorf("[xprotocol][dubbo] service type error")
	}
	meta[ServiceNameHeader] = str

	// get method name
	serviceVersion, err = decoder.Decode()
	if err != nil {
		return nil, nil, fmt.Errorf("[xprotocol][dubbo] decode method version fail: %s", err)
	}
	str, ok = serviceVersion.(string)
	if !ok {
		return nil, nil, fmt.Errorf("[xprotocol][dubbo] method version type fail")
	}

	method, err = decoder.Decode()
	if err != nil {
		return nil, nil, fmt.Errorf("[xprotocol][dubbo] decode method fail: %s", err)
	}
	str, ok = method.(string)
	if !ok {
		return nil, nil, fmt.Errorf("[xprotocol][dubbo] method type error")
	}
	meta[MethodNameHeader] = str

	argsTypes, err = decoder.Decode()
	if err != nil {
		return nil, nil, fmt.Errorf("[xprotocol][dubbo] decode argsTypes fail: %s", err)
	}

	ats := hessian.DescRegex.FindAllString(argsTypes.(string), -1)
	args = []interface{}{}
	var arg interface{}
	for i := 0; i < len(ats); i++ {
		arg, err = decoder.Decode()
		if err != nil {
			return nil, nil, fmt.Errorf("[xprotocol][dubbo] decode args fail: %s", err)
		}
		args = append(args, arg)
	}

	attachmentObj, err := decoder.Decode()
	if err != nil {
		return nil, nil, fmt.Errorf("[xprotocol][dubbo] decode attachment fail: %s", err)
	}
	attachItf := attachmentObj.(map[interface{}]interface{})

	encode := hessian.NewEncoder()
	encode.Encode(attachItf)
	attachmentBytes := encode.Buffer()
	frame.attachmentLen = uint32(len(attachmentBytes))

	attachment := ToMapStringString(attachItf)

	// mock inject
	attachment["mosn-inject"] = time.Now().Format("2006-01-02 15:04:05")

	frame.attachment = attachment

	return meta, attachment, nil
}

func ToMapStringString(origin map[interface{}]interface{}) map[string]string {
	dest := make(map[string]string)
	for k, v := range origin {
		if kv, ok := k.(string); ok {
			if v == nil {
				dest[kv] = ""
				continue
			}

			if vv, ok := v.(string); ok {
				dest[kv] = vv
			}
		}
	}
	return dest
}
