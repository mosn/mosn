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

	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
)

func InjectTrack(ctx context.Context) context.Context {
	// track exists, do nothing
	if TrackFromContext(ctx) != nil {
		return ctx
	}
	return mosnctx.WithValue(ctx, types.ContextKeyTrackTimes, &Tracks{})
}

func TrackFromContext(ctx context.Context) *Tracks {
	tracks := mosnctx.Get(ctx, types.ContextKeyTrackTimes)
	if tk, ok := tracks.(*Tracks); ok {
		return tk
	}
	return nil
}

func StartTrack(ctx context.Context, phase TrackPhase, track time.Time) {
	t := TrackFromContext(ctx)
	if t == nil {
		return
	}
	t.StartTrack(phase, track)
}

func EndTrack(ctx context.Context, phase TrackPhase, track time.Time) {
	t := TrackFromContext(ctx)
	if t == nil {
		return
	}
	t.EndTrack(phase, track)
}

func RangeCosts(ctx context.Context, f func(TrackPhase, TrackTime) bool) {
	track := TrackFromContext(ctx)
	if track == nil {
		return
	}
	track.RangeCosts(f)
}

func GetTrackCosts(ctx context.Context) string {
	track := TrackFromContext(ctx)
	if track == nil {
		return ""
	}
	return track.GetTrackCosts()
}
