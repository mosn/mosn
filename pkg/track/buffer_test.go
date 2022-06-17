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
	"regexp"
	"testing"
	"time"

	"mosn.io/pkg/buffer"
)

func TestTrackFromContext(t *testing.T) {
	ctx := buffer.NewBufferPoolContext(context.Background())
	tb := TrackBufferByContext(ctx).Tracks
	defer func() {
		if c := buffer.PoolContext(ctx); c != nil {
			c.Give()
		}
	}()
	for _, ph := range []TrackPhase{
		ProtocolDecode, StreamFilterBeforeRoute, MatchRoute,
	} {
		tb.StartTrack(ph)
		tb.EndTrack(ph)
	}
	tb.Range(func(p TrackPhase, tk TrackTime) bool {
		switch p {
		case ProtocolDecode, StreamFilterBeforeRoute, MatchRoute:
			if len(tk.Costs) != 1 {
				t.Fatalf("%d phase is not setted", p)
			}
		default:
		}
		return true
	})
	s := tb.GetTrackCosts()
	if !outexp.MatchString(s) {
		t.Fatalf("unexpected output: %s", s)
	}
	t.Logf("output is %s", s)
}

func TestTrackTransmit(t *testing.T) {
	dstCtx := buffer.NewBufferPoolContext(context.Background())
	srcCtx := buffer.NewBufferPoolContext(context.Background())
	dstTb := TrackBufferByContext(dstCtx).Tracks
	srcTb := TrackBufferByContext(srcCtx).Tracks
	defer func() {
		for _, ctx := range []context.Context{
			dstCtx, srcCtx,
		} {
			if c := buffer.PoolContext(ctx); c != nil {
				c.Give()
			}
		}
	}()
	// set value
	dstTb.Begin()
	for _, ph := range []TrackPhase{
		ProtocolDecode, StreamFilterBeforeRoute, MatchRoute,
	} {
		dstTb.StartTrack(ph)
		dstTb.EndTrack(ph)
	}
	time.Sleep(100 * time.Millisecond)
	srcTb.StartTrack(ProtocolDecode)
	srcTb.EndTrack(ProtocolDecode)
	srcTb.Begin()
	// Transmit
	BindRequestAndResponse(dstCtx, srcCtx)
	// Verify
	dstTb.Range(func(p TrackPhase, tk TrackTime) bool {
		switch p {
		case ProtocolDecode:
			if len(tk.Costs) != 2 {
				t.Fatalf("%d phase is not setted", p)
			}
		case StreamFilterBeforeRoute, MatchRoute:
			if len(tk.Costs) != 1 {
				t.Fatalf("%d phase is not setted", p)
			}
		default:
		}
		return true
	})
	var expectReqTime time.Time
	var expectRespTime time.Time
	dstTb.VisitTimestamp(func(p TimestampPhase, tm time.Time) bool {
		if tm.IsZero() {
			t.Fatalf("%d phase time is zero", p)
		}
		switch p {
		case RequestStartTimestamp:
			expectReqTime = tm
		case ResponseStartTimestamp:
			expectRespTime = tm
		default:
		}
		return true
	})
	if expectReqTime.IsZero() || expectRespTime.IsZero() || expectRespTime.Sub(expectReqTime) < 0 {
		t.Fatalf("request and response time is not bind success, reqtime: %v, resp time: %v", expectReqTime, expectRespTime)
	}
	// timestamp wrapper
	s := dstTb.GetTrackTimestamp()
	exp := regexp.MustCompile(`\[.+?,.+?\]`)
	if !exp.MatchString(s) {
		t.Fatalf("unexpected output timestamp: %s", s)
	}
	t.Logf("output is %s, reqtime: %v, resptime: %v", s, expectReqTime, expectRespTime)

}

func TestBufferReuse(t *testing.T) {
	bkg := context.Background()
	ctx := buffer.NewBufferPoolContext(bkg)
	tb := TrackBufferByContext(ctx)
	for _, p := range []TrackPhase{
		ProtocolDecode, StreamFilterBeforeRoute, MatchRoute, StreamFilterAfterRoute, LoadBalanceChooseHost,
		StreamFilterAfterChooseHost, NetworkDataWrite, ProtocolDecode, StreamSendFilter, NetworkDataWrite,
	} {

		tb.StartTrack(p)
		tb.EndTrack(p)
	}
	if c := buffer.PoolContext(ctx); c != nil {
		c.Give()
	}
	// a new context, all datas should be cleaned
	ctx2 := buffer.NewBufferPoolContext(bkg)
	tb2 := TrackBufferByContext(ctx2)
	tb2.Range(func(p TrackPhase, tt TrackTime) bool {
		if p > MaxServedField {
			return false
		}
		if !(tt.P.IsZero() &&
			len(tt.Costs) == 0 &&
			cap(tt.Costs) > 0) {
			t.Fatalf("%d is not reused correctlly, len: %d, cap: %d", p, len(tt.Costs), cap(tt.Costs))
		}
		return true
	})
	tb2.VisitTimestamp(func(p TimestampPhase, tm time.Time) bool {
		if !tm.IsZero() {
			t.Fatalf("%d is not cleaned", p)
		}
		return true
	})
}

func BenchmarkTrackAPI(b *testing.B) {
	bkg := context.Background()
	for i := 0; i < b.N; i++ {
		ctx := buffer.NewBufferPoolContext(bkg)
		tb := TrackBufferByContext(ctx).Tracks
		for _, p := range []TrackPhase{
			ProtocolDecode, StreamFilterBeforeRoute, MatchRoute, StreamFilterAfterRoute, LoadBalanceChooseHost,
			StreamFilterAfterChooseHost, NetworkDataWrite, ProtocolDecode, StreamSendFilter, NetworkDataWrite,
		} {

			tb.StartTrack(p)
			tb.EndTrack(p)
		}
		if c := buffer.PoolContext(ctx); c != nil {
			c.Give()
		}
	}
}
