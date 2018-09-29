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
	"fmt"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// types.Encoder & types.Decoder
type boltV1Codec struct{}

func (c *boltV1Codec) EncodeHeaders(ctx context.Context, headers types.HeaderMap) (types.IoBuffer, error) {
	switch cmd := headers.(type) {
	case *sofarpc.BoltRequestCommand:
		return c.encodeRequestCommand(ctx, cmd)
	case *sofarpc.BoltResponseCommand:
		return c.encodeResponseCommand(ctx, cmd)
	default:

		errMsg := sofarpc.InvalidCommandType
		err := errors.New(errMsg)
		log.ByContext(ctx).Errorf("boltV1" + errMsg)
		return nil, err
	}
}

func (c *boltV1Codec) EncodeData(ctx context.Context, data types.IoBuffer) types.IoBuffer {
	return data
}

func (c *boltV1Codec) EncodeTrailers(ctx context.Context, trailers types.HeaderMap) types.IoBuffer {
	return nil
}

func (c *boltV1Codec) encodeRequestCommand(ctx context.Context, cmd *sofarpc.BoltRequestCommand) (types.IoBuffer, error) {
	return c.doEncodeRequestCommand(ctx, cmd), nil
}

func (c *boltV1Codec) encodeResponseCommand(ctx context.Context, cmd *sofarpc.BoltResponseCommand) (types.IoBuffer, error) {
	return c.doEncodeResponseCommand(ctx, cmd), nil
}

func (c *boltV1Codec) doEncodeRequestCommand(ctx context.Context, cmd *sofarpc.BoltRequestCommand) types.IoBuffer {
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
	//buf := sofarpc.GetBuffer(context, size)

	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	buf := protocolCtx.GetReqHeader(size)
	//buf := make([]byte, 22, size)

	b[0] = cmd.Protocol
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

	//log.ByContext(ctx).Debugf("BoltV1 ENCODE REQUEST, CmdType = %d, CmdCode = %d, ReqID = %d, Bytes = %d",
	//	cmd.CmdType, cmd.CmdCode, cmd.ReqID, sofarpc.REQUEST_HEADER_LEN_V1 + int(cmd.ClassLen)+int(cmd.HeaderLen)+int(cmd.ContentLen) )

	return buf
}

func (c *boltV1Codec) doEncodeResponseCommand(ctx context.Context, cmd *sofarpc.BoltResponseCommand) types.IoBuffer {
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
	//buf := sofarpc.GetBuffer(context, size)
	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	buf := protocolCtx.GetRspHeader(size)

	b[0] = cmd.Protocol
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

	//log.ByContext(ctx).Debugf("BoltV1 ENCODE RESPONSE,RespStatus = %d, CmdType = %d, CmdCode = %d, ReqID = %d, Bytes = %d",
	//	cmd.ResponseStatus, cmd.CmdType, cmd.CmdCode, cmd.ReqID, sofarpc.RESPONSE_HEADER_LEN_V1 + int(cmd.ClassLen)+int(cmd.HeaderLen)+int(cmd.ContentLen) )

	return buf
}

