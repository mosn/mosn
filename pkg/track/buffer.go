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
	"time"

	"mosn.io/pkg/buffer"
)

func init() {
	buffer.RegisterBuffer(&ins)
}

var ins = trackBufferCtx{}

type trackBufferCtx struct {
	buffer.TempBufferCtx
}

func (ctx trackBufferCtx) New() interface{} {
	return &TrackBuffer{
		Tracks: &Tracks{},
	}
}

type TrackBuffer struct {
	*Tracks
}

func (ctx trackBufferCtx) Reset(i interface{}) {
	buf, _ := i.(*TrackBuffer)
	buf.disabled = false
	for i := range buf.times {
		buf.times[i] = time.Time{}
	}
	for i := range buf.datas {
		buf.datas[i].P = time.Time{}
		buf.datas[i].Costs = buf.datas[i].Costs[:0]
	}
}

func TrackBufferByContext(ctx context.Context) *TrackBuffer {
	poolCtx := buffer.PoolContext(ctx)
	tb := poolCtx.Find(&ins, nil).(*TrackBuffer)
	// once the track enabled is false, the track is disabled.
	if !TrackEnabled() {
		tb.disabled = true
	}
	return tb
}

func BindRequestAndResponse(req context.Context, resp context.Context) {
	reqTb := TrackBufferByContext(req)
	respTb := TrackBufferByContext(resp)
	if reqTb == nil || respTb == nil {
		return
	}
	// response have protocol decodes only
	// just bind it
	p := ProtocolDecode
	tk := respTb.datas[p]
	reqTb.datas[p].P = tk.P
	if len(tk.Costs) > 0 {
		reqTb.datas[p].Costs = append(reqTb.datas[p].Costs, tk.Costs...)
	}
	reqTb.times[RequestStartTimestamp] = reqTb.times[TrackStartTimestamp]
	reqTb.times[ResponseStartTimestamp] = respTb.times[TrackStartTimestamp]
}
