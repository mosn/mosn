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

package v1

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/bolt"
)

type SofaProtocolBufferCtx struct{}

func (ctx SofaProtocolBufferCtx) Name() int {
	return buffer.SofaProtocol
}

func (ctx SofaProtocolBufferCtx) New() interface{} {
	buffer := new(SofaProtocolBuffers)
	return buffer
}

func (ctx SofaProtocolBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*SofaProtocolBuffers)
	buf.BoltReq = bolt.Request{}
	buf.BoltRsp = bolt.Response{}
	buf.BoltEncodeReq = bolt.Request{}
	buf.BoltEncodeRsp = bolt.Response{}
}

type SofaProtocolBuffers struct {
	BoltReq       bolt.Request
	BoltRsp       bolt.Response
	BoltEncodeReq bolt.Request
	BoltEncodeRsp bolt.Response
}

func SofaProtocolBuffersByContext(ctx context.Context) *SofaProtocolBuffers {
	poolCtx := buffer.PoolContext(ctx)
	return poolCtx.Find(SofaProtocolBufferCtx{}, nil).(*SofaProtocolBuffers)
}
