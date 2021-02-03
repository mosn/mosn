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

package tars

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func encodeRequest(ctx context.Context, request *Request) (types.IoBuffer, error) {
	sbuf := bytes.NewBuffer(nil)
	sbuf.Write(make([]byte, 4))
	os := codec.NewBuffer()
	err := request.cmd.WriteTo(os)
	if err != nil {
		return nil, err
	}
	bs := os.ToBytes()
	sbuf.Write(bs)
	len := sbuf.Len()
	binary.BigEndian.PutUint32(sbuf.Bytes(), uint32(len))
	data := sbuf.Bytes()
	return buffer.NewIoBufferBytes(data), nil
}
func encodeResponse(ctx context.Context, response *Response) (types.IoBuffer, error) {
	os := codec.NewBuffer()
	response.cmd.WriteTo(os)
	bs := os.ToBytes()
	sbuf := bytes.NewBuffer(nil)
	sbuf.Write(make([]byte, 4))
	sbuf.Write(bs)
	len := sbuf.Len()
	binary.BigEndian.PutUint32(sbuf.Bytes(), uint32(len))
	data := sbuf.Bytes()
	return buffer.NewIoBufferBytes(data), nil
}
