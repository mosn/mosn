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

package router

import (
	"errors"
	"fmt"
	"sync"

	"mosn.io/mosn/pkg/admin/store"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

type RoutersWrapper struct {
	mux           sync.RWMutex
	routers       types.Routers
	routersConfig *v2.RouterConfiguration
}

func (rw *RoutersWrapper) GetRouters() types.Routers {
	rw.mux.RLock()
	defer rw.mux.RUnlock()

	return rw.routers
}

func (rw *RoutersWrapper) GetRoutersConfig() v2.RouterConfiguration {
	rw.mux.RLock()
	defer rw.mux.RUnlock()
	return *rw.routersConfig
}

// RoutersManager implementation
type routersManagerImpl struct {
	routersWrapperMap sync.Map
}

// AddOrUpdateRouters used to add or update router
func (rm *routersManagerImpl) AddOrUpdateRouters(routerConfig *v2.RouterConfiguration) error {
	if routerConfig == nil {
		log.DefaultLogger.Errorf(RouterLogFormat, "routers_manager", "AddOrUpdateRouters", "error: %v", ErrNilRouterConfig)
		return ErrNilRouterConfig
	}
	if v, ok := rm.routersWrapperMap.Load(routerConfig.RouterConfigName); ok {
		rw, ok := v.(*RoutersWrapper)
		if !ok {
			log.DefaultLogger.Errorf(RouterLogFormat, "routers_manager", "AddOrUpdateRouters", "unexpected object in routers map")
			return ErrUnexpected
		}
		routers, err := NewRouters(routerConfig)
		if err != nil {
			// TODO: the rds maybe call this function with a invalid routers(nil) just like Add
			// so we should ignore the alert
			return err
		}
		rw.mux.Lock()
		rw.routers = routers
		rw.routersConfig = routerConfig
		rw.mux.Unlock()
		log.DefaultLogger.Infof(RouterLogFormat, "routers_manager", "AddOrUpdateRouters", "update router: "+routerConfig.RouterConfigName)
	} else {
		// adds new router
		// if a routerConfig with no routes, it is a valid config
		// we ignore the error when we addsd a new router
		// becasue we may stored a nil routers, which is used in istio "RDS" mode
		routers, _ := NewRouters(routerConfig)
		rm.routersWrapperMap.Store(routerConfig.RouterConfigName, &RoutersWrapper{
			routers:       routers,
			routersConfig: routerConfig,
		})
		log.DefaultLogger.Infof(RouterLogFormat, "routers_manager", "AddOrUpdateRouters", "add router: "+routerConfig.RouterConfigName)
	}
	// update admin stored config for admin api dump
	store.SetRouter(routerConfig.RouterConfigName, *routerConfig)
	return nil
}

// GetRouterWrapperByName returns a router wrapper from manager
func (rm *routersManagerImpl) GetRouterWrapperByName(routerConfigName string) types.RouterWrapper {
	if v, ok := rm.routersWrapperMap.Load(routerConfigName); ok {
		rw, ok := v.(*RoutersWrapper)
		if !ok {
			log.DefaultLogger.Errorf(RouterLogFormat, "routers_manager", "GetRouterWrapperByName", "unexpected object in routers map")
			return nil
		}
		return rw
	}
	return nil
}

// AddRoute adds a single router rule into specified virtualhost(by domain)
func (rm *routersManagerImpl) AddRoute(routerConfigName, domain string, route *v2.Router) error {
	if v, ok := rm.routersWrapperMap.Load(routerConfigName); ok {
		rw, ok := v.(*RoutersWrapper)
		if !ok {
			log.DefaultLogger.Errorf(RouterLogFormat, "routers_manager", "AddRoute", "unexpected object in routers map")
			return ErrUnexpected
		}
		rw.mux.Lock()
		defer rw.mux.Unlock()
		routers := rw.routers
		// Stored routers should not be nil when called the api
		if routers == nil {
			log.DefaultLogger.Errorf(RouterLogFormat, "routers_manager", "AddRoute", "error: %v", ErrNoRouters)
			return ErrNoRouters
		}
		cfg := rw.routersConfig
		index := routers.AddRoute(domain, route)
		if index == -1 {
			errMsg := fmt.Sprintf("add route: %s into domain: %s failed", routerConfigName, domain)
			log.DefaultLogger.Errorf(RouterLogFormat, "routers_manager", "AddRoute", errMsg)
			return errors.New(errMsg)
		}
		// modify config
		routersCfg := cfg.VirtualHosts[index].Routers
		routersCfg = append(routersCfg, *route)
		cfg.VirtualHosts[index].Routers = routersCfg
		rw.routersConfig = cfg
		store.SetRouter(routerConfigName, *cfg)
	}
	return nil
}

// RemoceAllRoutes clear all of the specified virtualhost's routes
func (rm *routersManagerImpl) RemoveAllRoutes(routerConfigName, domain string) error {
	if v, ok := rm.routersWrapperMap.Load(routerConfigName); ok {
		rw, ok := v.(*RoutersWrapper)
		if !ok {
			log.DefaultLogger.Errorf(RouterLogFormat, "routers_manager", "RemoveAllRoutes", "unexpected object in routers map")
			return ErrUnexpected
		}
		rw.mux.Lock()
		defer rw.mux.Unlock()
		routers := rw.routers
		// Stored routers should not be nil when called the api
		if routers == nil {
			log.DefaultLogger.Errorf(RouterLogFormat, "routers_manager", "RemoveAllRoutes", "error:%v", ErrNoRouters)
			return ErrNoRouters
		}
		cfg := rw.routersConfig
		index := routers.RemoveAllRoutes(domain)
		if index == -1 {
			errMsg := fmt.Sprintf("clear route: %s in domain: %s failed", routerConfigName, domain)
			log.DefaultLogger.Errorf(RouterLogFormat, "routers_manager", "RemoveAllRoutes", errMsg)
			return errors.New(errMsg)
		}
		// modify config
		routersCfg := cfg.VirtualHosts[index].Routers
		cfg.VirtualHosts[index].Routers = routersCfg[:0]
		rw.routersConfig = cfg
		store.SetRouter(routerConfigName, *cfg)
	}
	return nil
}

var (
	singletonMutex         sync.Mutex
	routersManagerInstance *routersManagerImpl
)

// compatible
func GetRoutersMangerInstance() types.RouterManager {
	return NewRouterManager()
}

func NewRouterManager() types.RouterManager {
	singletonMutex.Lock()
	defer singletonMutex.Unlock()
	if routersManagerInstance != nil {
		return routersManagerInstance
	}

	routersManagerInstance = &routersManagerImpl{
		routersWrapperMap: sync.Map{},
	}

	return routersManagerInstance
}
