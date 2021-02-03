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
	"context"

	mbuffer "mosn.io/mosn/pkg/buffer"
	"mosn.io/pkg/buffer"
)

func init() {
	mbuffer.RegisterBuffer(&ins)
}

var (
	ins = protocolBufferCtx{}

	defaultMapSize    = 1 << 3
	defaultDataSize   = 1 << 10
	defaultHeaderSize = 1 << 5
)

type ProtocolBuffers struct {
	reqData     buffer.IoBuffer
	reqHeader   buffer.IoBuffer
	reqHeaders  map[string]string
	reqTrailers map[string]string

	rspData     buffer.IoBuffer
	rspHeader   buffer.IoBuffer
	rspHeaders  map[string]string
	rspTrailers map[string]string
}

type protocolBufferCtx struct {
	mbuffer.TempBufferCtx
}

func (ctx protocolBufferCtx) New() interface{} {
	p := new(ProtocolBuffers)

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

// GetReqData returns IoBuffer for request data
func (p *ProtocolBuffers) GetReqData(size int) buffer.IoBuffer {
	if size <= 0 {
		size = defaultDataSize
	}
	p.reqData = buffer.GetIoBuffer(size)
	return p.reqData
}

// GetReqHeader returns IoBuffer for request header
func (p *ProtocolBuffers) GetReqHeader(size int) buffer.IoBuffer {
	if size <= 0 {
		size = defaultHeaderSize
	}
	p.reqHeader = buffer.GetIoBuffer(size)
	return p.reqHeader
}

// GetReqHeaders returns map for request header
func (p *ProtocolBuffers) GetReqHeaders() map[string]string {
	return p.reqHeaders
}

// GetReqTailers returns map for request tailers
func (p *ProtocolBuffers) GetReqTailers() map[string]string {
	return p.reqTrailers
}

// GetRspData returns IoBuffer for response data
func (p *ProtocolBuffers) GetRspData(size int) buffer.IoBuffer {
	if size <= 0 {
		size = defaultDataSize
	}
	p.rspData = buffer.GetIoBuffer(size)
	return p.rspData
}

// GetRspHeader returns IoBuffer for response header
func (p *ProtocolBuffers) GetRspHeader(size int) buffer.IoBuffer {
	if size <= 0 {
		size = defaultHeaderSize
	}
	p.rspHeader = buffer.GetIoBuffer(size)
	return p.rspHeader
}

// GetRspHeaders returns map for response header
func (p *ProtocolBuffers) GetRspHeaders() map[string]string {
	return p.rspHeaders
}

// GetRspTailers returns IoBuffer for response tailers
func (p *ProtocolBuffers) GetRspTailers() map[string]string {
	return p.rspTrailers
}

// ProtocolBuffersByContext returns ProtocolBuffers by context
func ProtocolBuffersByContext(ctx context.Context) *ProtocolBuffers {
	poolCtx := mbuffer.PoolContext(ctx)
	return poolCtx.Find(&ins, nil).(*ProtocolBuffers)
}
