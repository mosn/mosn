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

	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/davecgh/go-spew/spew"
	mbuffer "mosn.io/mosn/pkg/buffer"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func encodeRequest(ctx context.Context, request *Frame) (types.IoBuffer, error) {
	encoder := hessian.NewEncoder()
	withTimeoutAttachment := make([]byte, request.DataLen-request.attachmentLen)
	copy(withTimeoutAttachment, request.payload[:request.DataLen-request.attachmentLen])
	encoder.Append(withTimeoutAttachment)
	encoder.Encode(request.attachment)

	byteArray := encoder.Buffer()
	pkgLen := len(byteArray)

	// alloc encode buffer
	frameLen := int(HeaderLen + pkgLen)
	buf := *mbuffer.GetBytesByContext(ctx, frameLen)
	// encode header
	buf[0] = request.Magic[0]
	buf[1] = request.Magic[1]
	buf[2] = request.Flag
	buf[3] = request.Status
	binary.BigEndian.PutUint64(buf[4:], request.Id)
	binary.BigEndian.PutUint32(buf[12:], uint32(pkgLen))

	// encode payload
	copy(buf[HeaderLen:], byteArray)

	return buffer.NewIoBufferBytes(buf), nil
}

func encodeResponse(ctx context.Context, response *Frame) (types.IoBuffer, error) {
	// alloc encode buffer
	frameLen := int(HeaderLen + response.DataLen)
	buf := *mbuffer.GetBytesByContext(ctx, frameLen)
	// encode header
	buf[0] = response.Magic[0]
	buf[1] = response.Magic[1]
	buf[2] = response.Flag
	buf[3] = response.Status
	binary.BigEndian.PutUint64(buf[4:], response.Id)
	binary.BigEndian.PutUint32(buf[12:], response.DataLen)

	// encode payload
	copy(buf[HeaderLen:], response.payload)
	spew.Dump(buf)

	return buffer.NewIoBufferBytes(buf), nil
}
