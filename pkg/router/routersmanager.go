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
	"sync"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var instanceMutex = sync.Mutex{}
var routersMangerInstance *routersManager

// GetClusterMngAdapterInstance used to get clusterMngAdapterInstance
func GetRoutersMangerInstance() *routersManager {
	return routersMangerInstance
}

type routersManager struct {
	routersMap sync.Map // key is router's name used to associated to the listener, value's type is: type.Routers
}

type RoutersWrapper struct {
	routers types.Routers
}

func (rw *RoutersWrapper) GetRouters() types.Routers {
	return rw.routers
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

	if routers, err := NewRouteMatcher(routerConfig); err != nil {
		log.DefaultLogger.Errorf("AddOrUpdateRouters Error,  When NewRouteMatcher:", err.Error())
		return err
	} else {
		if v, ok := rm.routersMap.Load(routerConfig.RouterConfigName); ok {
			// update a router
			if primaryRouters, ok := v.(*RoutersWrapper); ok {
				primaryRouters.routers = routers
			}

		} else {
			// new a router
			rm.routersMap.Store(routerConfig.RouterConfigName, &RoutersWrapper{
				routers: routers,
			})
		}
	}

	return nil
}

// AddOrUpdateRouters used to add or update router
func (rm *routersManager) GetRouterByListenerName(routerConfigName string) types.RouterWapper {

	if value, ok := rm.routersMap.Load(routerConfigName); ok {
		if router, ok := value.(*RoutersWrapper); ok {
			return router
		}
	}

	return nil
}
