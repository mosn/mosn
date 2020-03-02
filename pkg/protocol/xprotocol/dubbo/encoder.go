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

	"mosn.io/mosn/pkg/buffer"
	"mosn.io/mosn/pkg/types"
)

func encodeRequest(ctx context.Context, request *Frame) (types.IoBuffer, error) {
	return encodeFrame(ctx, request)
}
func encodeResponse(ctx context.Context, response *Frame) (types.IoBuffer, error) {
	return encodeFrame(ctx, response)
}

func encodeFrame(ctx context.Context, frame *Frame) (types.IoBuffer, error) {
	// alloc encode buffer
	frameLen := int(HeaderLen + frame.DataLen)
	buf := *buffer.GetBytesByContext(ctx, frameLen)
	// encode header
	buf[0] = frame.Magic[0]
	buf[1] = frame.Magic[1]
	buf[2] = frame.Flag
	buf[3] = frame.Status
	binary.BigEndian.PutUint64(buf[4:], frame.Id)
	binary.BigEndian.PutUint32(buf[12:], frame.DataLen)
	// encode payload
	copy(buf[HeaderLen:], frame.payload)
	return buffer.NewIoBufferBytes(buf), nil
}
