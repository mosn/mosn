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

package xprotocol

import (
	"sync/atomic"
)

// KeepaliveConfig is the config for xprotocol keepalive
// the config parameter can be set from the outside system
type KeepaliveConfig struct {
	// the next hb will be sent after tick_count_if_fail times of ConnReadTimeout if the current hb fails
	// if normal request comes after a heartbeat request, the next tick will be delayed
	TickCountIfFail uint32
	// the next hb will be sent after tick_count_if_succ times of ConnReadTimeout if the current hb succs
	// if normal request comes after a heartbeat request, the next tick will be delayed
	TickCountIfSucc uint32
	// if hb fails in a line, and count = fail_count_to_close, close this connection
	FailCountToClose uint32
}

var xprotoKeepaliveConfig atomic.Value

// DefaultKeepaliveConfig keeps the same with previous behavior
var DefaultKeepaliveConfig = KeepaliveConfig{
	TickCountIfFail:  1,
	TickCountIfSucc:  1,
	FailCountToClose: 6,
}

func init() {
	RefreshKeepaliveConfig(DefaultKeepaliveConfig)
}

// RefreshKeepaliveConfig refresh the keepalive config
func RefreshKeepaliveConfig(c KeepaliveConfig) {
	xprotoKeepaliveConfig.Store(c)
}
