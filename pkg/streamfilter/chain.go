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
	"sync"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// StreamReceiverFilterStatusHandler allow users to deal with the receiver filter status.
type StreamReceiverFilterStatusHandler func(phase api.ReceiverFilterPhase, status api.StreamFilterStatus)

// StreamSenderFilterStatusHandler allow users to deal with the sender filter status.
type StreamSenderFilterStatusHandler func(phase api.SenderFilterPhase, status api.StreamFilterStatus)

// StreamReceiverFilterIteratorHandler is used to implement iterator handler for receive filter.
type StreamReceiverFilterIteratorHandler func(context.Context, types.HeaderMap, types.IoBuffer, types.HeaderMap, api.StreamReceiverFilter)

// StreamSenderFilterIteratorHandler is used to implement iterator handler for send filter.
type StreamSenderFilterIteratorHandler func(context.Context, types.HeaderMap, types.IoBuffer, types.HeaderMap, api.StreamSenderFilter)

// StreamFilterChain manages the lifecycle of streamFilters.
type StreamFilterChain interface {
	// register StreamSenderFilter, StreamReceiverFilter and AccessLog.
	api.StreamFilterChainFactoryCallbacks

	// SetReceiveFilterHandler set filter handler for each filter in this chain
	SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler)

	// SetSenderFilterHandler set filter handler for each filter in this chain
	SetSenderFilterHandler(handler api.StreamSenderFilterHandler)

	// RangeReceiverFilter range all receive filter.
	RangeReceiverFilter(context.Context, types.HeaderMap, types.IoBuffer,
		types.HeaderMap, StreamReceiverFilterIteratorHandler)
	// RangeSenderFilter range all send filter.
	RangeSenderFilter(context.Context, types.HeaderMap, types.IoBuffer,
		types.HeaderMap, StreamSenderFilterIteratorHandler)

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
	// use two slice to avoid the allocation of small object
	senderFilters      []api.StreamSenderFilter
	senderFiltersPhase []api.SenderFilterPhase
	senderFiltersIndex int

	// use two slice to avoid the allocation of small object
	receiverFilters      []api.StreamReceiverFilter
	receiverFiltersPhase []api.ReceiverFilterPhase
	receiverFiltersIndex int

	streamAccessLogs []api.AccessLog
}

var streamFilterChainPool = sync.Pool{
	New: func() interface{} {
		return &DefaultStreamFilterChainImpl{}
	},
}

// GetDefaultStreamFilterChain return a pool-cached DefaultStreamFilterChainImpl.
func GetDefaultStreamFilterChain() *DefaultStreamFilterChainImpl {
	chain := streamFilterChainPool.Get().(*DefaultStreamFilterChainImpl)
	return chain
}

// PutStreamFilterChain reset DefaultStreamFilterChainImpl and return it to pool.
func PutStreamFilterChain(chain *DefaultStreamFilterChainImpl) {
	chain.senderFilters = chain.senderFilters[:0]
	chain.senderFiltersPhase = chain.senderFiltersPhase[:0]
	chain.senderFiltersIndex = 0

	chain.receiverFilters = chain.receiverFilters[:0]
	chain.receiverFiltersPhase = chain.receiverFiltersPhase[:0]
	chain.receiverFiltersIndex = 0

	chain.streamAccessLogs = chain.streamAccessLogs[:0]

	streamFilterChainPool.Put(chain)
}

// AddStreamSenderFilter registers senderFilters.
func (d *DefaultStreamFilterChainImpl) AddStreamSenderFilter(filter api.StreamSenderFilter, p api.SenderFilterPhase) {
	d.senderFilters = append(d.senderFilters, filter)
	d.senderFiltersPhase = append(d.senderFiltersPhase, p)
}

// SetSenderFilterHandler set filter handler for each filter in this chain
func (d *DefaultStreamFilterChainImpl) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	if handler == nil {
		return
	}

	for _, filter := range d.senderFilters {
		filter.SetSenderFilterHandler(handler)
	}
}

// AddStreamReceiverFilter registers receiver filters.
func (d *DefaultStreamFilterChainImpl) AddStreamReceiverFilter(filter api.StreamReceiverFilter, p api.ReceiverFilterPhase) {
	d.receiverFilters = append(d.receiverFilters, filter)
	d.receiverFiltersPhase = append(d.receiverFiltersPhase, p)
}

// SetReceiveFilterHandler set filter handler for each filter in this chain
func (d *DefaultStreamFilterChainImpl) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	if handler == nil {
		return
	}

	for _, filter := range d.receiverFilters {
		filter.SetReceiveFilterHandler(handler)
	}
}

// RangeReceiverFilter range receive filter.
func (d *DefaultStreamFilterChainImpl) RangeReceiverFilter(ctx context.Context, headers types.HeaderMap,
	data types.IoBuffer, trailers types.HeaderMap, handler StreamReceiverFilterIteratorHandler) {
	for _, filter := range d.receiverFilters {
		handler(ctx, headers, data, trailers, filter)
	}
}

// RangeSenderFilter range send filter.
func (d *DefaultStreamFilterChainImpl) RangeSenderFilter(ctx context.Context, headers types.HeaderMap,
	data types.IoBuffer, trailers types.HeaderMap, handler StreamSenderFilterIteratorHandler) {
	for _, filter := range d.senderFilters {
		handler(ctx, headers, data, trailers, filter)
	}
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
		p := d.receiverFiltersPhase[d.receiverFiltersIndex]
		if phase != p {
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
		p := d.senderFiltersPhase[d.senderFiltersIndex]
		if phase != p {
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
