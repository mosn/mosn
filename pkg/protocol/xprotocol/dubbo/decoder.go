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
	"runtime/debug"
	"sync"

	hessian "github.com/apache/dubbo-go-hessian2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

// Decoder is heavy and caches to improve performance.
// Avoid allocating 4k memory every time you create an object
var (
	decodePoolCheap = &sync.Pool{
		New: func() interface{} {
			return hessian.NewCheapDecoderWithSkip([]byte{})
		},
	}
	decodePool = &sync.Pool{
		New: func() interface{} {
			return hessian.NewDecoderWithSkip([]byte{})
		},
	}
)

func decodeFrame(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	// convert data to dubbo frame
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
	reqIDRaw := dataBytes[IdIdx:(IdIdx + IdLen)]
	frame.Id = binary.BigEndian.Uint64(reqIDRaw)
	// decode data length
	frame.DataLen = binary.BigEndian.Uint32(dataBytes[DataLenIdx:(DataLenIdx + DataLenSize)])

	// decode event
	frame.IsEvent = (frame.Flag & (1 << 5)) != 0

	// decode twoway
	frame.IsTwoWay = (frame.Flag & (1 << 6)) != 0

	// decode direction
	directionBool := frame.Flag & (1 << 7)
	if directionBool != 0 {
		frame.Direction = EventRequest
	} else {
		frame.Direction = EventResponse
	}
	// decode serializationId
	frame.SerializationId = int(frame.Flag & 0x1f)

	frameLen := HeaderLen + frame.DataLen
	// decode payload
	body := make([]byte, frameLen)
	copy(body, dataBytes[:frameLen])
	frame.payload = body[HeaderLen:]
	frame.content = buffer.NewIoBufferBytes(frame.payload)

	// not heartbeat & is request
	if !frame.IsEvent && frame.Direction == EventRequest {
		// service aware
		meta, err := getServiceAwareMeta(ctx, frame)
		if err != nil {
			return nil, err
		}
		for k, v := range meta {
			frame.Set(k, v)
		}
	}

	frame.rawData = body
	frame.data = buffer.NewIoBufferBytes(frame.rawData)
	switch frame.Direction {
	case EventRequest:
		// notice: read-only!!! do not modify the raw data!!!
		variable.Set(ctx, types.VarRequestRawData, frame.rawData)
	case EventResponse:
		// notice: read-only!!! do not modify the raw data!!!
		variable.Set(ctx, types.VarResponseRawData, frame.rawData)
	}

	data.Drain(int(frameLen))
	return frame, nil
}

