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

package store

import (
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
)

type State int

var state = Init

const (
	Init State = iota
	Running
	Active_Reconfiguring
	Passive_Reconfiguring
)

func init() {
	RegisterOnStateChanged(SetStateCode)
}

func GetMosnState() State {
	return state
}

func SetMosnState(s State) {
	state = s
	log.DefaultLogger.Infof("[admin store] [mosn state] state changed to %d", s)
	for _, cb := range onStateChangedCallbacks {
		cb(s)
	}
}

type OnStateChanged func(s State)

var onStateChangedCallbacks []OnStateChanged

func RegisterOnStateChanged(f OnStateChanged) {
	onStateChangedCallbacks = append(onStateChangedCallbacks, f)
}

func SetStateCode(s State) {
	metrics.SetStateCode(int64(s))
}
