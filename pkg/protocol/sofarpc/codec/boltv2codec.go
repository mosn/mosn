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

package codec

import (
	"context"
	"encoding/binary"
	"errors"
	"time"

	"fmt"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// types.Encoder & types.Decoder

type boltV2Codec struct{}

func (c *boltV2Codec) EncodeHeaders(ctx context.Context, headers types.HeaderMap) (types.IoBuffer, error) {
	switch cmd := headers.(type) {
	case *sofarpc.BoltV2RequestCommand:
		return c.encodeRequestCommand(ctx, cmd)
	case *sofarpc.BoltV2ResponseCommand:
		return c.encodeResponseCommand(ctx, cmd)
	default:
		errMsg := sofarpc.InvalidCommandType
		err := errors.New(errMsg)
		log.ByContext(ctx).Errorf("boltV2" + errMsg)
		return nil, err
	}
}

func (c *boltV2Codec) EncodeData(ctx context.Context, data types.IoBuffer) types.IoBuffer {
	return data
}

func (c *boltV2Codec) EncodeTrailers(ctx context.Context, trailers types.HeaderMap) types.IoBuffer {
	return nil
}

func (c *boltV2Codec) encodeRequestCommand(ctx context.Context, cmd *sofarpc.BoltV2RequestCommand) (types.IoBuffer, error) {
	return c.doEncodeRequestCommand(ctx, cmd), nil
}

func (c *boltV2Codec) encodeResponseCommand(ctx context.Context, cmd *sofarpc.BoltV2ResponseCommand) (types.IoBuffer, error) {
	return c.doEncodeResponseCommand(ctx, cmd), nil
}

func (c *boltV2Codec) doEncodeRequestCommand(ctx context.Context, cmd *sofarpc.BoltV2RequestCommand) types.IoBuffer {
	// serialize classname and header
	if cmd.RequestClass != "" {
		cmd.ClassName, _ = serialize.Instance.Serialize(cmd.RequestClass)
		cmd.ClassLen = int16(len(cmd.ClassName))
	}

	if cmd.RequestHeader != nil {
		cmd.HeaderMap, _ = serialize.Instance.Serialize(cmd.RequestHeader)
		cmd.HeaderLen = int16(len(cmd.HeaderMap))
	}

	var b [4]byte
	// todo: reuse bytes @boqin
	//data := make([]byte, 22, defaultTmpBufferSize)
	size := 22 + int(cmd.ClassLen) + len(cmd.HeaderMap)
	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	buf := protocolCtx.GetReqHeader(size)

	b[0] = cmd.Protocol
	buf.Write(b[0:1])
	b[0] = cmd.Version1
	buf.Write(b[0:1])
	b[0] = cmd.CmdType
	buf.Write(b[0:1])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.CmdCode))
	buf.Write(b[0:2])

	b[0] = cmd.Version
	buf.Write(b[0:1])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ReqID))
	buf.Write(b[0:4])

	b[0] = cmd.CodecPro
	buf.Write(b[0:1])
	b[0] = cmd.SwitchCode
	buf.Write(b[0:1])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.Timeout))
	buf.Write(b[0:4])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.ClassLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.HeaderLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ContentLen))
	buf.Write(b[0:4])

	if cmd.ClassLen > 0 {
		buf.Write(cmd.ClassName)
	}

	if cmd.HeaderLen > 0 {
		buf.Write(cmd.HeaderMap)
	}

	return buf
}

func (c *boltV2Codec) doEncodeResponseCommand(ctx context.Context, cmd *sofarpc.BoltV2ResponseCommand) types.IoBuffer {
	// serialize classname and header
	if cmd.ResponseClass != "" {
		cmd.ClassName, _ = serialize.Instance.Serialize(cmd.ResponseClass)
		cmd.ClassLen = int16(len(cmd.ClassName))
	}

	if cmd.ResponseHeader != nil {
		cmd.HeaderMap, _ = serialize.Instance.Serialize(cmd.ResponseHeader)
		cmd.HeaderLen = int16(len(cmd.HeaderMap))
	}

	var b [4]byte
	// todo: reuse bytes @boqin
	size := 20 + int(cmd.ClassLen) + len(cmd.HeaderMap)
	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	buf := protocolCtx.GetRspHeader(size)

	b[0] = cmd.Protocol
	buf.Write(b[0:1])
	b[0] = cmd.Version1
	buf.Write(b[0:1])
	b[0] = cmd.CmdType
	buf.Write(b[0:1])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.CmdCode))
	buf.Write(b[0:2])

	if cmd.CmdCode == sofarpc.HEARTBEAT {
		log.ByContext(ctx).Debugf("Build HeartBeat Response")
	}

	b[0] = cmd.Version
	buf.Write(b[0:1])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ReqID))
	buf.Write(b[0:4])

	b[0] = cmd.CodecPro
	buf.Write(b[0:1])
	b[0] = cmd.SwitchCode
	buf.Write(b[0:1])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.ResponseStatus))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.ClassLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.HeaderLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ContentLen))
	buf.Write(b[0:4])

	if cmd.ClassLen > 0 {
		buf.Write(cmd.ClassName)
	}

	if cmd.HeaderLen > 0 {
		buf.Write(cmd.HeaderMap)
	}

	return buf
}

