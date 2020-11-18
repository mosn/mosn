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

package filter

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

// StreamFilterChainStatus determines the running status of filter chain.
type StreamFilterChainStatus int

const (
	// StreamFilterChainContinue continues running the filter chain.
	StreamFilterChainContinue StreamFilterChainStatus = iota
	// StreamFilterChainStop stops running the filter chain, next time should retry the current filter.
	StreamFilterChainStop
	// StreamFilterChainReset stops running the filter chain and reset index, next time should run the first filter.
	StreamFilterChainReset
)

// StreamFilterStatusHandler converts api.StreamFilterStatus to StreamFilterChainStatus.
type StreamFilterStatusHandler func(status api.StreamFilterStatus) StreamFilterChainStatus

// DefaultStreamFilterStatusHandler is the default implementation of StreamFilterStatusHandler.
func DefaultStreamFilterStatusHandler(status api.StreamFilterStatus) StreamFilterChainStatus {
	switch status {
	case api.StreamFilterContinue:
		return StreamFilterChainContinue
	case api.StreamFilterStop:
		return StreamFilterChainReset
	case api.StreamFiltertermination:
		return StreamFilterChainReset
	}

	return StreamFilterChainContinue
}

// StreamFilterManager manages the lifecycle of streamFilters.
type StreamFilterManager interface {
	// register StreamSenderFilter, StreamReceiverFilter and AccessLog.
	api.StreamFilterChainFactoryCallbacks

	// invoke the receiver filter chain.
	RunReceiverFilter(ctx context.Context, phase api.ReceiverFilterPhase,
		headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
		statusHandler StreamFilterStatusHandler) api.StreamFilterStatus

	// invoke the sender filter chain.
	RunSenderFilter(ctx context.Context, phase api.SenderFilterPhase,
		headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
		statusHandler StreamFilterStatusHandler) api.StreamFilterStatus

	// invoke all access log.
	Log(ctx context.Context, reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo)

	// destroy the sender filter chain and receiver filter chain.
	OnDestroy()
}

// StreamReceiverFilterWithPhase combines the StreamReceiverFilter with its Phase.
type StreamReceiverFilterWithPhase interface {
	api.StreamReceiverFilter
	ValidatePhase(phase api.ReceiverFilterPhase) bool
}

// StreamReceiverFilterWithPhaseImpl is the default implementation of StreamReceiverFilterWithPhase.
type StreamReceiverFilterWithPhaseImpl struct {
	api.StreamReceiverFilter
	phase api.ReceiverFilterPhase
}

// NewStreamReceiverFilterWithPhase returns a StreamReceiverFilterWithPhaseImpl struct..
func NewStreamReceiverFilterWithPhase(
	f api.StreamReceiverFilter, p api.ReceiverFilterPhase) *StreamReceiverFilterWithPhaseImpl {
	return &StreamReceiverFilterWithPhaseImpl{
		StreamReceiverFilter: f,
		phase:                p,
	}
}

// ValidatePhase checks the current phase.
func (s *StreamReceiverFilterWithPhaseImpl) ValidatePhase(phase api.ReceiverFilterPhase) bool {
	return s.phase == phase
}

// StreamSenderFilterWithPhase combines the StreamSenderFilter which its Phase.
type StreamSenderFilterWithPhase interface {
	api.StreamSenderFilter
	ValidatePhase(phase api.SenderFilterPhase) bool
}

// StreamSenderFilterWithPhaseImpl is default implementation of StreamSenderFilterWithPhase.
type StreamSenderFilterWithPhaseImpl struct {
	api.StreamSenderFilter
	phase api.SenderFilterPhase
}

// NewStreamSenderFilterWithPhaseImpl returns a new StreamSenderFilterWithPhaseImpl.
func NewStreamSenderFilterWithPhase(f api.StreamSenderFilter, p api.SenderFilterPhase) *StreamSenderFilterWithPhaseImpl {
	return &StreamSenderFilterWithPhaseImpl{
		StreamSenderFilter: f,
		phase:              p,
	}
}

// ValidatePhase checks the current phase.
func (s *StreamSenderFilterWithPhaseImpl) ValidatePhase(phase api.SenderFilterPhase) bool {
	return s.phase == phase
}

