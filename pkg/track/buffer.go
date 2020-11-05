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

package track

import (
	"context"

	"mosn.io/mosn/pkg/buffer"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
)

func init() {
	buffer.RegisterBuffer(&ins)
}

var ins = proxyBufferCtx{}

type proxyBufferCtx struct {
	buffer.TempBufferCtx
}

func (ctx proxyBufferCtx) New() interface{} {
	return new(trackBuffer)
}

func (ctx proxyBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*trackBuffer)
	*buf = trackBuffer{}
}

type trackBuffer struct {
	Tracks
}

func trackBufferByContext(ctx context.Context) *trackBuffer {
	// if track is not enabled, returns nil means no track records
	if !TrackEnabled() {
		return nil
	}
	// add a check to avoid ctx is not initialized by buffer.NewBufferPoolContext
	if val := mosnctx.Get(ctx, types.ContextKeyBufferPoolCtx); val == nil {
		return nil
	}
	poolCtx := buffer.PoolContext(ctx)
	return poolCtx.Find(&ins, nil).(*trackBuffer)
}

func TransmitBufferByContext(dst context.Context, src context.Context) {
	dstTb := trackBufferByContext(dst)
	if dstTb == nil {
		return
	}
	srcTb := trackBufferByContext(src)
	if srcTb == nil {
		return
	}
	dstTb.Tracks.DataReceiveTimes = append(dstTb.Tracks.DataReceiveTimes, srcTb.Tracks.DataReceiveTimes...)
	for p, data := range srcTb.datas {
		if len(data.Costs) > 0 {
			dstTb.datas[p].Costs = append(dstTb.datas[p].Costs, data.Costs...)
		}
	}
}