func (c *boltV2Codec) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	readableBytes := data.Len()
	read := 0
	var cmd interface{}
	logger := log.ByContext(ctx)

	if readableBytes >= sofarpc.LESS_LEN_V2 {
		bytes := data.Bytes()

		ver1 := bytes[1]

		dataType := bytes[2]

		//1. request
		if dataType == sofarpc.REQUEST || dataType == sofarpc.REQUEST_ONEWAY {
			if readableBytes >= sofarpc.REQUEST_HEADER_LEN_V2 {

				cmdCode := binary.BigEndian.Uint16(bytes[3:5])
				ver2 := bytes[5]
				requestID := binary.BigEndian.Uint32(bytes[6:10])
				codec := bytes[10]

				switchCode := bytes[11]

				timeout := binary.BigEndian.Uint32(bytes[12:16])
				classLen := binary.BigEndian.Uint16(bytes[16:18])
				headerLen := binary.BigEndian.Uint16(bytes[18:20])
				contentLen := binary.BigEndian.Uint32(bytes[20:24])

				read = sofarpc.REQUEST_HEADER_LEN_V2
				var class, header, content []byte

				if readableBytes >= read+int(classLen)+int(headerLen)+int(contentLen) {
					if classLen > 0 {
						class = bytes[read : read+int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytes[read : read+int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytes[read : read+int(contentLen)]
						read += int(contentLen)
					}
					data.Drain(read)
				} else { // not enough data
					logger.Debugf("[BOLTV2 Decoder]no enough data for fully decode")
					return cmd, nil
				}

				request := &sofarpc.BoltV2RequestCommand{
					BoltRequestCommand: sofarpc.BoltRequestCommand{
						Protocol:   sofarpc.PROTOCOL_CODE_V1,
						CmdType:    dataType,
						CmdCode:    int16(cmdCode),
						Version:    ver2,
						ReqID:      requestID,
						CodecPro:   codec,
						Timeout:    int(timeout),
						ClassLen:   int16(classLen),
						HeaderLen:  int16(headerLen),
						ContentLen: int(contentLen),
						ClassName:  class,
						HeaderMap:  header,
						Content:    content,
					},
					Version1:   ver1,
					SwitchCode: switchCode,
				}

				logger.Debugf("[Decoder]bolt v2 decode request:%+v", request)

				cmd = request
			}
		} else if dataType == sofarpc.RESPONSE {
			//2. response
			if readableBytes >= sofarpc.RESPONSE_HEADER_LEN_V2 {

				cmdCode := binary.BigEndian.Uint16(bytes[3:5])
				ver2 := bytes[5]
				requestID := binary.BigEndian.Uint32(bytes[6:10])
				codec := bytes[10]
				switchCode := bytes[11]

				status := binary.BigEndian.Uint16(bytes[12:14])
				classLen := binary.BigEndian.Uint16(bytes[14:16])
				headerLen := binary.BigEndian.Uint16(bytes[16:18])
				contentLen := binary.BigEndian.Uint32(bytes[18:22])

				read = sofarpc.RESPONSE_HEADER_LEN_V2
				var class, header, content []byte

				if readableBytes >= read+int(classLen)+int(headerLen)+int(contentLen) {
					if classLen > 0 {
						class = bytes[read : read+int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytes[read : read+int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytes[read : read+int(contentLen)]
						read += int(contentLen)
					}
				} else { // not enough data
					logger.Debugf("[BOLTBV2 Decoder]no enough data for fully decode")
					return cmd, nil
				}

				response := &sofarpc.BoltV2ResponseCommand{
					BoltResponseCommand: sofarpc.BoltResponseCommand{
						Protocol:           sofarpc.PROTOCOL_CODE_V1,
						CmdType:            dataType,
						CmdCode:            int16(cmdCode),
						Version:            ver2,
						ReqID:              requestID,
						CodecPro:           codec,
						ResponseStatus:     int16(status),
						ClassLen:           int16(classLen),
						HeaderLen:          int16(headerLen),
						ContentLen:         int(contentLen),
						ClassName:          class,
						HeaderMap:          header,
						Content:            content,
						ResponseTimeMillis: time.Now().UnixNano() / int64(time.Millisecond),
					},
					Version1:   ver1,
					SwitchCode: switchCode,
				}

				logger.Debugf("[Decoder]bolt v2 decode response:%+v\n", response)
				cmd = response
			}
		} else {
			// 3. unknown type error
			return nil, fmt.Errorf("Decode Error, type = %s, value = %d", sofarpc.UnKnownReqtype, dataType)

		}
	}

	return cmd, nil
}

func (c *boltV2Codec) insertToBytes(slice []byte, idx int, b byte) {
	slice = append(slice, 0)
	copy(slice[idx+1:], slice[idx:])
	slice[idx] = b
}
