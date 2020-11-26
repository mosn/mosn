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

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// StreamReceiverFilterStatusHandler allow users to deal with the receiver filter status.
type StreamReceiverFilterStatusHandler func(phase api.ReceiverFilterPhase, status api.StreamFilterStatus)

// StreamSenderFilterStatusHandler allow users to deal with the sender filter status.
type StreamSenderFilterStatusHandler func(phase api.SenderFilterPhase, status api.StreamFilterStatus)

// StreamFilterChain manages the lifecycle of streamFilters.
type StreamFilterChain interface {
	// register StreamSenderFilter, StreamReceiverFilter and AccessLog.
	api.StreamFilterChainFactoryCallbacks

	// invoke the receiver filter chain.
	RunReceiverFilter(ctx context.Context, phase api.ReceiverFilterPhase,
		headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
		statusHandler StreamReceiverFilterStatusHandler) api.StreamFilterStatus

	// invoke the sender filter chain.
	RunSenderFilter(ctx context.Context, phase api.SenderFilterPhase,
		headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
		statusHandler StreamSenderFilterStatusHandler) api.StreamFilterStatus

	// invoke all access log.
	Log(ctx context.Context, reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo)

	// destroy the sender filter chain and receiver filter chain.
	OnDestroy()
}

// DefaultStreamFilterChainImpl is default implementation of the StreamFilterChain.
type DefaultStreamFilterChainImpl struct {
	senderFilters      []StreamSenderFilterWithPhase
	senderFiltersIndex int

	receiverFilters      []StreamReceiverFilterWithPhase
	receiverFiltersIndex int

	streamAccessLogs []api.AccessLog
}

// AddStreamSenderFilter registers senderFilters.
func (d *DefaultStreamFilterChainImpl) AddStreamSenderFilter(filter api.StreamSenderFilter, p api.SenderFilterPhase) {
	f := NewStreamSenderFilterWithPhase(filter, p)
	d.senderFilters = append(d.senderFilters, f)
}

// AddStreamReceiverFilter registers receiver filters.
func (d *DefaultStreamFilterChainImpl) AddStreamReceiverFilter(filter api.StreamReceiverFilter, p api.ReceiverFilterPhase) {
	f := NewStreamReceiverFilterWithPhase(filter, p)
	d.receiverFilters = append(d.receiverFilters, f)
}

// AddStreamAccessLog registers access logger.
func (d *DefaultStreamFilterChainImpl) AddStreamAccessLog(accessLog api.AccessLog) {
	d.streamAccessLogs = append(d.streamAccessLogs, accessLog)
}

// RunReceiverFilter invokes the receiver filter chain.
func (d *DefaultStreamFilterChainImpl) RunReceiverFilter(ctx context.Context, phase api.ReceiverFilterPhase,
	headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
	statusHandler StreamReceiverFilterStatusHandler) (filterStatus api.StreamFilterStatus) {
	filterStatus = api.StreamFilterContinue

	for ; d.receiverFiltersIndex < len(d.receiverFilters); d.receiverFiltersIndex++ {
		filter := d.receiverFilters[d.receiverFiltersIndex]
		if phase != filter.GetPhase() {
			continue
		}

		filterStatus = filter.OnReceive(ctx, headers, data, trailers)

		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("DefaultStreamFilterChainImpl.RunReceiverFilter phase: %v, index: %v, status: %v",
				phase, d.receiverFiltersIndex, filterStatus)
		}

		if statusHandler != nil {
			statusHandler(phase, filterStatus)
		}

		switch filterStatus {
		case api.StreamFilterContinue:
			continue
		case api.StreamFilterStop, api.StreamFiltertermination:
			d.receiverFiltersIndex = 0
			return
		case api.StreamFilterReMatchRoute, api.StreamFilterReChooseHost:
			return
		}
	}

	d.receiverFiltersIndex = 0

	return
}

// RunSenderFilter invokes the sender filter chain.
func (d *DefaultStreamFilterChainImpl) RunSenderFilter(ctx context.Context, phase api.SenderFilterPhase,
	headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
	statusHandler StreamSenderFilterStatusHandler) (filterStatus api.StreamFilterStatus) {
	filterStatus = api.StreamFilterContinue

	for ; d.senderFiltersIndex < len(d.senderFilters); d.senderFiltersIndex++ {
		filter := d.senderFilters[d.senderFiltersIndex]
		if phase != filter.GetPhase() {
			continue
		}

		filterStatus = filter.Append(ctx, headers, data, trailers)

		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("DefaultStreamFilterChainImpl.RunSenderFilter, phase: %v, index: %v, status: %v",
				phase, d.senderFiltersIndex, filterStatus)
		}

		if statusHandler != nil {
			statusHandler(phase, filterStatus)
		}

		switch filterStatus {
		case api.StreamFilterContinue:
			continue
		case api.StreamFilterStop, api.StreamFiltertermination:
			d.senderFiltersIndex = 0
			return
		case api.StreamFilterReMatchRoute, api.StreamFilterReChooseHost:
			log.DefaultLogger.Errorf("DefaultStreamFilterChainImpl.RunSenderFilter filter return invalid status: %v", filterStatus)
			d.senderFiltersIndex = 0
			return
		}
	}

	d.senderFiltersIndex = 0

	return
}

// Log invokes all access loggers.
func (d *DefaultStreamFilterChainImpl) Log(ctx context.Context,
	reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	for _, l := range d.streamAccessLogs {
		l.Log(ctx, reqHeaders, respHeaders, requestInfo)
	}
}

// OnDestroy invokes the destroy callback of both sender filters and receiver filters.
func (d *DefaultStreamFilterChainImpl) OnDestroy() {
	for _, filter := range d.receiverFilters {
		filter.OnDestroy()
	}

	for _, filter := range d.senderFilters {
		filter.OnDestroy()
	}
}
