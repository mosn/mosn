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
package msgconnpool

import (
	"sync/atomic"

	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

// KeepAlive object for sending heartbeat
type KeepAlive interface {
	Stop()                            // Stop the KeepAlive and clear resources
	GetKeepAliveData() []byte         // get a heartbeat frame binary data
	SaveHeartBeatFailCallback(func()) // save the heartbeat fail callback to heartbeat object
}

// keepAliveListener is a types.ConnectionEventListener
type keepAliveListener struct {
	stopped   uint64
	keepAlive KeepAlive
	conn      api.Connection
}

// OnEvent impl types.ConnectionEventListener
func (l *keepAliveListener) OnEvent(event api.ConnectionEvent) {
	switch {
	case event == api.OnReadTimeout && atomic.LoadUint64(&l.stopped) == 0:
		l.conn.Write(buffer.NewIoBufferBytes(l.keepAlive.GetKeepAliveData()))
	}
}
