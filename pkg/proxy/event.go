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
	"github.com/alipay/sofa-mosn/pkg/types"
)

const (
	Downstream = 1
	Upstream   = 2
)

type streamEvent struct {
	streamId  string
	direction int
}

func (s *streamEvent) Source() int {
	source, _ := strconv.ParseInt(s.streamId, 10, 32)
	return int(source)
}

type startEvent struct {
	streamEvent
	ds *downStream
}

type receiveHeadersEvent struct {
	streamEvent

	headers   map[string]string
	endStream bool
}

type receiveDataEvent struct {
	streamEvent

	data      types.IoBuffer
	endStream bool
}

type receiveTrailerEvent struct {
	streamEvent

	trailers map[string]string
}

type resetEvent struct {
	streamEvent

	reason types.StreamResetReason
}

func eventDispatch(shard int, jobsChan chan interface{}) {

	streamMap := streamProcessMap[shard]

	for event := range jobsChan {
		switch event.(type) {

		case *startEvent:
			e := event.(*startEvent)

			streamProcessMap[shard][e.streamId] = e.ds
		case *receiveHeadersEvent:
			e := event.(*receiveHeadersEvent)

			if ds, ok := streamMap[e.streamId]; ok {
				switch e.direction {
				case Downstream:
					ds.ReceiveHeaders(e.headers, e.endStream)
				case Upstream:
					ds.upstreamRequest.ReceiveHeaders(e.headers, e.endStream)
				default:
					ds.logger.Errorf("Unknown receiveHeadersEvent direction %s", e.direction)
				}
			}
		case *receiveDataEvent:
			e := event.(*receiveDataEvent)

			if ds, ok := streamMap[e.streamId]; ok {
				switch e.direction {
				case Downstream:
					ds.ReceiveData(e.data, e.endStream)
				case Upstream:
					ds.upstreamRequest.ReceiveData(e.data, e.endStream)
				default:
					ds.logger.Errorf("Unknown receiveDataEvent direction %s", e.direction)
				}
			}
		case *receiveTrailerEvent:
			e := event.(*receiveTrailerEvent)

			if ds, ok := streamMap[e.streamId]; ok {
				switch e.direction {
				case Downstream:
					ds.ReceiveTrailers(e.trailers)
				case Upstream:
					ds.upstreamRequest.ReceiveTrailers(e.trailers)
				default:
					ds.logger.Errorf("Unknown receiveTrailerEvent direction %s", e.direction)
				}
			}
		case *resetEvent:
			e := event.(*resetEvent)

			if ds, ok := streamMap[e.streamId]; ok {
				switch e.direction {
				case Downstream:
					ds.ResetStream(e.reason)
				case Upstream:
					ds.upstreamRequest.ResetStream(e.reason)
				default:
					ds.logger.Errorf("Unknown receiveTrailerEvent direction %s", e.direction)
				}
			}
		default:
			log.DefaultLogger.Errorf("Unknown event type %s", event)
		}

	}
}
