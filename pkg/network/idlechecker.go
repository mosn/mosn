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

package network

import (
	"math"
	"sync/atomic"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/buffer"
)

// getIdleCount calculates the idle timeout as max idle count.
func getIdleCount(readTimeout time.Duration, idleTimeout time.Duration) uint32 {
	if readTimeout == 0 {
		readTimeout = buffer.ConnReadTimeout
	}
	fd := float64(idleTimeout)
	ft := float64(readTimeout)
	return uint32(math.Ceil(fd / ft))
}

// idleChecker checks whether a server side connection is idle
// if the idleCount is greater than maxIdleCount, close the connection
// idleChecker is an implementation of types.ConnectionEventListener
type idleChecker struct {
	conn         *connection
	maxIdleCount uint32
	idleCount    uint32
	lastWrite    int64
	lastRead     int64
}

func (conn *connection) newIdleChecker(readTimeout time.Duration, idleTimeout time.Duration) {
	checker := &idleChecker{
		conn:         conn,
		maxIdleCount: getIdleCount(readTimeout, idleTimeout),
	}
	log.DefaultLogger.Debugf("new idlechecker: maxIdleCount:%d, conn:%d", checker.maxIdleCount, conn.id)
	conn.AddConnectionEventListener(checker)
}

func (c *idleChecker) closeConnection() {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[network] [server idle checker] close the idle connection %d", c.conn.id)
	}
	c.conn.Close(api.NoFlush, api.LocalClose)
}

func (c *idleChecker) OnEvent(event api.ConnectionEvent) {
	if event != api.OnReadTimeout || c == nil || c.maxIdleCount == 0 {
		return
	}
	// if maxIdleCount is 1, close the connection directly
	if c.maxIdleCount == 1 {
		c.closeConnection()
		return
	}
	read := c.conn.stats.ReadTotal.Count()
	write := c.conn.stats.WriteTotal.Count()
	if atomic.LoadInt64(&c.lastWrite) == write && atomic.LoadInt64(&c.lastRead) == read {
		if atomic.AddUint32(&c.idleCount, 1) >= c.maxIdleCount {
			c.closeConnection()
			return
		}
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[network] [server idle checker] connection idle %d times, maxIdleCount:%d", atomic.LoadUint32(&c.idleCount), c.maxIdleCount)
		}
	} else {
		atomic.StoreUint32(&c.idleCount, 1)
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[network] [server idle checker] connection have read/write data before this read timeout: %d, %d, %d, %d",
				atomic.LoadInt64(&c.lastRead),
				read,
				atomic.LoadInt64(&c.lastWrite),
				write,
			)
		}
	}
	atomic.StoreInt64(&c.lastWrite, write)
	atomic.StoreInt64(&c.lastRead, read)
}
