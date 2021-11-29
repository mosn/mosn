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

package stream

import (
	"sync"
	"sync/atomic"

	"mosn.io/mosn/pkg/types"
)

const (
	streamStateReset uint32 = iota
	streamStateDestroying
	streamStateDestroyed
)

type BaseStream struct {
	sync.Mutex
	streamListeners []types.StreamEventListener

	state uint32
}

func (s *BaseStream) AddEventListener(streamCb types.StreamEventListener) {
	s.Lock()
	s.streamListeners = append(s.streamListeners, streamCb)
	s.Unlock()
}

func (s *BaseStream) RemoveEventListener(streamCb types.StreamEventListener) {
	s.Lock()
	defer s.Unlock()
	cbIdx := -1

	for i, cb := range s.streamListeners {
		if cb == streamCb {
			cbIdx = i
			break
		}
	}

	if cbIdx > -1 {
		s.streamListeners = append(s.streamListeners[:cbIdx], s.streamListeners[cbIdx+1:]...)
	}
}

func (s *BaseStream) ResetStream(reason types.StreamResetReason) {
	if atomic.LoadUint32(&s.state) != streamStateReset {
		return
	}
	defer s.DestroyStream()
	s.Lock()
	defer s.Unlock()

	for _, listener := range s.streamListeners {
		listener.OnResetStream(reason)
	}
}

func (s *BaseStream) DestroyStream() {
	if !atomic.CompareAndSwapUint32(&s.state, streamStateReset, streamStateDestroying) {
		return
	}
	s.Lock()
	defer s.Unlock()
	for _, listener := range s.streamListeners {
		listener.OnDestroyStream()
	}
	atomic.StoreUint32(&s.state, streamStateDestroyed)
}
