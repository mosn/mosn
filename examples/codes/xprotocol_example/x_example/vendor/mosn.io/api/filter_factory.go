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

package api

import "fmt"

// ListenerFilterFactoryCreator creates a ListenerFilterChainFactory according to config
type ListenerFilterFactoryCreator func(config map[string]interface{}) (ListenerFilterChainFactory, error)

// StreamFilterFactoryCreator creates a StreamFilterChainFactory according to config
type StreamFilterFactoryCreator func(config map[string]interface{}) (StreamFilterChainFactory, error)

// NetworkFilterFactoryCreator creates a NetworkFilterChainFactory according to config
type NetworkFilterFactoryCreator func(config map[string]interface{}) (NetworkFilterChainFactory, error)

var creatorListenerFactory map[string]ListenerFilterFactoryCreator
var creatorStreamFactory map[string]StreamFilterFactoryCreator
var creatorNetworkFactory map[string]NetworkFilterFactoryCreator

func init() {
	creatorListenerFactory = make(map[string]ListenerFilterFactoryCreator)
	creatorStreamFactory = make(map[string]StreamFilterFactoryCreator)
	creatorNetworkFactory = make(map[string]NetworkFilterFactoryCreator)
}

// RegisterListener registers the filterType as ListenerFilterFactoryCreator
func RegisterListener(filterType string, creator ListenerFilterFactoryCreator) {
	creatorListenerFactory[filterType] = creator
}

// RegisterStream registers the filterType as StreamFilterFactoryCreator
func RegisterStream(filterType string, creator StreamFilterFactoryCreator) {
	creatorStreamFactory[filterType] = creator
}

// RegisterNetwork registers the filterType as  NetworkFilterFactoryCreator
func RegisterNetwork(filterType string, creator NetworkFilterFactoryCreator) {
	creatorNetworkFactory[filterType] = creator
}

// CreateListenerFilterChainFactory creates a ListenerFilterChainFactory according to filterType
func CreateListenerFilterChainFactory(filterType string, config map[string]interface{}) (ListenerFilterChainFactory, error) {
	if cf, ok := creatorListenerFactory[filterType]; ok {
		lfcf, err := cf(config)
		if err != nil {
			return nil, fmt.Errorf("create listener filter chain factory failed: %v", err)
		}
		return lfcf, nil
	}
	return nil, fmt.Errorf("unsupported listener filter type: %v", filterType)
}

// CreateStreamFilterChainFactory creates a StreamFilterChainFactory according to filterType
func CreateStreamFilterChainFactory(filterType string, config map[string]interface{}) (StreamFilterChainFactory, error) {
	if cf, ok := creatorStreamFactory[filterType]; ok {
		sfcf, err := cf(config)
		if err != nil {
			return nil, fmt.Errorf("create stream filter chain factory failed: %v", err)
		}
		return sfcf, nil
	}
	return nil, fmt.Errorf("unsupported stream filter type: %v", filterType)
}

// CreateNetworkFilterChainFactory creates a NetworkFilterChainFactory according to filterType
func CreateNetworkFilterChainFactory(filterType string, config map[string]interface{}) (NetworkFilterChainFactory, error) {
	if cf, ok := creatorNetworkFactory[filterType]; ok {
		nfcf, err := cf(config)
		if err != nil {
			return nil, fmt.Errorf("create network filter chain factory failed: %v", err)
		}
		return nfcf, nil
	}
	return nil, fmt.Errorf("unsupported network filter type: %v", filterType)
}
