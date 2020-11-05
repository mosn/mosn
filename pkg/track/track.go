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
	datas            [MaxTrackPhase]TrackTime
	DataReceiveTimes []time.Time
}

func (t *Tracks) StartTrack(phase TrackPhase) {
	if phase >= MaxTrackPhase || phase < 0 {
		return
	}
	tk := t.datas[phase]
	tk.P = time.Now()
	t.datas[phase] = tk
}

func (t *Tracks) EndTrack(phase TrackPhase) {
	if phase >= MaxTrackPhase || phase < 0 {
		return
	}
	tk := t.datas[phase]
	if tk.P.IsZero() {
		return
	}
	tk.Costs = append(tk.Costs, time.Now().Sub(tk.P))
	t.datas[phase] = tk
}

func (t *Tracks) AddDataReceived() {
	t.DataReceiveTimes = append(t.DataReceiveTimes, time.Now())
}

func (t *Tracks) GetDataReceived() []time.Time {
	return t.DataReceiveTimes
}

// RangeCosts ranges the tracks data by f.
// if f returns false, terminate the range
func (t *Tracks) RangeCosts(f func(TrackPhase, TrackTime) bool) {
	for i := range t.datas {
		phase := TrackPhase(i)
		track := t.datas[i]
		if !f(phase, track) {
			return
		}
	}
}

// GetTrackCosts is a wrapper for tracks, only get strings to reserved fields
// if a extends fields added, use RangeCosts
// [][][]...[]
func (t *Tracks) GetTrackCosts() string {
	var buf strings.Builder
	t.RangeCosts(func(phase TrackPhase, track TrackTime) bool {
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
