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

package protocol

import (
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"context"
)

var defaultMapSize = 1 << 3
var defaultDataSize = 1 << 10
var defaultHeaderSize = 1 << 5

type ProtocolBuffers struct {
	reqData     types.IoBuffer
	reqHeader   types.IoBuffer
	reqHeaders  map[string]string
	reqTrailers map[string]string

	rspData     types.IoBuffer
	rspHeader   types.IoBuffer
	rspHeaders  map[string]string
	rspTrailers map[string]string

	ioBufferPool *buffer.IoBufferPool
}

type protocolBufferCtx struct{}

func (ctx protocolBufferCtx) Name() int {
	return buffer.Protocol
}

func (ctx protocolBufferCtx) New() interface{} {
	p := new(ProtocolBuffers)

	p.ioBufferPool = buffer.NewIoBufferPool()

	p.reqHeaders = make(map[string]string, defaultMapSize)
	p.rspHeaders = make(map[string]string, defaultMapSize)
	p.reqTrailers = make(map[string]string)
	p.rspTrailers = make(map[string]string)
	return p
}

func (ctx protocolBufferCtx) Reset(i interface{}) {
	p, _ := i.(*ProtocolBuffers)

	p.reqData = nil
	p.reqHeader = nil
	p.rspData = nil
	p.rspHeader = nil

	for k := range p.reqHeaders {
		delete(p.reqHeaders, k)
	}
	for k := range p.reqTrailers {
		delete(p.reqTrailers, k)
	}
	for k := range p.rspHeaders {
		delete(p.rspHeaders, k)
	}
	for k := range p.rspTrailers {
		delete(p.rspTrailers, k)
	}
}

func (p *ProtocolBuffers) GetReqData(size int) types.IoBuffer {
	if size <= 0 {
		size = defaultDataSize
	}
	p.reqData = p.ioBufferPool.Take(size)
	return p.reqData
}

func (p *ProtocolBuffers) GetReqHeader(size int) types.IoBuffer {
	if size <= 0 {
		size = defaultHeaderSize
	}
	p.reqHeader = p.ioBufferPool.Take(size)
	return p.reqHeader
}

func (p *ProtocolBuffers) GetReqHeaders() map[string]string {
	return p.reqHeaders
}

func (p *ProtocolBuffers) GetReqTailers() map[string]string {
	return p.reqTrailers
}

func (p *ProtocolBuffers) GetRspData(size int) types.IoBuffer {
	if size <= 0 {
		size = defaultDataSize
	}
	p.rspData = p.ioBufferPool.Take(size)
	return p.rspData
}

func (p *ProtocolBuffers) GetRspHeader(size int) types.IoBuffer {
	if size <= 0 {
		size = defaultHeaderSize
	}
	p.rspHeader = p.ioBufferPool.Take(size)
	return p.rspHeader
}

func (p *ProtocolBuffers) GetRspHeaders() map[string]string {
	return p.rspHeaders
}

func (p *ProtocolBuffers) GetRspTailers() map[string]string {
	return p.rspTrailers
}

func ProtocolBuffersByContent(context context.Context) *ProtocolBuffers {
	ctx := buffer.PoolContext(context)
	if ctx == nil {
		return nil
	}
	return ctx.Find(protocolBufferCtx{}, nil).(*ProtocolBuffers)
}