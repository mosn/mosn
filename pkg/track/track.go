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
	"strings"
	"time"
)

type TrackTime struct {
	P     time.Time
	Costs []time.Duration
}

type Tracks struct {
	datas    [MaxTrackPhase]TrackTime
	times    [MaxTimestampPhase]time.Time
	disabled bool
}

func (t *Tracks) Begin() {
	if t == nil || t.disabled {
		return
	}
	if t.times[TrackStartTimestamp].IsZero() {
		t.times[TrackStartTimestamp] = time.Now()
	}
}

func (t *Tracks) StartTrack(phase TrackPhase) {
	if t == nil || t.disabled || phase >= MaxTrackPhase || phase <= NoTrack {
		return
	}
	t.datas[phase].P = time.Now()
}

func (t *Tracks) EndTrack(phase TrackPhase) {
	if t == nil || t.disabled || phase >= MaxTrackPhase || phase <= NoTrack {
		return
	}
	tk := t.datas[phase]
	if tk.P.IsZero() {
		return
	}
	tk.Costs = append(tk.Costs, time.Since(tk.P))
	t.datas[phase] = tk
}

// Range ranges the tracks data by f.
// if f returns false, terminate the range
func (t *Tracks) Range(f func(TrackPhase, TrackTime) bool) {
	if t == nil || t.disabled {
		return
	}
	for i := range t.datas {
		phase := TrackPhase(i)
		track := t.datas[i]
		if !f(phase, track) {
			return
		}
	}
}

func (t *Tracks) VisitTimestamp(f func(TimestampPhase, time.Time) bool) {
	if t == nil || t.disabled {
		return
	}
	for i := range t.times {
		phase := TimestampPhase(i)
		timestamp := t.times[i]
		if !f(phase, timestamp) {
			return
		}
	}
}

// GetTrackCosts is a wrapper for tracks.Range, only get strings to reserved fields
// if a extends fields added, use Range
// [][][]...[]
func (t *Tracks) GetTrackCosts() string {
	// fast fail
	if t == nil {
		return ""
	}
	var buf strings.Builder
	t.Range(func(phase TrackPhase, track TrackTime) bool {
		if phase > MaxServedField {
			return false
		}
		buf.WriteString("[")
		for i, c := range track.Costs {
			buf.WriteString(c.String())
			if i+1 < len(track.Costs) {
				buf.WriteString(",")
			}
		}
		buf.WriteString("]")
		return true
	})
	return buf.String()
}

// GetTrackTimestamp is a wrapper for tracks.VisitTimestamp, get request and response timestamp
func (t *Tracks) GetTrackTimestamp() string {
	// fast fail
	if t == nil {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("[")
	t.VisitTimestamp(func(p TimestampPhase, tm time.Time) bool {
		switch p {
		case RequestStartTimestamp:
			if !tm.IsZero() {
				buf.WriteString(tm.Format("2006-01-02 15:04:05.000"))
			}
			buf.WriteString(",")
		case ResponseStartTimestamp:
			if !tm.IsZero() {
				buf.WriteString(tm.Format("2006-01-02 15:04:05.000"))
			}
		default:
		}
		return true
	})
	buf.WriteString("]")
	return buf.String()
}
