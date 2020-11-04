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
)

func StartTrack(ctx context.Context, phase TrackPhase, track time.Time) {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		tb.StartTrack(phase, track)
	}
}

func EndTrack(ctx context.Context, phase TrackPhase, track time.Time) {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		tb.EndTrack(phase, track)
	}
}

func RangeCosts(ctx context.Context, f func(TrackPhase, TrackTime) bool) {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		tb.RangeCosts(f)
	}
}

func SetRequestReceiveTime(ctx context.Context, t time.Time) {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		tb.RequestReceiveTime = t
	}
}

func GetRequestReceiveTime(ctx context.Context) time.Time {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		return tb.RequestReceiveTime
	}
	return time.Time{}
}

func SetResponseReceiveTime(ctx context.Context, t time.Time) {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		tb.ResponseReceiveTime = t
	}
}

func GetResponseReceiveTime(ctx context.Context) time.Time {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		return tb.ResponseReceiveTime
	}
	return time.Time{}
}

func GetTrackCosts(ctx context.Context) string {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		return tb.GetTrackCosts()
	}
	return ""
}
