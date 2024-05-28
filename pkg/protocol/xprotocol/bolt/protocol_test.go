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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/header"
)

func TestProto(t *testing.T) {
	var (
		bp      = boltProtocol{}
		ctx     = context.TODO()
		payload = "hello world"
		header  = header.CommonHeader{"k": "v"}
		req     = NewRpcRequest(111, header, buffer.NewIoBufferString(payload))
		resp    = NewRpcResponse(111, 0, header, buffer.NewIoBufferString(payload))
	)

	assert.Equal(t, bp.Name(), ProtocolName)

	/////// request
	buf, err := bp.Encode(ctx, req)
	assert.Nil(t, err)

	cmdInter, err := bp.Decode(ctx, buf)
	assert.Nil(t, err)
	cmd, ok := cmdInter.(*Request)
	assert.True(t, ok)
	assert.Equal(t, cmd.Content.String(), payload)
	/////// request end

	/////// heartbeat
	frame := bp.Trigger(context.TODO(), 111)
	assert.NotNil(t, frame)
	assert.Equal(t, frame.(*Request).RequestHeader.CmdType, CmdTypeRequest)
	assert.Equal(t, frame.(*Request).RequestHeader.CmdCode, CmdCodeHeartbeat)
	/////// heartbeat end

	/////// response
	buf, err = bp.Encode(ctx, resp)
	assert.Nil(t, err)

	cmdInter, err = bp.Decode(ctx, buf)
	assert.Nil(t, err)
	cmdResp, ok := cmdInter.(*Response)
	assert.True(t, ok)
	assert.Equal(t, cmdResp.Content.String(), payload)
	/////// response end
}

func TestMapping(t *testing.T) {
	var (
		bp      = boltProtocol{}
		mapping = map[uint32]uint32{
			http.StatusOK:             uint32(ResponseStatusSuccess),
			api.RouterUnavailableCode: uint32(ResponseStatusNoProcessor),
			api.NoHealthUpstreamCode:  uint32(ResponseStatusNoProcessor),
			api.UpstreamOverFlowCode:  uint32(ResponseStatusServerThreadpoolBusy),
			api.CodecExceptionCode:    uint32(ResponseStatusCodecException),
			api.DeserialExceptionCode: uint32(ResponseStatusServerDeserialException),
			api.TimeoutExceptionCode:  uint32(ResponseStatusTimeout),
			999999:                    uint32(ResponseStatusUnknown),
		}
	)

	for k, v := range mapping {
		assert.Equal(t, bp.Mapping(k), v)
	}
}

func TestReply(t *testing.T) {
	bp := boltProtocol{}
	// reply heartbeat
	resp := bp.Reply(context.TODO(), NewRpcResponse(1, 0, nil, buffer.NewIoBufferString("hello")))
	assert.True(t, resp.IsHeartbeatFrame())
}

func TestHijack(t *testing.T) {
	bp := boltProtocol{}
	rsp := NewRpcResponse(1, 0, nil, buffer.NewIoBufferString("hello"))
	frame := bp.Hijack(context.TODO(), rsp, 999)
	assert.Equal(t, frame.GetStatusCode(), uint32(999))
}

func TestBufferReset(t *testing.T) {
	req := NewRpcRequest(111, nil, buffer.NewIoBufferString("hell"))
	rsp := NewRpcResponse(1, 0, nil, buffer.NewIoBufferString("hello"))
	var buf = &boltBuffer{*req, *rsp}
	ins.Reset(buf)

	assert.Nil(t, buf.request.Content)
	assert.Nil(t, buf.response.Content)
}
