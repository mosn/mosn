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

// "routersMap" in "routersMangerInstance" stored all routers with "RouterConfigureName" as the unique identifier
// when update, update wrapper's routes
// when use, proxy's get wrapper's routers

package router

import (
	"errors"
	"sync"

	"fmt"

	admin "github.com/alipay/sofa-mosn/pkg/admin/store"
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var instanceMutex = sync.Mutex{}
var routersMangerInstance *routersManager

// GetClusterMngAdapterInstance used to get clusterMngAdapterInstance
func GetRoutersMangerInstance() types.RouterManager {
	return routersMangerInstance
}

type routersManager struct {
	routersMap sync.Map // key is router's name used to associated to the listener, value's type is: RoutersWrapper
}

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

func NewRouterManager() types.RouterManager {
	instanceMutex.Lock()
	defer instanceMutex.Unlock()

	if routersMangerInstance != nil {
		return routersMangerInstance
	}

	routersMap := sync.Map{}

	routersMangerInstance = &routersManager{
		routersMap: routersMap,
	}

	return routersMangerInstance
}

// AddOrUpdateRouters used to add or update router
func (rm *routersManager) AddOrUpdateRouters(routerConfig *v2.RouterConfiguration) error {
	if routerConfig == nil {
		errMsg := "AddOrUpdateRouters Error,routerConfig is nil "
		log.DefaultLogger.Errorf(errMsg)
		return fmt.Errorf(errMsg)
	}

	if v, ok := rm.routersMap.Load(routerConfig.RouterConfigName); ok {
		// try to update router
		if primaryRouters, ok := v.(*RoutersWrapper); ok {
			routers, err := NewRouteMatcher(routerConfig)
			if err != nil {
				log.DefaultLogger.Errorf("AddOrUpdateRouters, update router:%s error: %v", routerConfig.RouterConfigName, err)
				return err
			}
			log.DefaultLogger.Debugf("AddOrUpdateRouters, update router:%s success", routerConfig.RouterConfigName)
			primaryRouters.mux.Lock()
			primaryRouters.routers = routers
			primaryRouters.routersConfig = routerConfig
			primaryRouters.mux.Unlock()
		}
	} else {
		// try to create a new router
		// if a routerConfig with no routes, it is a valid config
		routers, err := NewRouteMatcher(routerConfig)
		log.DefaultLogger.Debugf("AddOrUpdateRouters, add router %s error: %v", routerConfig.RouterConfigName, err)
		// we may stored a nil routers
		// used in istio "RDS" mode
		rm.routersMap.Store(routerConfig.RouterConfigName, &RoutersWrapper{
			routers:       routers,
			routersConfig: routerConfig,
		})
	}
	admin.SetRouter(routerConfig.RouterConfigName, *routerConfig)

	return nil
}

// AddOrUpdateRouters used to add or update router
func (rm *routersManager) GetRouterWrapperByName(routerConfigName string) types.RouterWrapper {

	if value, ok := rm.routersMap.Load(routerConfigName); ok {
		if routerWrapper, ok := value.(*RoutersWrapper); ok {
			return routerWrapper
		}
	}

	return nil
}

func (rm *routersManager) AddRoute(routerConfigName, domain string, route *v2.Router) error {
	if v, ok := rm.routersMap.Load(routerConfigName); ok {
		if primaryRouters, ok := v.(*RoutersWrapper); ok {
			primaryRouters.mux.Lock()
			defer primaryRouters.mux.Unlock()
			routers := primaryRouters.routers
			// Stored primaryRouters should not be nil
			if routers == nil {
				return errors.New("no routers inited")
			}
			cfg := primaryRouters.routersConfig
			index := routers.AddRoute(domain, route)
			if index == -1 {
				log.DefaultLogger.Errorf("add route failed")
				return errors.New("add route failed")
			}
			// modify config
			routersCfg := cfg.VirtualHosts[index].Routers
			routersCfg = append(routersCfg, *route)
			cfg.VirtualHosts[index].Routers = routersCfg
			primaryRouters.routersConfig = cfg
			admin.SetRouter(routerConfigName, *cfg)
		}
	}
	return nil
}

func (rm *routersManager) RemoveAllRoutes(routerConfigName, domain string) error {
	if v, ok := rm.routersMap.Load(routerConfigName); ok {
		if primaryRouters, ok := v.(*RoutersWrapper); ok {
			primaryRouters.mux.Lock()
			defer primaryRouters.mux.Unlock()
			routers := primaryRouters.routers
			// Stored primaryRouters should not be nil
			if routers == nil {
				return errors.New("no routers inited")
			}
			cfg := primaryRouters.routersConfig
			index := routers.RemoveAllRoutes(domain)
			if index == -1 {
				log.DefaultLogger.Errorf("remove all route failed")
				return errors.New("remove all route failed")
			}
			// modify config
			routersCfg := cfg.VirtualHosts[index].Routers
			cfg.VirtualHosts[index].Routers = routersCfg[:0]
			primaryRouters.routersConfig = cfg
			admin.SetRouter(routerConfigName, *cfg)
		}
	}
	return nil
}
