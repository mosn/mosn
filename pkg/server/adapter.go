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

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

var listenerAdapterInstance *ListenerAdapter

type ListenerAdapter struct {
	connHandlerMap     map[string]types.ConnectionHandler // key is server's name
	defaultConnHandler types.ConnectionHandler
	defaultName        string
}

// todo consider to use singleton
func initListenerAdapterInstance(name string, connHandler types.ConnectionHandler) {
	if listenerAdapterInstance == nil {
		listenerAdapterInstance = &ListenerAdapter{
			connHandlerMap: make(map[string]types.ConnectionHandler),
			// we set the first handler as the default handler
			// the handler name should be keeped, so if the handler changed, the default handler changed too.
			defaultName: name,
		}
	}

	// if the handler's name is same as default, the default handler changed too
	if name == listenerAdapterInstance.defaultName {
		listenerAdapterInstance.defaultConnHandler = connHandler
	}

	listenerAdapterInstance.connHandlerMap[name] = connHandler
	log.DefaultLogger.Debugf("[server] [init] add server conn handler, server name = %s", name)
}

func (adapter *ListenerAdapter) findHandler(serverName string) types.ConnectionHandler {
	if adapter == nil {
		return nil
	}
	if serverName != "" {
		// if serverName is not exists, return nil
		return adapter.connHandlerMap[serverName]
	}
	return adapter.defaultConnHandler
}

// FindListenerByName
func (adapter *ListenerAdapter) FindListenerByName(serverName string, listenerName string) types.Listener {
	connHandler := adapter.findHandler(serverName)
	if connHandler == nil {
		return nil
	}
	return connHandler.FindListenerByName(listenerName)
}

func GetListenerAdapterInstance() *ListenerAdapter {
	return listenerAdapterInstance
}

// ResetAdapter only used in test/debug mode
func ResetAdapter() {
	log.DefaultLogger.Infof("[server] adapter reset, only expected in test/debug mode")
	listenerAdapterInstance = nil
}

// AddOrUpdateListener used to:
// Add and start listener when listener doesn't exist
// Update listener when listener already exist
func (adapter *ListenerAdapter) AddOrUpdateListener(serverName string, lc *v2.Listener) error {

	connHandler := adapter.findHandler(serverName)
	if connHandler == nil {
		return fmt.Errorf("AddOrUpdateListener error, servername = %s not found", serverName)
	}

	listener, err := connHandler.AddOrUpdateListener(lc)
	if err != nil {
		return fmt.Errorf("connHandler.AddOrUpdateListener called error: %s", err.Error())
	}

	if listener == nil {
		return nil
	}

	if al, ok := listener.(*activeListener); ok {
		if !al.updatedLabel {
			// start listener if this is new
			utils.GoWithRecover(func() {
				al.listener.Start(nil, false)
			}, nil)
		}

		return nil
	}

	return fmt.Errorf("AddOrUpdateListener Error, got listener is not activeListener")
}

func (adapter *ListenerAdapter) DeleteListener(serverName string, listenerName string) error {
	connHandler := adapter.findHandler(serverName)
	if connHandler == nil {
		return fmt.Errorf("DeleteListener error, servername = %s not found", serverName)
	}

	// stop listener first
	if err := connHandler.StopListener(nil, listenerName, true); err != nil {
		return err
	}

	// then remove it from array
	connHandler.RemoveListeners(listenerName)
	return nil
}

func (adapter *ListenerAdapter) UpdateListenerTLS(serverName string, listenerName string, inspector bool, tlsConfigs []v2.TLSConfig) error {
	connHandler := adapter.findHandler(serverName)
	if connHandler == nil {
		return fmt.Errorf("UpdateListenerTLS error, servername = %s not found", serverName)
	}

	if ln := connHandler.FindListenerByName(listenerName); ln != nil {
		cfg := *ln.Config() // should clone a config
		cfg.Inspector = inspector
		cfg.FilterChains = []v2.FilterChain{
			{
				FilterChainConfig: v2.FilterChainConfig{
					FilterChainMatch: cfg.FilterChains[0].FilterChainMatch,
					Filters:          cfg.FilterChains[0].Filters,
					TLSConfigs:       tlsConfigs,
				},
				TLSContexts: tlsConfigs,
			},
		}

		if _, err := connHandler.AddOrUpdateListener(&cfg); err != nil {
			return fmt.Errorf("connHandler.UpdateListenerTLS called error, server:%s, error: %s", serverName, err.Error())
		}
		return nil
	}
	return fmt.Errorf("listener %s is not found", listenerName)

}
