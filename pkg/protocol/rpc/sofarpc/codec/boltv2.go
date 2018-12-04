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
	"fmt"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/buffer"
)

var (
	BoltCodecV2 = &boltCodecV2{}
)

func init() {
	sofarpc.RegisterProtocol(sofarpc.PROTOCOL_CODE_V2, BoltCodecV2, BoltCodecV2, nil)
	sofarpc.RegisterResponseBuilder(sofarpc.PROTOCOL_CODE_V2, BoltCodecV2)
	// the heartbeat processing is same with boltV1
	sofarpc.RegisterHeartbeatBuilder(sofarpc.PROTOCOL_CODE_V2, BoltCodec)
}

// ~~ types.Encoder
// ~~ types.Decoder
type boltCodecV2 struct{}

func (c *boltCodecV2) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	switch cmd := model.(type) {
	case *sofarpc.BoltRequestV2:
		return encodeRequestV2(ctx, cmd)
	case *sofarpc.BoltResponseV2:
		return encodeResponseV2(ctx, cmd)
	default:
		log.ByContext(ctx).Errorf("unknown model : %+v", model)
		return nil, rpc.ErrUnknownType
	}
}

func encodeRequestV2(ctx context.Context, cmd *sofarpc.BoltRequestV2) (types.IoBuffer, error) {
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
	size := sofarpc.REQUEST_HEADER_LEN_V2 + int(cmd.ClassLen) + len(cmd.HeaderMap)
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

func encodeResponseV2(ctx context.Context, cmd *sofarpc.BoltResponseV2) (types.IoBuffer, error) {
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
	size := sofarpc.RESPONSE_HEADER_LEN_V2 + int(cmd.ClassLen) + len(cmd.HeaderMap)
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

func (c *boltCodecV2) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	readableBytes := data.Len()
	read := 0
	var cmd interface{}
	logger := log.ByContext(ctx)

	if readableBytes >= sofarpc.LESS_LEN_V2 {
		bytes := data.Bytes()

		ver1 := bytes[1]

		cmdType := bytes[2]

		//1. request
		if cmdType == sofarpc.REQUEST || cmdType == sofarpc.REQUEST_ONEWAY {
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

				request := &sofarpc.BoltRequestV2{
					BoltRequest: sofarpc.BoltRequest{
						sofarpc.PROTOCOL_CODE_V2,
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
						buffer.NewIoBufferBytes(content),
						"",
						nil,
					},
					Version1:   ver1,
					SwitchCode: switchCode,
				}

				sofarpc.DeserializeBoltRequest(ctx, &request.BoltRequest)

				logger.Debugf("[Decoder]bolt v2 decode request:%+v", request)

				cmd = request
			}
		} else if cmdType == sofarpc.RESPONSE {
			//2. bolt.ResponseV2
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

				response := &sofarpc.BoltResponseV2{
					BoltResponse: sofarpc.BoltResponse{
						sofarpc.PROTOCOL_CODE_V2,
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
						buffer.NewIoBufferBytes(content),
						"",
						nil,
						time.Now().UnixNano() / int64(time.Millisecond),
					},
					Version1:   ver1,
					SwitchCode: switchCode,
				}

				sofarpc.DeserializeBoltResponse(ctx, &response.BoltResponse)

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

// ~ ResponseBuilder
func (c *boltCodecV2) BuildResponse(respStatus int16) sofarpc.SofaRpcCmd {
	return &sofarpc.BoltResponseV2{
		BoltResponse: sofarpc.BoltResponse{
			Protocol:       sofarpc.PROTOCOL_CODE_V2,
			CmdType:        sofarpc.RESPONSE,
			CmdCode:        sofarpc.RPC_RESPONSE,
			Version:        1,
			ReqID:          0,                          // this would be overwrite by stream layer
			Codec:          sofarpc.HESSIAN2_SERIALIZE, //todo: read default codec from config
			ResponseStatus: respStatus,
		},
		Version1:   sofarpc.PROTOCOL_VERSION_2,
		SwitchCode: 0,
	}
}
