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

	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/log"
	mosnsync "github.com/alipay/sofa-mosn/pkg/sync"
)

const (
	Downstream = 1
	Upstream   = 2
)

type streamEvent struct {
	direction int
	streamId  string
	stream    *downStream
}

func (s *streamEvent) Source() int {
	source, _ := strconv.ParseInt(s.streamId, 10, 32)
	return int(source)
}

// control evnets
type controlEvent struct {
	streamEvent
}

func (s *controlEvent) Type() int {
	return mosnsync.CONTROL
}

type startEvent struct {
	controlEvent
}

type stopEvent struct {
	controlEvent
}

type resetEvent struct {
	controlEvent

	reason types.StreamResetReason
}

// job events
type normalEvent struct {
	streamEvent
}

func (s *normalEvent) Type() int {
	return mosnsync.NORMAL
}


type receiveHeadersEvent struct {
	normalEvent

	headers   map[string]string
	endStream bool
}

type receiveDataEvent struct {
	normalEvent

	data      types.IoBuffer
	endStream bool
}

type receiveTrailerEvent struct {
	normalEvent

	trailers map[string]string
}

func eventDispatch(shard int, jobChan chan interface{}, ctrlChan chan interface{}) {
	// stream process status map with shard, we use this to indicate a given stream is processing or not
	streamMap := make(map[string]bool, 1<<10)

	for {
		var event interface{}

		// control event process first
		select {
		case event = <-ctrlChan:
			eventProcess(streamMap, event)
			// next loop
			continue
		default:
		}

		// job & ctrl event process
		select {
		case event = <-ctrlChan:
		case event = <-jobChan:
		}

		eventProcess(streamMap, event)
	}
}

func eventProcess(streamMap map[string]bool, event interface{}) {
	// TODO: event handles by itself. just call event.handle() here
	switch event.(type) {
	case *startEvent:
		e := event.(*startEvent)
		//log.DefaultLogger.Debugf("[start event] direction %d, streamId %s", e.direction, e.stream.streamID)

		streamMap[e.streamId] = false
	case *stopEvent:
		e := event.(*stopEvent)
		//log.DefaultLogger.Debugf("[stop event] direction %d, streamId %s", e.direction, e.stream.streamID)

		delete(streamMap, e.streamId)
	case *resetEvent:
		e := event.(*resetEvent)
		//log.DefaultLogger.Debugf("[reset event] direction %d, streamId %s", e.direction, e.stream.streamID)

		if done, ok := streamMap[e.streamId]; ok && !done {
			switch e.direction {
			case Downstream:
				e.stream.ResetStream(e.reason)
			case Upstream:
				e.stream.upstreamRequest.ResetStream(e.reason)
			default:
				e.stream.logger.Errorf("Unknown receiveTrailerEvent direction %s", e.direction)
			}
			streamMap[e.streamId] =  streamMap[e.streamId]  || e.stream.upstreamProcessDone
		}
	case *receiveHeadersEvent:
		e := event.(*receiveHeadersEvent)
		//log.DefaultLogger.Debugf("[header event] direction %d, streamId %s", e.direction, e.stream.streamID)

		if done, ok := streamMap[e.streamId]; ok && !done {
			switch e.direction {
			case Downstream:
				e.stream.ReceiveHeaders(e.headers, e.endStream)
			case Upstream:
				e.stream.upstreamRequest.ReceiveHeaders(e.headers, e.endStream)
			default:
				e.stream.logger.Errorf("Unknown receiveHeadersEvent direction %s", e.direction)
			}
			streamMap[e.streamId] =  streamMap[e.streamId]  || e.stream.upstreamProcessDone
		}
	case *receiveDataEvent:
		e := event.(*receiveDataEvent)
		//log.DefaultLogger.Debugf("[data event] direction %d, streamId %s", e.direction, e.stream.streamID)

		if done, ok := streamMap[e.streamId]; ok && !done {
			switch e.direction {
			case Downstream:
				e.stream.ReceiveData(e.data, e.endStream)
			case Upstream:
				e.stream.upstreamRequest.ReceiveData(e.data, e.endStream)
			default:
				e.stream.logger.Errorf("Unknown receiveDataEvent direction %s", e.direction)
			}
			streamMap[e.streamId] =  streamMap[e.streamId]  || e.stream.upstreamProcessDone
		}
	case *receiveTrailerEvent:
		e := event.(*receiveTrailerEvent)
		//log.DefaultLogger.Debugf("[trailer event] direction %d, streamId %s", e.direction, e.stream.streamID)

		if done, ok := streamMap[e.streamId]; ok && !done {
			switch e.direction {
			case Downstream:
				e.stream.ReceiveTrailers(e.trailers)
			case Upstream:
				e.stream.upstreamRequest.ReceiveTrailers(e.trailers)
			default:
				e.stream.logger.Errorf("Unknown receiveTrailerEvent direction %s", e.direction)
			}
			streamMap[e.streamId] =  streamMap[e.streamId]  || e.stream.upstreamProcessDone
		}

	default:
		log.DefaultLogger.Errorf("Unknown event type %s", event)
	}
}
