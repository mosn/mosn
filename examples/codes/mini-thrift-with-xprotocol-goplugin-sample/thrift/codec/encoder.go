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

package main

import (
	"context"
	"encoding/binary"

	"mosn.io/pkg/buffer"
)

var pingCmd = []byte{
	128, 1, 0, 1, 0, 0, 0, 4, 112, 105, 110, 103, 0, 0, 0, 0, 0,
}

var pongCmd = []byte{
	128, 1, 0, 2, 0, 0, 0, 4, 112, 111, 110, 103, 0, 0, 0, 0, 0,
}

func encodeRequest(ctx context.Context, request *Request) (buffer.IoBuffer, error) {
	// 1. fast-path, use existed raw data
	logger.Printf("in encodeRequest")

	switch request.CmdCode {
	case CmdCodeHeartbeat:
		request.Data = buffer.GetIoBuffer(len(pingCmd))

		//5. copy data for io multiplexing
		request.Data.Write(pingCmd)
		request.rawData = request.Data.Bytes()
		request.rawContent = request.rawData[:]
		request.Content = buffer.NewIoBufferBytes(request.rawContent)
	default:

	}

	nameLen := binary.BigEndian.Uint32(request.rawData[4:8])
	binary.BigEndian.PutUint32(request.rawData[8+nameLen:8+nameLen+4], request.RequestId)

	//fmt.Printf("encodeRequest %+v\n", request)

	//binary.BigEndian.PutUint32(request.rawMeta[RequestIdIndex:], request.RequestId)
	// 1.2 check if header/content changed
	if !request.BytesHeader.Changed && !request.ContentChanged {
		// hack: increase the buffer count to avoid premature recycle
		request.Data.Count(1)
	}

	buf := buffer.GetIoBuffer(len(request.Data.Bytes()))
	buf.Write(request.Data.Bytes())

	return buf, nil
}

func encodeResponse(ctx context.Context, response *Response) (buffer.IoBuffer, error) {
	logger.Printf("in encodeResponse")

	switch response.CmdCode {
	case CmdCodeHeartbeat:
		response.Data = buffer.GetIoBuffer(len(pongCmd))
		//fmt.Printf("encodeResponse: %+v", response)
		//5. copy data for io multiplexing
		response.Data.Write(pongCmd)
		response.rawData = response.Data.Bytes()
		response.rawContent = response.rawData[:]
		response.Content = buffer.NewIoBufferBytes(response.rawContent)
	default:

	}

	nameLen := binary.BigEndian.Uint32(response.rawData[4:8])
	binary.BigEndian.PutUint32(response.rawData[8+nameLen:8+nameLen+4], response.RequestId)

	//fmt.Printf("encodeResponse %+v\n", response)

	//binary.BigEndian.PutUint32(request.rawMeta[RequestIdIndex:], request.RequestId)
	// 1.2 check if header/content changed
	if !response.BytesHeader.Changed && !response.ContentChanged {
		// hack: increase the buffer count to avoid premature recycle
		response.Data.Count(1)
	}

	buf := buffer.GetIoBuffer(len(response.Data.Bytes()))
	buf.Write(response.Data.Bytes())

	return buf, nil
}
