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
package http

import (
	"context"

	"github.com/valyala/fasthttp"
	"mosn.io/mosn/pkg/buffer"
)

func init() {
	buffer.RegisterBuffer(&ins)
}

var ins = httpBufferCtx{}

type httpBufferCtx struct {
	buffer.TempBufferCtx
}

func (ctx httpBufferCtx) New() interface{} {
	return new(httpBuffers)
}

func (ctx httpBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*httpBuffers)
	buf.serverStream = serverStream{}
	buf.serverRequest.Reset()
	buf.serverResponse.Reset()
	buf.clientStream = clientStream{}
	buf.clientRequest.Reset()
	buf.clientResponse.Reset()
}

type httpBuffers struct {
	serverStream   serverStream
	serverRequest  fasthttp.Request
	serverResponse fasthttp.Response

	clientStream   clientStream
	clientRequest  fasthttp.Request
	clientResponse fasthttp.Response
}

func httpBuffersByContext(context context.Context) *httpBuffers {
	ctx := buffer.PoolContext(context)
	return ctx.Find(&ins, nil).(*httpBuffers)
}
