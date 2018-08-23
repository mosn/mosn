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
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"context"
)

type SofaProtocolBufferCtx struct{}

func (ctx SofaProtocolBufferCtx) Name() int {
	return buffer.SofaProtocol
}

func (ctx SofaProtocolBufferCtx) New(interface{}) interface{} {
	buffer := new(SofaProtocolBuffers)
	return buffer
}

func (ctx SofaProtocolBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*SofaProtocolBuffers)
	buf.BoltReq = BoltRequestCommand{}
	buf.BoltRsp = BoltResponseCommand{}
	buf.BoltEncodeReq = BoltRequestCommand{}
	buf.BoltEncodeRsp = BoltResponseCommand{}
}

type SofaProtocolBuffers struct {
	BoltReq BoltRequestCommand
	BoltRsp BoltResponseCommand
	BoltEncodeReq BoltRequestCommand
	BoltEncodeRsp BoltResponseCommand
}

func SofaProtocolBuffersByContent(context context.Context) *SofaProtocolBuffers {
	ctx := buffer.PoolContext(context)
	if ctx == nil {
		return nil
	}
	return ctx.Find(SofaProtocolBufferCtx{}, nil).(*SofaProtocolBuffers)
}