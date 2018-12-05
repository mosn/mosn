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

package proxy

import "github.com/alipay/sofa-mosn/pkg/log"

type direction uint8
type eventType uint8

const (
	// direction
	downstream direction = 1
	upstream   direction = 2

	// event type
	recvHeader  eventType = 1
	recvData    eventType = 2
	recvTrailer eventType = 3
	reset       eventType = 4
)

type event struct {
	id  uint32
	dir direction
	evt eventType

	handle func()
}

func (ev *event) Source() uint32 {
	return ev.id
}

func eventDispatch(shard int, jobChan <-chan interface{}) {
	for job := range jobChan {
		eventProcess(shard, job)
	}
}

func eventProcess(shard int, job interface{}) {
	if ev, ok := job.(*event); ok {
		if log.DefaultLogger.Level >= log.DEBUG {
			log.DefaultLogger.Debugf("enter event process, proxyID = %d, dir = %d, type = %d", ev.id, ev.dir, ev.evt)
		}

		ev.handle()
	}
}
