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

package v2

import (
	"context"
	"encoding/binary"
	"time"

	"fmt"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/bolt"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
)

var (
	Codec = &codec{}
)

func init() {
	sofarpc.Register(bolt.PROTOCOL_CODE_V2, Codec, Codec)
}

// ~~ types.Encoder
// ~~ types.Decoder
type codec struct{}

func (c *codec) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	switch cmd := model.(type) {
	case *bolt.RequestV2:
		return encodeRequest(ctx, cmd)
	case *bolt.ResponseV2:
		return encodeResponse(ctx, cmd)
	default:
		log.ByContext(ctx).Errorf("unknown model : %+v", model)
		return nil, rpc.ErrUnknownType
	}
}

func encodeRequest(ctx context.Context, cmd *bolt.RequestV2) (types.IoBuffer, error) {
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
	size := bolt.REQUEST_HEADER_LEN_V2 + int(cmd.ClassLen) + len(cmd.HeaderMap)
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

	b[0] = cmd.Codec
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

	return buf, nil
}

func encodeResponse(ctx context.Context, cmd *bolt.ResponseV2) (types.IoBuffer, error) {
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
	size := bolt.RESPONSE_HEADER_LEN_V2 + int(cmd.ClassLen) + len(cmd.HeaderMap)
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

	b[0] = cmd.Version
	buf.Write(b[0:1])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ReqID))
	buf.Write(b[0:4])

	b[0] = cmd.Codec
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

	return buf, nil
}

func (c *codec) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	readableBytes := data.Len()
	read := 0
	var cmd interface{}
	logger := log.ByContext(ctx)

	if readableBytes >= bolt.LESS_LEN_V2 {
		bytes := data.Bytes()

		ver1 := bytes[1]

		cmdType := bytes[2]

		//1. request
		if cmdType == bolt.REQUEST || cmdType == bolt.REQUEST_ONEWAY {
			if readableBytes >= bolt.REQUEST_HEADER_LEN_V2 {

				cmdCode := binary.BigEndian.Uint16(bytes[3:5])
				ver2 := bytes[5]
				requestID := binary.BigEndian.Uint32(bytes[6:10])
				codec := bytes[10]

				switchCode := bytes[11]

				timeout := binary.BigEndian.Uint32(bytes[12:16])
				classLen := binary.BigEndian.Uint16(bytes[16:18])
				headerLen := binary.BigEndian.Uint16(bytes[18:20])
				contentLen := binary.BigEndian.Uint32(bytes[20:24])

				read = bolt.REQUEST_HEADER_LEN_V2
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

				request := &bolt.RequestV2{
					Request: bolt.Request{
						bolt.PROTOCOL_CODE_V2,
						cmdType,
						int16(cmdCode),
						ver2,
						requestID,
						codec,
						int(timeout),
						int16(classLen),
						int16(headerLen),
						int(contentLen),
						class,
						header,
						content,
						"",
						nil,
					},
					Version1:   ver1,
					SwitchCode: switchCode,
				}

				bolt.DeserializeRequest(ctx, &request.Request)

				logger.Debugf("[Decoder]bolt v2 decode request:%+v", request)

				cmd = request
			}
		} else if cmdType == bolt.RESPONSE {
			//2. bolt.ResponseV2
			if readableBytes >= bolt.RESPONSE_HEADER_LEN_V2 {

				cmdCode := binary.BigEndian.Uint16(bytes[3:5])
				ver2 := bytes[5]
				requestID := binary.BigEndian.Uint32(bytes[6:10])
				codec := bytes[10]
				switchCode := bytes[11]

				status := binary.BigEndian.Uint16(bytes[12:14])
				classLen := binary.BigEndian.Uint16(bytes[14:16])
				headerLen := binary.BigEndian.Uint16(bytes[16:18])
				contentLen := binary.BigEndian.Uint32(bytes[18:22])

				read = bolt.RESPONSE_HEADER_LEN_V2
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

				response := &bolt.ResponseV2{
					Response: bolt.Response{
						bolt.PROTOCOL_CODE_V2,
						cmdType,
						int16(cmdCode),
						ver2,
						requestID,
						codec,
						int16(status),
						int16(classLen),
						int16(headerLen),
						int(contentLen),
						class,
						header,
						content,
						"",
						nil,
						time.Now().UnixNano() / int64(time.Millisecond),
					},
					Version1:   ver1,
					SwitchCode: switchCode,
				}

				bolt.DeserializeResponse(ctx, &response.Response)

				logger.Debugf("[Decoder]bolt v2 decode bolt.ResponseV2:%+v\n", response)
				cmd = response
			}
		} else {
			// 3. unknown type error
			return nil, fmt.Errorf("Decode Error, type = %s, value = %d", sofarpc.UnKnownCmdType, cmdType)

		}
	}

	return cmd, nil
}

func insertToBytes(slice []byte, idx int, b byte) {
	slice = append(slice, 0)
	copy(slice[idx+1:], slice[idx:])
	slice[idx] = b
}
