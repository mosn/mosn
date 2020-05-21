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

package boltv2

import (
	"context"

	"mosn.io/pkg/log"

	mbuffer "mosn.io/mosn/pkg/buffer"
	"mosn.io/pkg/buffer"
)

var ins boltv2BufferCtx

func init() {
	mbuffer.RegisterBuffer(&ins)
}

type boltv2BufferCtx struct {
	mbuffer.TempBufferCtx
}

func (ctx boltv2BufferCtx) New() interface{} {
	return new(boltv2Buffer)
}

func (ctx boltv2BufferCtx) Reset(i interface{}) {
	buf, _ := i.(*boltv2Buffer)

	// recycle ioBuffer
	if buf.request.Data != nil {
		if e := buffer.PutIoBuffer(buf.request.Data); e != nil {
			log.DefaultLogger.Errorf("[protocol] [bolt] [buffer] [reset] PutIoBuffer error: %v", e)
		}
	}

	if buf.response.Data != nil {
		if e := buffer.PutIoBuffer(buf.response.Data); e != nil {
			log.DefaultLogger.Errorf("[protocol] [bolt] [buffer] [reset] PutIoBuffer error: %v", e)
		}
	}

	*buf = boltv2Buffer{}
}

type boltv2Buffer struct {
	request  Request
	response Response
}

func bufferByContext(ctx context.Context) *boltv2Buffer {
	poolCtx := mbuffer.PoolContext(ctx)
	return poolCtx.Find(&ins, nil).(*boltv2Buffer)
}
