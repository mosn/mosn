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

package bolt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/header"
)

// decoded result should equal to request
func TestEncodeDecode(t *testing.T) {
	var (
		ctx            = context.TODO()
		originalHeader = header.CommonHeader{
			"k1": "v1",
			"k2": "v2",
		}
		payload   = "hello world"
		requestID = uint32(111)
	)

	//////////// request part start
	req := NewRpcRequest(requestID, originalHeader,
		buffer.NewIoBufferString(payload))

	assert.NotNil(t, req)

	buf, err := encodeRequest(ctx, req)
	assert.Nil(t, err)

	cmd, err := decodeRequest(ctx, buf, false)
	assert.Nil(t, err)

	decodedReq, ok := cmd.(*Request)
	assert.True(t, ok)

	// the decoded header should contains every header set in beginning
	for k, v := range originalHeader {
		val, ok := decodedReq.Get(k)
		assert.True(t, ok)
		assert.Equal(t, v, val)
	}

	// should equal to the original request
	assert.Equal(t, decodedReq.GetData().String(), payload)
	assert.Equal(t, decodedReq.GetRequestId(), uint64(requestID))
	assert.Equal(t, decodedReq.CmdType, CmdTypeRequest)
	assert.False(t, decodedReq.IsHeartbeatFrame())
	assert.Equal(t, decodedReq.GetStreamType(), api.Request)

	decodedReq.SetRequestId(222)
	assert.Equal(t, decodedReq.GetRequestId(), uint64(222))

	newData := "new hello world"
	decodedReq.SetData(buffer.NewIoBufferString(newData))
	assert.Equal(t, newData, decodedReq.GetData().String())
	//////////// request part end

	//////////// response part start
	resp := NewRpcResponse(requestID, 0, originalHeader, buffer.NewIoBufferString(payload))
	buf, err = encodeResponse(ctx, resp)
	assert.Nil(t, err)

	cmd, err = decodeResponse(ctx, buf)
	assert.Nil(t, err)
	decodedResp, ok := cmd.(*Response)
	assert.True(t, ok)
	assert.Equal(t, decodedResp.GetData().String(), payload)
	assert.Equal(t, decodedResp.GetRequestId(), uint64(requestID))
	assert.Equal(t, decodedResp.CmdType, CmdTypeResponse)
	assert.False(t, decodedResp.IsHeartbeatFrame())
	assert.Equal(t, decodedResp.GetStreamType(), api.Response)

	newRespData := "new resp data"
	decodedResp.SetData(buffer.NewIoBufferString(newRespData))
	assert.Equal(t, decodedResp.GetData().String(), newRespData)

	decodedResp.SetRequestId(222)
	assert.Equal(t, decodedResp.GetRequestId(), uint64(222))
	//////////// response part end
}
