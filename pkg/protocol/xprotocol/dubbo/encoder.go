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
		// 1.1 replace requestId
		binary.BigEndian.PutUint64(frame.rawData[IdIdx:], frame.Id)

		// hack: increase the buffer count to avoid premature recycle
		frame.data.Count(1)
		return frame.data, nil
	}

	// alloc encode buffer
	frameLen := int(HeaderLen + frame.DataLen)
	buf := buffer.GetIoBuffer(frameLen)
	// encode header
	buf.WriteByte(frame.Magic[0])
	buf.WriteByte(frame.Magic[1])
	buf.WriteByte(frame.Flag)
	buf.WriteByte(frame.Status)
	buf.WriteUint64(frame.Id)
	buf.WriteUint32(frame.DataLen)
	// encode payload
	buf.Write(frame.payload)
	return buf, nil
}
