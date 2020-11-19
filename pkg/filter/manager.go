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
	"errors"
	"sync"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

type StreamFilterChainConfig = []v2.Filter

// StreamFilterChainFactoryWrapper combine the StreamFilterChainFactory (type: []api.StreamFilterChainFactory)
// with its config (type: []v2.Filter).
type StreamFilterChainFactoryWrapper interface {

	// CreateFilterChain call 'CreateFilterChain' method for each api.StreamFilterChainFactory.
	CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks)

	// GetConfig return raw config of the StreamFilterChainFactory.
	GetConfig() StreamFilterChainConfig
}

// NewStreamFilterChainFactoryWrapper return a StreamFilterChainFactoryWrapperImpl struct.
func NewStreamFilterChainFactoryWrapper(config StreamFilterChainConfig) StreamFilterChainFactoryWrapper {
	return &StreamFilterChainFactoryWrapperImpl{
		factories: configmanager.GetStreamFilters(config),
		config:    config,
	}
}

// StreamFilterChainFactoryWrapperImpl is implementation of interface StreamFilterChainFactoryWrapper.
type StreamFilterChainFactoryWrapperImpl struct {
	mux       sync.Mutex
	factories []api.StreamFilterChainFactory
	config    StreamFilterChainConfig
}

// CreateFilterChain call 'CreateFilterChain' method for each api.StreamFilterChainFactory.
func (s *StreamFilterChainFactoryWrapperImpl) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	for _, factory := range s.factories {
		factory.CreateFilterChain(context, callbacks)
	}
}

// GetConfig return raw config of the StreamFilterChainFactory.
func (s *StreamFilterChainFactoryWrapperImpl) GetConfig() StreamFilterChainConfig {
	return s.config
}

// StreamFilterChainFactoryManager manager the config of all StreamFilterChainFactorys
// each StreamFilterChainFactory is bound to a key, which is the listenerName by now.
type StreamFilterChainFactoryManager interface {

	// AddOrUpdateStreamFilterChain map the key to streamFilter chain config
	AddOrUpdateStreamFilterChain(key string, config StreamFilterChainConfig) error

	// GetStreamFilterChainFactoryByKey return StreamFilterChainFactoryWrapper indexed by key
	GetStreamFilterChainFactoryByKey(key string) StreamFilterChainFactoryWrapper
}

var (
	someSingleton                           sync.Mutex
	streamFilterChainFactoryManagerInstance *StreamFilterChainFactoryManagerImpl
)

// GetStreamFilterChainFactoryManager return a singleton of StreamFilterChainFactoryManager
func GetStreamFilterChainFactoryManager() StreamFilterChainFactoryManager {
	someSingleton.Lock()
	defer someSingleton.Unlock()

	if streamFilterChainFactoryManagerInstance == nil {
		streamFilterChainFactoryManagerInstance = &StreamFilterChainFactoryManagerImpl{
			streamFilterChainMap: sync.Map{},
		}
	}

	return streamFilterChainFactoryManagerInstance
}

// StreamFilterChainFactoryManagerImpl is an implementation of interface StreamFilterChainFactoryManager
type StreamFilterChainFactoryManagerImpl struct {
	streamFilterChainMap sync.Map
}

// AddOrUpdateStreamFilterChain map the key to streamFilter chain config
func (s *StreamFilterChainFactoryManagerImpl) AddOrUpdateStreamFilterChain(key string, config StreamFilterChainConfig) error {
	if v, ok := s.streamFilterChainMap.Load(key); ok {
		factoryWrapper, ok := v.(*StreamFilterChainFactoryWrapperImpl)
		if !ok {
			log.DefaultLogger.Errorf("StreamFilterChainFactoryManagerImpl.AddOrUpdateStreamFilterChain unexpected object in map")
			return errors.New("unexpected object in map")
		}
		factories := configmanager.GetStreamFilters(config)
		factoryWrapper.mux.Lock()
		factoryWrapper.factories = factories
		factoryWrapper.config = config
		factoryWrapper.mux.Unlock()
		log.DefaultLogger.Infof("StreamFilterChainFactoryManagerImpl.AddOrUpdateStreamFilterChain update filter chain key: %v", key)
	} else {
		factoryWrapper := NewStreamFilterChainFactoryWrapper(config)
		s.streamFilterChainMap.Store(key, factoryWrapper)
		log.DefaultLogger.Infof("StreamFilterChainFactoryManagerImpl.AddOrUpdateStreamFilterChain add filter chain key: %v", key)
	}
	return nil
}

// GetStreamFilterChainFactoryByKey return StreamFilterChainFactoryWrapper indexed by key
func (s *StreamFilterChainFactoryManagerImpl) GetStreamFilterChainFactoryByKey(key string) StreamFilterChainFactoryWrapper {
	if v, ok := s.streamFilterChainMap.Load(key); ok {
		factoryWrapper, ok := v.(StreamFilterChainFactoryWrapper)
		if !ok {
			log.DefaultLogger.Errorf("StreamFilterChainFactoryManagerImpl.GetStreamFilterChainFactoryByKey unexpected object in map")
			return nil
		}
		return factoryWrapper
	}
	return nil
}

// StreamFilterStatusHandler allow users to deal with the filter status.
type StreamFilterStatusHandler func(status api.StreamFilterStatus)

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

	// StreamReceiverFilter interface.
	api.StreamReceiverFilter

	// GetPhase return the working phase of current filter.
	GetPhase() api.ReceiverFilterPhase
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

// StreamSenderFilterWithPhase combines the StreamSenderFilter which its Phase.
type StreamSenderFilterWithPhase interface {

	// StreamSenderFilter interface
	api.StreamSenderFilter

	// GetPhase return the working phase of current filter.
	GetPhase() api.SenderFilterPhase
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
	filterStatus = api.StreamFilterContinue

	for ; d.receiverFiltersIndex < len(d.receiverFilters); d.receiverFiltersIndex++ {
		filter := d.receiverFilters[d.receiverFiltersIndex]
		if phase != filter.GetPhase() {
			continue
		}

		filterStatus = filter.OnReceive(ctx, headers, data, trailers)

		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("DefaultStreamFilterManagerImpl.RunReceiverFilter phase: %v, index: %v, status: %v",
				phase, d.receiverFiltersIndex, filterStatus)
		}

		if statusHandler != nil {
			statusHandler(filterStatus)
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
func (d *DefaultStreamFilterManagerImpl) RunSenderFilter(ctx context.Context, phase api.SenderFilterPhase,
	headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
	statusHandler StreamFilterStatusHandler) (filterStatus api.StreamFilterStatus) {
	filterStatus = api.StreamFilterContinue

	for ; d.senderFiltersIndex < len(d.senderFilters); d.senderFiltersIndex++ {
		filter := d.senderFilters[d.senderFiltersIndex]
		if phase != filter.GetPhase() {
			continue
		}

		filterStatus = filter.Append(ctx, headers, data, trailers)

		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("DefaultStreamFilterManagerImpl.RunSenderFilter, phase: %v, index: %v, status: %v",
				phase, d.senderFiltersIndex, filterStatus)
		}

		if statusHandler != nil {
			statusHandler(filterStatus)
		}

		switch filterStatus {
		case api.StreamFilterContinue:
			continue
		case api.StreamFilterStop, api.StreamFiltertermination:
			d.senderFiltersIndex = 0
			return
		case api.StreamFilterReMatchRoute, api.StreamFilterReChooseHost:
			log.DefaultLogger.Errorf("DefaultStreamFilterManagerImpl.RunSenderFilter filter return invalid status: %v", filterStatus)
			d.senderFiltersIndex = 0
			return
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
