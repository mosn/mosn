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

const (
	// event type
	recvHeader  = 1
	recvData    = 2
	recvTrailer = 3
	reset       = 4

	defaultMaxEvent = 16
)

type eventSet struct {
	proxyID uint32
	reset   bool // do not accept new events if true

	// index
	consumer int
	producer int

	events [defaultMaxEvent]event
}

type event struct {
	etype  int
	handle func()
}

func (es *eventSet) Source() uint32 {
	return es.proxyID
}

func (es *eventSet) Init(id uint32) {
	es.proxyID = id
}

func (es *eventSet) Reset(ev event) {
	es.Append(ev)
	es.reset = true
}

func (es *eventSet) Append(ev event) {
	if !es.reset {
		if es.producer >= defaultMaxEvent {
			log.DefaultLogger.Errorf("events post exceeded for single set")
			return
		}
		es.events[es.producer] = ev
		es.producer++

		workerPool.Offer(es)
	}
}

func (es *eventSet) Pop() *event {
	if es.producer > es.consumer {
		if es.reset {
			// pop last event if reset
			ev := &es.events[es.producer-1]
			// clear
			es.consumer = 0
			es.producer = 0
			return ev
		} else {
			ev := &es.events[es.consumer]
			es.consumer++
			return ev
		}
	}
	return nil
}

func (es *eventSet) Size() int {
	return es.producer - es.consumer
}

func eventDispatch(shard int, jobChan <-chan interface{}) {
	for job := range jobChan {
		eventProcess(shard, job)
	}
}

func eventProcess(shard int, job interface{}) {
	if es, ok := job.(eventSet); ok {
		event := es.Pop()
		if event != nil {
			event.handle()
		}
	}
}