func getServiceAwareMeta(ctx context.Context, frame *Frame) (meta map[string]string, err error) {
	meta = make(map[string]string, 8)
	if frame.SerializationId != 2 {
		// not hessian , do not support
		return meta, fmt.Errorf("[xprotocol][dubbo] not hessian,do not support")
	}

	// Recycle decode
	var (
		decoder  *hessian.Decoder
		listener interface{}
	)

	if ctx != nil {
		listener, _ = variable.Get(ctx, types.VariableListenerName)
	}
	if listener == IngressDubbo || listener == EgressDubbo {
		decoder = decodePool.Get().(*hessian.Decoder)
		defer decodePool.Put(decoder)
	} else {
		decoder = decodePoolCheap.Get().(*hessian.Decoder)
		defer decodePoolCheap.Put(decoder)
	}
	decoder.Reset(frame.payload[:])

	var (
		field            interface{}
		ok               bool
		frameworkVersion string
		path             string
		version          string
		method           string
	)

	// framework version + path + version + method
	// get service name
	field, err = decoder.Decode()
	if err != nil {
		return meta, fmt.Errorf("[xprotocol][dubbo] decode framework version fail: %v", err)
	}
	frameworkVersion, ok = field.(string)
	if !ok {
		return meta, fmt.Errorf("[xprotocol][dubbo] decode framework version {%v} type error", field)
	}
	meta[FrameworkVersionNameHeader] = frameworkVersion

	field, err = decoder.Decode()
	if err != nil {
		return meta, fmt.Errorf("[xprotocol][dubbo] decode service path fail: %v", err)
	}
	path, ok = field.(string)
	if !ok {
		return meta, fmt.Errorf("[xprotocol][dubbo] service path {%v} type error", field)
	}
	meta[ServiceNameHeader] = path

	// get method name
	field, err = decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("[xprotocol][dubbo] decode method version fail: %v", err)
	}
	// callback maybe return nil
	if field != nil {
		version, ok = field.(string)
		if !ok {
			return nil, fmt.Errorf("[xprotocol][dubbo] method version {%v} type fail", field)
		}
	}
	meta[VersionNameHeader] = version

	field, err = decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("[xprotocol][dubbo] decode method fail: %v", err)
	}
	method, ok = field.(string)
	if !ok {
		return nil, fmt.Errorf("[xprotocol][dubbo] method {%v} type error", field)
	}
	meta[MethodNameHeader] = method

	if ctx != nil {
		var (
			node    *Node
			matched bool
		)

		// for better performance.
		// If the ingress scenario is not using group,
		// we can skip parsing attachment to improve performance
		if listener == IngressDubbo {
			if node, matched = DubboPubMetadata.Find(path, version); matched {
				meta[ServiceNameHeader] = node.Service
				meta[GroupNameHeader] = node.Group
			}
		} else if listener == EgressDubbo {
			// for better performance.
			// If the egress scenario is not using group,
			// we can skip parsing attachment to improve performance
			if node, matched = DubboSubMetadata.Find(path, version); matched {
				meta[ServiceNameHeader] = node.Service
				meta[GroupNameHeader] = node.Group
			}
		}

		// decode the attachment to get the real service and group parameters
		if !matched && (listener == EgressDubbo || listener == IngressDubbo) {

			// decode arguments maybe panic, when dubbo payload have complex struct
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("decode arguments error :%v\n%s", r, debug.Stack())
				}
			}()

			field, err = decoder.Decode()
			if err != nil {
				return nil, fmt.Errorf("[xprotocol][dubbo] decode dubbo argument types error: %v", err)
			}

			arguments := getArgumentCount(field.(string))
			// we must skip all method arguments.
			for i := 0; i < arguments; i++ {
				_, err = decoder.Decode()
				if err != nil {
					return nil, fmt.Errorf("[xprotocol][dubbo] decode dubbo argument error: %v", err)
				}
			}

			field, err = decoder.Decode()
			if err != nil {
				return nil, fmt.Errorf("[xprotocol][dubbo] decode dubbo attachments error: %v", err)
			}

			if field != nil {
				if origin, ok := field.(map[interface{}]interface{}); ok {
					// we loop all attachments and check element type,
					// we should only read string types.
					for k, v := range origin {
						if key, ok := k.(string); ok {
							if val, ok := v.(string); ok {
								meta[key] = val
								// we should use interface value,
								// convenient for us to do service discovery.
								if key == InterfaceNameHeader {
									meta[ServiceNameHeader] = val
								}
							}
						}
					}
				}
			}
		}
	}

	return meta, nil
}

//	more unit test:
//
// https://github.com/zonghaishang/dubbo/commit/e0fd702825a274379fb609229bdb06ca0586122e
func getArgumentCount(desc string) int {
	len := len(desc)
	if len == 0 {
		return 0
	}

	var args, next = 0, false
	for _, ch := range desc {

		// is array ?
		if ch == '[' {
			continue
		}

		// is object ?
		if next && ch != ';' {
			continue
		}

		switch ch {
		case 'V', // void
			'Z', // boolean
			'B', // byte
			'C', // char
			'D', // double
			'F', // float
			'I', // int
			'J', // long
			'S': // short
			args++
		default:
			// we found object
			if ch == 'L' {
				args++
				next = true
				// end of object ?
			} else if ch == ';' {
				next = false
			}
		}

	}
	return args
}
