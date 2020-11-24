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
	"errors"
	"sync"

	"mosn.io/mosn/pkg/log"
)

// ErrUnexpected indicate unexpected object type in sync.Map.
var ErrUnexpected = errors.New("unexpected object in map")

// StreamFilterManager manager the config of all StreamFilterChainFactorys,
// each StreamFilterChainFactory is bound to a key, which is the listenerName by now.
type StreamFilterManager interface {

	// AddOrUpdateStreamFilterConfig map the key to streamFilter chain config.
	AddOrUpdateStreamFilterConfig(key string, config StreamFiltersConfig) error

	// GetStreamFilterFactory return StreamFilterFactory indexed by key.
	GetStreamFilterFactory(key string) StreamFilterFactory
}

var (
	streamFilterManagerSingleton sync.Mutex
	streamFilterManagerInstance  *StreamFilterManagerImpl
)

// GetStreamFilterManager return a singleton of StreamFilterManager.
func GetStreamFilterManager() StreamFilterManager {
	streamFilterManagerSingleton.Lock()
	defer streamFilterManagerSingleton.Unlock()

	if streamFilterManagerInstance == nil {
		streamFilterManagerInstance = &StreamFilterManagerImpl{
			streamFilterChainMap: sync.Map{},
		}
	}

	return streamFilterManagerInstance
}

// StreamFilterManagerImpl is an implementation of interface StreamFilterManager.
type StreamFilterManagerImpl struct {
	streamFilterChainMap sync.Map
}

// AddOrUpdateStreamFilterConfig map the key to streamFilter chain config.
func (s *StreamFilterManagerImpl) AddOrUpdateStreamFilterConfig(key string, config StreamFiltersConfig) error {
	if v, ok := s.streamFilterChainMap.Load(key); ok {
		factoryWrapper, ok := v.(*StreamFilterFactoryImpl)
		if !ok {
			log.DefaultLogger.Errorf("StreamFilterManagerImpl.AddOrUpdateStreamFilterConfig unexpected object in map")
			return ErrUnexpected
		}

		factories := GetStreamFilters(config)

		factoryWrapper.mux.Lock()
		factoryWrapper.factories.Store(factories)
		factoryWrapper.config = config
		factoryWrapper.mux.Unlock()

		log.DefaultLogger.Infof("StreamFilterManagerImpl.AddOrUpdateStreamFilterConfig update filter chain key: %v", key)
	} else {
		factoryWrapper := NewStreamFilterFactory(config)
		s.streamFilterChainMap.Store(key, factoryWrapper)
		log.DefaultLogger.Infof("StreamFilterManagerImpl.AddOrUpdateStreamFilterConfig add filter chain key: %v", key)
	}
	return nil
}

// GetStreamFilterFactory return StreamFilterFactory indexed by key.
func (s *StreamFilterManagerImpl) GetStreamFilterFactory(key string) StreamFilterFactory {
	if v, ok := s.streamFilterChainMap.Load(key); ok {
		factoryWrapper, ok := v.(StreamFilterFactory)
		if !ok {
			log.DefaultLogger.Errorf("StreamFilterManagerImpl.GetStreamFilterFactory unexpected object in map")
			return nil
		}

		return factoryWrapper
	}
	return nil
}
