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

package boltv2

import (
	"context"
	"encoding/binary"
	"strconv"

	"mosn.io/mosn/pkg/variable"

	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func decodeRequest(ctx context.Context, data types.IoBuffer, oneway bool) (cmd interface{}, err error) {
	bytesLen := data.Len()
	bytes := data.Bytes()

	// 1. least bytes to decode header is RequestHeaderLen(22)
	if bytesLen < RequestHeaderLen {
		return
	}

	// 2. least bytes to decode whole frame
	classLen := binary.BigEndian.Uint16(bytes[16:18])
	headerLen := binary.BigEndian.Uint16(bytes[18:20])
	contentLen := binary.BigEndian.Uint32(bytes[20:24])

	frameLen := RequestHeaderLen + int(classLen) + int(headerLen) + int(contentLen)
	if bytesLen < frameLen {
		return
	}
	data.Drain(frameLen)

	// 3. decode header
	buf := bufferByContext(ctx)
	request := &buf.request

	cmdType := bolt.CmdTypeRequest
	if oneway {
		cmdType = bolt.CmdTypeRequestOneway
	}

	request.RequestHeader = RequestHeader{
		RequestHeader: bolt.RequestHeader{
			Protocol:   ProtocolCode,
			CmdType:    cmdType,
			CmdCode:    binary.BigEndian.Uint16(bytes[2:4]),
			Version:    bytes[5],
			RequestId:  binary.BigEndian.Uint32(bytes[6:10]),
			Codec:      bytes[10],
			Timeout:    int32(binary.BigEndian.Uint32(bytes[12:16])),
			ClassLen:   classLen,
			HeaderLen:  headerLen,
			ContentLen: contentLen,
		},
		Version1:   bytes[1],
		SwitchCode: bytes[11],
	}
	request.Data = buffer.GetIoBuffer(frameLen)

	// 4. set timeout to notify proxy
	variable.SetVariableValue(ctx, types.VarProxyGlobalTimeout, strconv.Itoa(int(request.Timeout)))

	//5. copy data for io multiplexing
	request.Data.Write(bytes[:frameLen])
	request.rawData = request.Data.Bytes()

	//6. process wrappers: Class, Header, Content, Data
	headerIndex := RequestHeaderLen + int(classLen)
	contentIndex := headerIndex + int(headerLen)

	request.rawMeta = request.rawData[:RequestHeaderLen]
	if classLen > 0 {
		request.rawClass = request.rawData[RequestHeaderLen:headerIndex]
		request.Class = string(request.rawClass)
	}
	if headerLen > 0 {
		request.rawHeader = request.rawData[headerIndex:contentIndex]
		err = xprotocol.DecodeHeader(request.rawHeader, &request.Header)
	}
	if contentLen > 0 {
		request.rawContent = request.rawData[contentIndex:]
		request.Content = buffer.NewIoBufferBytes(request.rawContent)
	}
	return request, err
}

func decodeResponse(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	bytesLen := data.Len()
	bytes := data.Bytes()

	// 1. least bytes to decode header is ResponseHeaderLen(20)
	if bytesLen < ResponseHeaderLen {
		return
	}

	// 2. least bytes to decode whole frame
	classLen := binary.BigEndian.Uint16(bytes[14:16])
	headerLen := binary.BigEndian.Uint16(bytes[16:18])
	contentLen := binary.BigEndian.Uint32(bytes[18:22])

	frameLen := ResponseHeaderLen + int(classLen) + int(headerLen) + int(contentLen)
	if bytesLen < frameLen {
		return
	}
	data.Drain(frameLen)

	// 3. decode header
	buf := bufferByContext(ctx)
	response := &buf.response

	response.ResponseHeader = ResponseHeader{
		ResponseHeader: bolt.ResponseHeader{
			Protocol:       ProtocolCode,
			CmdType:        bolt.CmdTypeResponse,
			CmdCode:        binary.BigEndian.Uint16(bytes[3:5]),
			Version:        bytes[5],
			RequestId:      binary.BigEndian.Uint32(bytes[6:10]),
			Codec:          bytes[10],
			ResponseStatus: binary.BigEndian.Uint16(bytes[12:14]),
			ClassLen:       classLen,
			HeaderLen:      headerLen,
			ContentLen:     contentLen,
		},
		SwitchCode: bytes[11],
		Version1:   bytes[1],
	}
	response.Data = buffer.GetIoBuffer(frameLen)

	//TODO: test recycle by model, so we can recycle request/response models, headers also
	//4. copy data for io multiplexing
	response.Data.Write(bytes[:frameLen])
	response.rawData = response.Data.Bytes()

	//5. process wrappers: Class, Header, Content, Data
	headerIndex := ResponseHeaderLen + int(classLen)
	contentIndex := headerIndex + int(headerLen)

	response.rawMeta = response.rawData[:ResponseHeaderLen]
	if classLen > 0 {
		response.rawClass = response.rawData[ResponseHeaderLen:headerIndex]
		response.Class = string(response.rawClass)
	}
	if headerLen > 0 {
		response.rawHeader = response.rawData[headerIndex:contentIndex]
		err = xprotocol.DecodeHeader(response.rawHeader, &response.Header)
	}
	if contentLen > 0 {
		response.rawContent = response.rawData[contentIndex:]
		response.Content = buffer.NewIoBufferBytes(response.rawContent)
	}
	return response, err
}
