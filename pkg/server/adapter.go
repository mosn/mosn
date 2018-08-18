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

package server

import (
	"fmt"
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
	"sync"
)

var listenerAdapterInstance *ListenerAdapter
var once sync.Once

type ListenerAdapter struct {
	connHandler types.ConnectionHandler
}

func InitListenerAdapterInstance(connHandler types.ConnectionHandler) {
	once.Do(func() {
		listenerAdapterInstance = &ListenerAdapter{
			connHandler: connHandler,
		}
	})
}

func GetListenerAdapterInstance() *ListenerAdapter {
	return listenerAdapterInstance
}

// AddOrUpdateListener used to:
// Add and start listener when listener doesn't exist
// Update listener when listener already exist
func (adapter *ListenerAdapter) AddOrUpdateListener(lc *v2.ListenerConfig, networkFiltersFactory types.NetworkFilterChainFactory,
	streamFiltersFactories []types.StreamFilterChainFactory) error {
	listener := adapter.connHandler.AddOrUpdateListener(lc, networkFiltersFactory, streamFiltersFactories)
	if al, ok := listener.(*activeListener); ok {
		if !al.updatedLabel {
			// start listener if this is new
			go al.listener.Start(nil)
		}

		return nil
	}

	return fmt.Errorf("AddOrUpdateListener Error")
}

func (adapter *ListenerAdapter) StopListener(name string) error {
	return adapter.connHandler.StopListener(nil, name, false)
}

func (adapter *ListenerAdapter) DeleteListener(lc v2.ListenerConfig) error {
	// stop listener first
	if err := adapter.connHandler.StopListener(nil, lc.Name, true); err != nil {
		return err
	}

	// then remove it from array
	adapter.connHandler.RemoveListeners(lc.Name)
	return nil
}
