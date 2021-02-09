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

var (
	// ErrUnexpected indicate unexpected object type in sync.Map.
	ErrUnexpected = errors.New("unexpected object in map")
	// ErrInvalidKey indicate invalid stream filter config name
	ErrInvalidKey = errors.New("invalid key")
)

var streamFilterManagerInstance StreamFilterManager = &StreamFilterManagerImpl{}

// GetStreamFilterManager return a global singleton of StreamFilterManager.
func GetStreamFilterManager() StreamFilterManager {
	return streamFilterManagerInstance
}

// StreamFilterManager manager the config of all StreamFilterChainFactorys,
// each StreamFilterChainFactory is bound to a key, which is the listenerName by now.
type StreamFilterManager interface {

	// AddOrUpdateStreamFilterConfig map the key to streamFilter chain config.
	AddOrUpdateStreamFilterConfig(key string, config StreamFiltersConfig) error

	// GetStreamFilterFactory return StreamFilterFactory indexed by key.
	GetStreamFilterFactory(key string) StreamFilterFactory
}

// StreamFilterManagerImpl is an implementation of interface StreamFilterManager.
type StreamFilterManagerImpl struct {
	streamFilterChainMap sync.Map
}

// AddOrUpdateStreamFilterConfig map the key to streamFilter chain config.
func (s *StreamFilterManagerImpl) AddOrUpdateStreamFilterConfig(key string, config StreamFiltersConfig) error {
	if key == "" {
		log.DefaultLogger.Errorf("[streamfilter] AddOrUpdateStreamFilterConfig invalid key: %v", key)
		return ErrInvalidKey
	}

	if v, ok := s.streamFilterChainMap.Load(key); ok {
		factory, ok := v.(StreamFilterFactory)
		if !ok {
			log.DefaultLogger.Errorf("[streamfilter] AddOrUpdateStreamFilterConfig unexpected object in map")
			return ErrUnexpected
		}

		factory.UpdateFactory(config)

		log.DefaultLogger.Infof("[streamfilter] AddOrUpdateStreamFilterConfig update filter chain key: %v", key)
	} else {
		factory := NewStreamFilterFactory(config)
		s.streamFilterChainMap.LoadOrStore(key, factory)
		log.DefaultLogger.Infof("[streamfilter] AddOrUpdateStreamFilterConfig add filter chain key: %v", key)
	}
	return nil
}

// GetStreamFilterFactory return StreamFilterFactory indexed by key.
func (s *StreamFilterManagerImpl) GetStreamFilterFactory(key string) StreamFilterFactory {
	if v, ok := s.streamFilterChainMap.Load(key); ok {
		factoryWrapper, ok := v.(StreamFilterFactory)
		if !ok {
			log.DefaultLogger.Errorf("[streamfilter] GetStreamFilterFactory unexpected object in map")
			return nil
		}

		return factoryWrapper
	}

	log.DefaultLogger.Errorf("[streamfilter] GetStreamFilterFactory stream filter factory not found in map, name: %v", key)
	return nil
}
