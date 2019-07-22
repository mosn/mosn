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

package types

import "time"

type KeepAlive interface {
	// SendKeepAlive sends a heartbeat request for keepalive
	SendKeepAlive()
	// StartIdleTimeout starts the idle checker, if there are only heartbeat requests for a while,
	// we will free the idle always connection, stop keeps it alive.
	StartIdleTimeout()
	GetTimeout() time.Duration
	HandleTimeout(id uint64)
	HandleSuccess(id uint64)
	AddCallback(cb KeepAliveCallback)
	Stop()
}

type KeepAliveStatus int

const (
	KeepAliveSuccess KeepAliveStatus = iota
	KeepAliveTimeout
)

// KeepAliveCallback is a callback when keep alive handle response/timeout
type KeepAliveCallback func(KeepAliveStatus)
