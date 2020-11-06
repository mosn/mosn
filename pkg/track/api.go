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

func Begin(ctx context.Context) {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		tb.Begin()
	}
}

func StartTrack(ctx context.Context, phase TrackPhase) {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		tb.StartTrack(phase)
	}
}

func EndTrack(ctx context.Context, phase TrackPhase) {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		tb.EndTrack(phase)
	}
}

func RangeCosts(ctx context.Context, f func(TrackPhase, TrackTime) bool) {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		tb.RangeCosts(f)
	}
}

func VisitTimestamp(ctx context.Context, f func(TimestampPhase, time.Time) bool) {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		tb.VisitTimestamp(f)
	}
}

func GetTrackCosts(ctx context.Context) string {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		return tb.GetTrackCosts()
	}
	return ""
}

func StreamTimestamp(ctx context.Context) string {
	tb := trackBufferByContext(ctx)
	if tb != nil {
		return tb.StreamTimestamp()
	}
	return ""
}
