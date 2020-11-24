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
	"sync/atomic"

	"mosn.io/api"
)

// StreamFilterFactory combine the StreamFilterChainFactory (type: []api.StreamFilterChainFactory)
// with its config (type: []v2.Filter).
type StreamFilterFactory interface {

	// CreateFilterChain call 'CreateFilterChain' method for each api.StreamFilterChainFactory.
	CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks)

	// GetConfig return raw config of the StreamFilterChainFactory.
	GetConfig() StreamFiltersConfig
}

// NewStreamFilterFactory return a StreamFilterFactoryImpl struct.
func NewStreamFilterFactory(config StreamFiltersConfig) StreamFilterFactory {
	wrapper := &StreamFilterFactoryImpl{
		config: config,
	}
	wrapper.factories.Store(GetStreamFilters(config))
	return wrapper
}

// StreamFilterFactoryImpl is an implementation of interface StreamFilterFactory.
type StreamFilterFactoryImpl struct {
	mux       sync.Mutex
	factories atomic.Value // actual type: []api.StreamFilterChainFactory
	config    StreamFiltersConfig
}

// CreateFilterChain call 'CreateFilterChain' method for each api.StreamFilterChainFactory.
func (s *StreamFilterFactoryImpl) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	factories := s.factories.Load().([]api.StreamFilterChainFactory)
	for _, factory := range factories {
		factory.CreateFilterChain(context, callbacks)
	}
}

// GetConfig return raw config of the StreamFilterChainFactory.
func (s *StreamFilterFactoryImpl) GetConfig() StreamFiltersConfig {
	return s.config
}