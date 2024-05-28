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

package dubbothrift

import (
	"context"
	"encoding/binary"
	"fmt"
	"runtime/debug"
	"strconv"

	"github.com/apache/thrift/lib/go/thrift"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

func decodeFrame(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {

	//panic handler
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("[xprotocol][thrift]decode message error :%v\n%s", r, debug.Stack())
		}
	}()

	// convert data to dubbo frame
	dataBytes := data.Bytes()
	frame := &Frame{
		Header: Header{
			CommonHeader: protocol.CommonHeader{},
		},
	}

	//decode frame header: duplicate message length
	messageLen := binary.BigEndian.Uint32(dataBytes[:MessageLenSize])
	//ignore first message length
	body := dataBytes[MessageLenSize : MessageLenSize+messageLen]
	//total frame length
	frame.FrameLength = messageLen + MessageLenSize
	// decode magic
	frame.Magic = body[:MagicLen]
	//decode message length
	frame.MessageLength = binary.BigEndian.Uint32(body[MessageLenIdx : MessageLenIdx+MessageLenSize])
	//decode header length
	frame.HeaderLength = binary.BigEndian.Uint16(body[MessageHeaderLenIdx : MessageHeaderLenIdx+MessageHeaderLenSize])

	frame.rawData = dataBytes[:frame.FrameLength]

	frame.payload = body[frame.HeaderLength:]

	dataBuf := buffer.NewIoBufferBytes(body[HeaderIdx:])

	tTransport := thrift.NewStreamTransportR(dataBuf)
	defer tTransport.Close()
	tProtocol := thrift.NewTBinaryProtocolTransport(tTransport)

	meta := make(map[string]string)

	err = decodeHeader(ctx, frame, tTransport, tProtocol, meta)
	if err != nil {
		return nil, err
	}

	err = decodeMessage(ctx, frame, tTransport, tProtocol, meta)
	if err != nil {
		return nil, err
	}

	//set headers
	for k, v := range meta {
		frame.Set(k, v)
	}

	frame.data = buffer.NewIoBufferBytes(frame.rawData)
	switch frame.Direction {
	case EventRequest:
		// notice: read-only!!! do not modify the raw data!!!
		variable.Set(ctx, types.VarRequestRawData, frame.rawData)
	case EventResponse:
		// notice: read-only!!! do not modify the raw data!!!
		variable.Set(ctx, types.VarResponseRawData, frame.rawData)
	}
	frame.content = buffer.NewIoBufferBytes(frame.payload)

	data.Drain(int(frame.FrameLength))
	return frame, nil
}

func decodeMessage(ctx context.Context, frame *Frame, transport *thrift.StreamTransport, tProtocol *thrift.TBinaryProtocol, meta map[string]string) error {
	//method name, type, seqId
	name, messageType, seqId, err := tProtocol.ReadMessageBegin()
	if err != nil {
		return fmt.Errorf("[xprotocol][thrift] decode message body fail: %v", err)
	}

	meta[MethodNameHeader] = name
	meta[SeqIdNameHeader] = strconv.Itoa(int(seqId))
	meta[MethodNameHeader] = name
	meta[MessageTypeNameHeader] = strconv.Itoa(int(messageType))

	// check type
	if messageType == 1 {
		frame.Direction = EventRequest
	} else {
		frame.Direction = EventResponse
	}
	err = tProtocol.ReadMessageEnd()
	if err != nil {
		return fmt.Errorf("[xprotocol][thrift] decode message body fail: %v", err)
	}
	return nil
}

func decodeHeader(ctx context.Context, frame *Frame, transport *thrift.StreamTransport, tProtocol *thrift.TBinaryProtocol, meta map[string]string) error {
	serviceName, err := tProtocol.ReadString()
	if err != nil {
		return fmt.Errorf("[xprotocol][thrift] decode service path fail: %v", err)
	}
	meta[ServiceNameHeader] = serviceName
	//decode requestId
	id, err := tProtocol.ReadI64()
	if err != nil {
		return fmt.Errorf("[xprotocol][thrift] decode requestId fail: %v", err)
	}
	frame.Id = uint64(id)
	return nil
}
