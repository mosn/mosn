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

package sofarpc

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/buffer"
)

func init() {
	buffer.RegisterBuffer(&ins)
}

var ins = SofaProtocolBufferCtx{}

type SofaProtocolBufferCtx struct{
	buffer.TempBufferCtx
}

func (ctx SofaProtocolBufferCtx) New() interface{} {
	buffer := new(SofaProtocolBuffers)
	return buffer
}

func (ctx SofaProtocolBufferCtx) Reset(i interface{}) {
	// TODO All fields of these will be assigned every time. So we can remove reset logic to avoid duffcopy

	buf, _ := i.(*SofaProtocolBuffers)
	buf.BoltReq = BoltRequest{}
	buf.BoltRsp = BoltResponse{}
	buf.BoltEncodeReq = BoltRequest{}
	buf.BoltEncodeRsp = BoltResponse{}
}

type SofaProtocolBuffers struct {
	BoltReq       BoltRequest
	BoltReqHeader *[]byte
	BoltRsp       BoltResponse
	BoltRspHeader *[]byte
	BoltEncodeReq BoltRequest
	BoltEncodeRsp BoltResponse
}

func (b *SofaProtocolBuffers) GetBoltReqHeader(size int) *[]byte {
	if b.BoltReqHeader != nil {
		if size <= cap(*b.BoltReqHeader) {
			*b.BoltReqHeader = (*b.BoltReqHeader)[0:size]
			return b.BoltReqHeader
		} else {
			buffer.PutBytes(b.BoltReqHeader)
		}
	}

	b.BoltReqHeader = buffer.GetBytes(size)
	return b.BoltReqHeader
}

func (b *SofaProtocolBuffers) GetBoltRspHeader(size int) *[]byte {
	if b.BoltRspHeader != nil {
		if size <= cap(*b.BoltRspHeader) {
			*b.BoltRspHeader = (*b.BoltRspHeader)[0:size]
			return b.BoltRspHeader
		} else {
			buffer.PutBytes(b.BoltRspHeader)
		}
	}

	b.BoltRspHeader = buffer.GetBytes(size)
	return b.BoltRspHeader
}

func SofaProtocolBuffersByContext(ctx context.Context) *SofaProtocolBuffers {
	poolCtx := buffer.PoolContext(ctx)
	return poolCtx.Find(&ins, nil).(*SofaProtocolBuffers)
}
