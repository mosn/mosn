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
	"math"

	"github.com/apache/thrift/lib/go/thrift"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func encodeRequest(ctx context.Context, request *Frame) (types.IoBuffer, error) {
	return encodeFrame(ctx, request)
}
func encodeResponse(ctx context.Context, response *Frame) (types.IoBuffer, error) {
	return encodeFrame(ctx, response)
}

func encodeFrame(ctx context.Context, frame *Frame) (types.IoBuffer, error) {

	// 1. fast-path, use existed raw data
	if frame.rawData != nil {
		//1.1 replace requestId
		idx := MessageLenSize + frame.HeaderLength - IdLen
		//i64 := make([]byte, 8)
		binary.BigEndian.PutUint64(frame.rawData[idx:], frame.Id)
		// hack: increase the buffer count to avoid premature recycle
		frame.data.Count(1)
		return frame.data, nil
	}

	bufferBytes := buffer.NewIoBuffer(1024)
	transport := thrift.NewStreamTransportW(bufferBytes)
	defer transport.Close()
	protocol := thrift.NewTBinaryProtocolTransport(transport)

	serviceName, _ := frame.GetHeader().Get(ServiceNameHeader)

	//header
	transport.Write(MagicTag)
	protocol.WriteI32(math.MaxInt32)
	protocol.WriteI16(math.MaxInt16)
	protocol.WriteByte(1)
	protocol.WriteString(serviceName)
	protocol.WriteI64(int64(frame.GetRequestId()))
	protocol.Flush(nil)

	headerLen := bufferBytes.Len()

	//message body
	bufferBytes.Write(frame.payload)

	message := bufferBytes.Bytes()
	messageLen := len(message)

	binary.BigEndian.PutUint16(message[MessageHeaderLenIdx:MessageHeaderLenIdx+MessageLenSize], uint16(headerLen))
	binary.BigEndian.PutUint32(message[MessageLenIdx:MessageLenIdx+MessageLenSize], uint32(messageLen))

	data := make([]byte, messageLen+MessageLenSize)
	binary.BigEndian.PutUint32(data, uint32(messageLen))
	copy(data[MessageLenSize:], message)

	return buffer.NewIoBufferBytes(data), nil
}
