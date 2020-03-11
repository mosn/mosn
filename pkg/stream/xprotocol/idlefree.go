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
	"math"
	"sync/atomic"
	"time"

	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/buffer"
)

var maxIdleCount uint32 = 0

// SetIdleTimeout calculates the idle timeout as max idle count.
func SetIdleTimeout(d time.Duration) {
	fd := float64(d)
	ft := float64(buffer.ConnReadTimeout)
	maxIdleCount = uint32(math.Ceil(fd / ft))
}

// If a connection is always send keep alive heartbeat, we will free the idle connection
type idleFree struct {
	idleCount    uint32
	lastStreamID uint64
}

func newIdleFree() *idleFree {
	return &idleFree{}
}

func (f *idleFree) CheckFree(id uint64) bool {
	// empty idle free means never free
	if f == nil || maxIdleCount == 0 {
		return false
	}
	// maxIdleCount is 1, free it directly
	if maxIdleCount == 1 {
		return true
	}
	if atomic.LoadUint64(&f.lastStreamID)+1 == id {
		if atomic.AddUint32(&f.idleCount, 1) >= maxIdleCount {
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
	}
	atomic.StoreUint64(&f.lastStreamID, id)
	return false
}
