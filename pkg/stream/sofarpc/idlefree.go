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

package sofarpc

import (
	"sync/atomic"

	"sofastack.io/sofa-mosn/pkg/log"
)

// DefaultConnReadTimeout is 15s, default idle free is 600s
var defaultMaxIdleCount uint32 = 40

// If a connection is always send keep alive heartbeat, we will free the idle connection
type idleFree struct {
	maxIdleCount uint32
	idleCount    uint32
	lastStreamID uint64
}

func newIdleFree() *idleFree {
	return &idleFree{
		maxIdleCount: defaultMaxIdleCount,
	}
}

func (f *idleFree) CheckFree(id uint64) bool {
	// empty idle free means never free
	if f == nil {
		return false
	}
	if atomic.LoadUint64(&f.lastStreamID)+1 == id {
		if atomic.AddUint32(&f.idleCount, 1) >= f.maxIdleCount {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[stream] [sofarpc] [keepalive] connections only have heartbeat for a while, close it")
			}
			return true
		}
	} else {
		atomic.StoreUint32(&f.idleCount, 1)
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[stream] [sofarpc] [keepalive] last stream id: %d, current id: %d", f.lastStreamID, id)
		}
		// maxIdleCount should be greater than 1
	}
	atomic.StoreUint64(&f.lastStreamID, id)
	return false
}
