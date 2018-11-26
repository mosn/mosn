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

import (
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/log"
)

type shadowEvent struct {
	streamID string
	stream   *shadowDownstream
}

type startShadow struct {
	shadowEvent
}

type stopShadow struct {
	shadowEvent
}

type doShadow struct {
	shadowEvent
}

func (s *shadowEvent) Source() uint32 {
	source, err := strconv.ParseUint(s.streamID, 10, 32)
	if err != nil {
		panic("streamID must be numeric, unexpected :" + s.streamID)
	}
	return uint32(source)

}

func shadowDispatch(shard int, jobChan <-chan interface{}) {
	streamMap := make(map[string]bool, 1<<10)
	for event := range jobChan {
		shadowProcess(shard, streamMap, event)
	}
}

func shadowProcess(shard int, streamMap map[string]bool, event interface{}) {
	switch event.(type) {
	case *startShadow:
		e := event.(*startShadow)
		log.DefaultLogger.Tracef("start shadow, shard=%d, id=%s", shard, e.streamID)
		streamMap[e.streamID] = false
	case *stopShadow:
		e := event.(*stopShadow)
		log.DefaultLogger.Tracef("stop shadow, shard=%d, id=%s", shard, e.streamID)
		e.stream.GiveStream()
		delete(streamMap, e.streamID)
	case *doShadow:
		e := event.(*doShadow)
		log.DefaultLogger.Tracef("receive do shadow, shard=%d, id=%s", shard, e.streamID)
		if done, ok := streamMap[e.streamID]; ok && !(done || shadowProcessDown(e.stream)) {
			log.DefaultLogger.Tracef("do shadow, shard=%d, id=%s", shard, e.streamID)
			e.stream.sendRequest()
		}
	default:
		log.DefaultLogger.Errorf("Unknown event type %s", event)
	}
}

func shadowProcessDown(s *shadowDownstream) bool {
	return s.upstreamProcessDone || s.downstreamReset == 1
}
