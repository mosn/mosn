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
	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
	"mosn.io/api"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

type Request struct {
	cmd     *requestf.RequestPacket
	rawData []byte         // raw data
	data    types.IoBuffer // wrapper of data
	protocol.CommonHeader
}

var _ api.XFrame = &Request{}

// ~ XFrame
func (r *Request) GetRequestId() uint64 {
	return uint64(r.cmd.IRequestId)
}

func (r *Request) SetRequestId(id uint64) {
	r.cmd.IRequestId = int32(id)
}

func (r *Request) IsHeartbeatFrame() bool {
	// un support
	return false
}

// TODO: add timeout
func (r *Request) GetTimeout() int32 {
	return 0
}

func (r *Request) GetStreamType() api.StreamType {
	return api.Request
}

func (r *Request) GetHeader() types.HeaderMap {
	return r
}

func (r *Request) GetData() types.IoBuffer {
	return r.data
}

func (r *Request) SetData(data types.IoBuffer) {
	r.data = data
}

type Response struct {
	cmd     *requestf.ResponsePacket
	rawData []byte         // raw data
	data    types.IoBuffer // wrapper of data
	protocol.CommonHeader
}

var _ api.XRespFrame = &Response{}

// ~ XFrame
func (r *Response) GetRequestId() uint64 {
	return uint64(r.cmd.IRequestId)
}

func (r *Response) SetRequestId(id uint64) {
	r.cmd.IRequestId = int32(id)
}

func (r *Response) IsHeartbeatFrame() bool {
	// un support
	return false
}

// response contains no timeout
func (r *Response) GetTimeout() int32 {
	return -1
}

func (r *Response) GetStreamType() api.StreamType {
	return api.Response
}

func (r *Response) GetHeader() types.HeaderMap {
	return r
}

func (r *Response) GetData() types.IoBuffer {
	return r.data
}

func (r *Response) SetData(data types.IoBuffer) {
	r.data = data
}

func (r *Response) GetStatusCode() uint32 {
	return uint32(r.cmd.IRet)
}