// DefaultStreamFilterManagerImpl is default implementation of the StreamFilterManager.
type DefaultStreamFilterManagerImpl struct {
	senderFilters      []StreamSenderFilterWithPhase
	senderFiltersIndex int

	receiverFilters      []StreamReceiverFilterWithPhase
	receiverFiltersIndex int

	streamAccessLogs []api.AccessLog
}

// AddStreamSenderFilter registers senderFilters.
func (d *DefaultStreamFilterManagerImpl) AddStreamSenderFilter(filter api.StreamSenderFilter, p api.SenderFilterPhase) {
	f := NewStreamSenderFilterWithPhase(filter, p)
	d.senderFilters = append(d.senderFilters, f)
}

// AddStreamReceiverFilter registers receiver filters.
func (d *DefaultStreamFilterManagerImpl) AddStreamReceiverFilter(filter api.StreamReceiverFilter, p api.ReceiverFilterPhase) {
	f := NewStreamReceiverFilterWithPhase(filter, p)
	d.receiverFilters = append(d.receiverFilters, f)
}

// AddStreamAccessLog registers access logger.
func (d *DefaultStreamFilterManagerImpl) AddStreamAccessLog(accessLog api.AccessLog) {
	d.streamAccessLogs = append(d.streamAccessLogs, accessLog)
}

// RunReceiverFilter invokes the receiver filter chain.
func (d *DefaultStreamFilterManagerImpl) RunReceiverFilter(ctx context.Context, phase api.ReceiverFilterPhase,
	headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
	statusHandler StreamFilterStatusHandler) (filterStatus api.StreamFilterStatus) {
	if statusHandler == nil {
		statusHandler = DefaultStreamFilterStatusHandler
	}

	filterStatus = api.StreamFilterContinue

	for ; d.receiverFiltersIndex < len(d.receiverFilters); d.receiverFiltersIndex++ {
		filter := d.receiverFilters[d.receiverFiltersIndex]
		if !filter.ValidatePhase(phase) {
			continue
		}

		filterStatus = filter.OnReceive(ctx, headers, data, trailers)

		chainStatus := statusHandler(filterStatus)
		switch chainStatus {
		case StreamFilterChainContinue:
			continue
		case StreamFilterChainStop:
			return
		case StreamFilterChainReset:
			d.receiverFiltersIndex = 0
			return
		default:
			continue
		}
	}

	d.receiverFiltersIndex = 0

	return
}

// RunSenderFilter invokes the sender filter chain.
func (d *DefaultStreamFilterManagerImpl) RunSenderFilter(ctx context.Context, phase api.SenderFilterPhase,
	headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
	statusHandler StreamFilterStatusHandler) (filterStatus api.StreamFilterStatus) {
	if statusHandler == nil {
		statusHandler = DefaultStreamFilterStatusHandler
	}

	filterStatus = api.StreamFilterContinue

	for ; d.senderFiltersIndex < len(d.senderFilters); d.senderFiltersIndex++ {
		filter := d.senderFilters[d.senderFiltersIndex]
		if !filter.ValidatePhase(phase) {
			continue
		}

		filterStatus = filter.Append(ctx, headers, data, trailers)

		chainStatus := statusHandler(filterStatus)
		switch chainStatus {
		case StreamFilterChainContinue:
			continue
		case StreamFilterChainStop:
			return
		case StreamFilterChainReset:
			d.receiverFiltersIndex = 0
			return
		default:
			continue
		}
	}

	d.senderFiltersIndex = 0

	return
}

// Log invokes all access loggers.
func (d *DefaultStreamFilterManagerImpl) Log(ctx context.Context,
	reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	for _, l := range d.streamAccessLogs {
		l.Log(ctx, reqHeaders, respHeaders, requestInfo)
	}
}

// OnDestroy invokes the destroy callback of both sender filters and receiver filters.
func (d *DefaultStreamFilterManagerImpl) OnDestroy() {
	for _, filter := range d.receiverFilters {
		filter.OnDestroy()
	}

	for _, filter := range d.senderFilters {
		filter.OnDestroy()
	}
}
