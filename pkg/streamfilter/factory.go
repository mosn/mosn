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
	"sync/atomic"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

// StreamFilterFactory is a wrapper of type []api.StreamFilterChainFactory.
type StreamFilterFactory interface {

	// CreateFilterChain call 'CreateFilterChain' method for each api.StreamFilterChainFactory.
	CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks)

	// UpdateFactory update factory according to config.
	UpdateFactory(config StreamFiltersConfig)
}

// NewStreamFilterFactory return a StreamFilterFactoryImpl struct.
func NewStreamFilterFactory(config StreamFiltersConfig) StreamFilterFactory {
	factory := &StreamFilterFactoryImpl{}

	sff := createStreamFilterFactoryFromConfig(config)
	factory.factories.Store(sff)

	return factory
}

// StreamFilterFactoryImpl is an implementation of interface StreamFilterFactory.
type StreamFilterFactoryImpl struct {
	factories atomic.Value // actual type: []api.StreamFilterChainFactory
}

// CreateFilterChain call 'CreateFilterChain' method for each api.StreamFilterChainFactory.
func (s *StreamFilterFactoryImpl) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	factories, ok := s.factories.Load().([]api.StreamFilterChainFactory)
	if ok {
		for _, factory := range factories {
			factory.CreateFilterChain(context, callbacks)
		}
	} else {
		log.DefaultLogger.Errorf("[streamfilter] CreateFilterChain unexpected object type in atomic.Value")
	}
}

// UpdateFactory update factory according to config.
func (s *StreamFilterFactoryImpl) UpdateFactory(config StreamFiltersConfig) {
	sff := createStreamFilterFactoryFromConfig(config)
	s.factories.Store(sff)
}