func (c *boltV1Codec) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	readableBytes := data.Len()
	read := 0
	var cmd interface{}
	logger := log.ByContext(ctx)

	if readableBytes >= sofarpc.LESS_LEN_V1 {
		bytes := data.Bytes()
		dataType := bytes[1]

		//1. request
		if dataType == sofarpc.REQUEST || dataType == sofarpc.REQUEST_ONEWAY {
			if readableBytes >= sofarpc.REQUEST_HEADER_LEN_V1 {

				cmdCode := binary.BigEndian.Uint16(bytes[2:4])

				//if cmdCode == uint16(sofarpc.HEARTBEAT) {
				//	logger.Debugf("BoltV1 DECODE Request: Get Bolt HB Msg")
				//}
				ver2 := bytes[4]
				requestID := binary.BigEndian.Uint32(bytes[5:9])
				codec := bytes[9]
				timeout := binary.BigEndian.Uint32(bytes[10:14])
				classLen := binary.BigEndian.Uint16(bytes[14:16])
				headerLen := binary.BigEndian.Uint16(bytes[16:18])
				contentLen := binary.BigEndian.Uint32(bytes[18:22])

				read = sofarpc.REQUEST_HEADER_LEN_V1
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

					logger.Debugf("BoltV1 DECODE Request, no enough data for fully decode")
					return cmd, nil
				}

				sofabuffers := sofarpc.SofaProtocolBuffersByContext(ctx)
				request := &sofabuffers.BoltReq
				//request := &sofarpc.BoltRequestCommand{}
				request.Protocol = sofarpc.PROTOCOL_CODE_V1
				request.CmdType = dataType
				request.CmdCode = int16(cmdCode)
				request.Version = ver2
				request.ReqID = requestID
				request.CodecPro = codec
				request.Timeout = int(timeout)
				request.ClassLen = int16(classLen)
				request.HeaderLen = int16(headerLen)
				request.ContentLen = int(contentLen)
				request.ClassName = class
				request.HeaderMap = header
				request.Content = content
				cmd = request

				//logger.Debugf("BoltV1 DECODE REQUEST, Protocol = %d, CmdType = %d, CmdCode = %d, ReqID = %d, Bytes = %d",
				//	request.Protocol, request.CmdType, request.CmdCode, request.ReqID, sofarpc.REQUEST_HEADER_LEN_V1 + int(classLen)+int(headerLen)+int(contentLen) )

				/*
					request := sofarpc.BoltRequestCommand{

						sofarpc.PROTOCOL_CODE_V1,
						dataType,
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
						nil,
						nil,
					}
					logger.Debugf("BoltV1 DECODE REQUEST, Protocol = %d, CmdType = %d, CmdCode = %d, ReqID = %d",
						request.Protocol, request.CmdType, request.CmdCode, request.ReqID)
					cmd = &request
				*/

			}
		} else if dataType == sofarpc.RESPONSE {
			//2. response
			if readableBytes >= sofarpc.RESPONSE_HEADER_LEN_V1 {

				cmdCode := binary.BigEndian.Uint16(bytes[2:4])
				ver2 := bytes[4]
				requestID := binary.BigEndian.Uint32(bytes[5:9])
				codec := bytes[9]
				status := binary.BigEndian.Uint16(bytes[10:12])
				classLen := binary.BigEndian.Uint16(bytes[12:14])
				headerLen := binary.BigEndian.Uint16(bytes[14:16])
				contentLen := binary.BigEndian.Uint32(bytes[16:20])

				read = sofarpc.RESPONSE_HEADER_LEN_V1
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
				} else {
					// not enough data
					logger.Debugf("BoltV1 DECODE RESPONSE: no enough data for fully decode")

					return cmd, nil
				}

				sofabuffers := sofarpc.SofaProtocolBuffersByContext(ctx)
				response := &sofabuffers.BoltRsp
				//response := &sofarpc.BoltResponseCommand{}
				response.Protocol = sofarpc.PROTOCOL_CODE_V1
				response.CmdType = dataType
				response.CmdCode = int16(cmdCode)
				response.Version = ver2
				response.ReqID = requestID
				response.CodecPro = codec
				response.ResponseStatus = int16(status)
				response.ClassLen = int16(classLen)
				response.HeaderLen = int16(headerLen)
				response.ContentLen = int(contentLen)
				response.ClassName = class
				response.HeaderMap = header
				response.Content = content
				response.ResponseTimeMillis = time.Now().UnixNano() / int64(time.Millisecond)
				cmd = response

				//logger.Debugf("BoltV1 DECODE RESPONSE,RespStatus = %d, Protocol = %d, CmdType = %d, CmdCode = %d, ReqID = %d, Bytes = %d",
				//	response.ResponseStatus, response.Protocol, response.CmdType, response.CmdCode, response.ReqID, sofarpc.RESPONSE_HEADER_LEN_V1 + int(classLen)+int(headerLen)+int(contentLen) )
				/*
					response := sofarpc.BoltResponseCommand{
						sofarpc.PROTOCOL_CODE_V1,
						dataType,
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
						nil,
						time.Now().UnixNano() / int64(time.Millisecond),
						nil,
					}

					if cmdCode == uint16(sofarpc.HEARTBEAT) {
						//logger.Debugf("BoltV1 DECODE RESPONSE: Get Bolt HB Msg")
					}
					logger.Debugf("BoltV1 DECODE RESPONSE,RespStatus = %d, Protocol = %d, CmdType = %d, CmdCode = %d, ReqID = %d",
						response.ResponseStatus, response.Protocol, response.CmdType, response.CmdCode, response.ReqID)
					cmd = &response
				*/
			}
		} else {
			// 3. unknown type error
			return nil, fmt.Errorf("Decode Error, type = %s, value = %d", sofarpc.UnKnownReqtype, dataType)
		}
	}

	return cmd, nil
}
