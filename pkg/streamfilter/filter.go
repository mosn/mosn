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

package streamfilter

import "mosn.io/api"

// StreamReceiverFilterWithPhase combines the StreamReceiverFilter with its Phase.
type StreamReceiverFilterWithPhase interface {

	// StreamReceiverFilter interface.
	api.StreamReceiverFilter

	// GetPhase return the working phase of current filter.
	GetPhase() api.ReceiverFilterPhase
}

// StreamSenderFilterWithPhase combines the StreamSenderFilter which its Phase.
type StreamSenderFilterWithPhase interface {

	// StreamSenderFilter interface
	api.StreamSenderFilter

	// GetPhase return the working phase of current filter.
	GetPhase() api.SenderFilterPhase
}

// StreamReceiverFilterWithPhaseImpl is the default implementation of StreamReceiverFilterWithPhase.
type StreamReceiverFilterWithPhaseImpl struct {
	api.StreamReceiverFilter
	phase api.ReceiverFilterPhase
}

// NewStreamReceiverFilterWithPhase returns a StreamReceiverFilterWithPhaseImpl struct.
func NewStreamReceiverFilterWithPhase(
	f api.StreamReceiverFilter, p api.ReceiverFilterPhase) *StreamReceiverFilterWithPhaseImpl {
	return &StreamReceiverFilterWithPhaseImpl{
		StreamReceiverFilter: f,
		phase:                p,
	}
}

// GetPhase return the working phase of current filter.
func (s *StreamReceiverFilterWithPhaseImpl) GetPhase() api.ReceiverFilterPhase {
	return s.phase
}

// StreamSenderFilterWithPhaseImpl is default implementation of StreamSenderFilterWithPhase.
type StreamSenderFilterWithPhaseImpl struct {
	api.StreamSenderFilter
	phase api.SenderFilterPhase
}

// NewStreamSenderFilterWithPhase returns a new StreamSenderFilterWithPhaseImpl.
func NewStreamSenderFilterWithPhase(f api.StreamSenderFilter, p api.SenderFilterPhase) *StreamSenderFilterWithPhaseImpl {
	return &StreamSenderFilterWithPhaseImpl{
		StreamSenderFilter: f,
		phase:              p,
	}
}

// GetPhase return the working phase of current filter.
func (s *StreamSenderFilterWithPhaseImpl) GetPhase() api.SenderFilterPhase {
	return s.phase
}
